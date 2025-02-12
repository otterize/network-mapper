package main

import (
	"context"
	"fmt"
	"github.com/bombsimon/logrusr/v3"
	"github.com/labstack/echo-contrib/echoprometheus"
	otterizev2alpha1 "github.com/otterize/intents-operator/src/operator/api/v2alpha1"
	"github.com/otterize/intents-operator/src/shared"
	"github.com/otterize/intents-operator/src/shared/clusterutils"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/telemetries/componentinfo"
	"github.com/otterize/intents-operator/src/shared/telemetries/errorreporter"
	istiowatcher "github.com/otterize/network-mapper/src/istio-watcher/pkg/watcher"
	"github.com/otterize/network-mapper/src/mapper/pkg/awsintentsholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/azureintentsholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/collectors/traffic"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnscache"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnsintentspublisher"
	"github.com/otterize/network-mapper/src/mapper/pkg/externaltrafficholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/incomingtrafficholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/resourcevisibility"
	"github.com/otterize/network-mapper/src/shared/echologrus"
	"golang.org/x/sync/errgroup"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	otterizev1alpha2 "github.com/otterize/intents-operator/src/operator/api/v1alpha2"
	otterizev1alpha3 "github.com/otterize/intents-operator/src/operator/api/v1alpha3"
	otterizev1beta1 "github.com/otterize/intents-operator/src/operator/api/v1beta1"
	otterizev2beta1 "github.com/otterize/intents-operator/src/operator/api/v2beta1"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetriesgql"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetrysender"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/clouduploader"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/mapper/pkg/metricexporter"
	"github.com/otterize/network-mapper/src/mapper/pkg/resolvers"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/otterize/network-mapper/src/shared/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(otterizev1alpha2.AddToScheme(scheme))
	utilruntime.Must(otterizev1alpha3.AddToScheme(scheme))
	utilruntime.Must(otterizev1beta1.AddToScheme(scheme))
	utilruntime.Must(otterizev2alpha1.AddToScheme(scheme))
	utilruntime.Must(otterizev2beta1.AddToScheme(scheme))
}

func getClusterDomainOrDefault() string {
	resolvedClusterDomain, err := kubeutils.GetClusterDomain()
	if err != nil {
		logrus.WithError(err).Warningf("Could not deduce the cluster domain. Operating on the default domain %s", kubeutils.DefaultClusterDomain)
		return kubeutils.DefaultClusterDomain
	} else {
		logrus.Info("Detected cluster domain: ", resolvedClusterDomain)
		return resolvedClusterDomain
	}
}

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	signalHandlerCtx := ctrl.SetupSignalHandler()

	clusterUID := clusterutils.GetOrCreateClusterUID(signalHandlerCtx)

	componentinfo.SetGlobalContextId(telemetrysender.Anonymize(clusterUID))

	errorreporter.Init(telemetriesgql.TelemetryComponentTypeNetworkMapper, version.Version())
	defer errorreporter.AutoNotify()
	shared.RegisterPanicHandlers()

	if !viper.IsSet(config.ClusterDomainKey) || viper.GetString(config.ClusterDomainKey) == "" {
		clusterDomain := getClusterDomainOrDefault()
		viper.Set(config.ClusterDomainKey, clusterDomain)
	}

	ctrl.SetLogger(logrusr.New(logrus.StandardLogger()))
	echologrus.Logger = logrus.StandardLogger()

	// start manager with operators
	options := manager.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
	}
	mgr, err := manager.New(clientconfig.GetConfigOrDie(), options)
	if err != nil {
		logrus.Panicf("unable to set up overall controller manager: %s", err)
	}

	errgrp, errGroupCtx := errgroup.WithContext(signalHandlerCtx)

	dnsCache := dnscache.NewDNSCache()
	dnsPublisher, dnsPublisherEnabled, err := dnsintentspublisher.InitWithManager(errGroupCtx, mgr, dnsCache)
	if err != nil {
		logrus.WithError(err).Panic("Failed to initialize DNS publisher")
	}

	mapperServer := echo.New()
	mapperServer.HideBanner = true

	kubeFinder, err := kubefinder.NewKubeFinder(errGroupCtx, mgr)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()

		logrus.Info("Starting operator manager")

		if err := mgr.Start(errGroupCtx); err != nil {
			logrus.WithError(err).Error("unable to run manager")
			return errors.Wrap(err)
		}

		return nil
	})

	// start API server
	mapperServer.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	mapperServer.Logger = echologrus.GetEchoLogger()
	mapperServer.Use(echologrus.Hook())
	mapperServer.Use(middleware.CORS())
	mapperServer.Use(middleware.RemoveTrailingSlash())
	initCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()
	mgr.GetCache().WaitForCacheSync(initCtx) // needed to let the manager initialize before used in intentsHolder

	intentsHolder := intentsstore.NewIntentsHolder()
	externalTrafficIntentsHolder := externaltrafficholder.NewExternalTrafficIntentsHolder()
	incomingTrafficIntentsHolder := incomingtrafficholder.NewIncomingTrafficIntentsHolder()
	awsIntentsHolder := awsintentsholder.New()
	azureIntentsHolder := azureintentsholder.New()
	trafficCollector := traffic.NewCollector()

	resolver := resolvers.NewResolver(
		kubeFinder,
		serviceidresolver.NewResolver(mgr.GetClient()),
		intentsHolder,
		externalTrafficIntentsHolder,
		awsIntentsHolder,
		azureIntentsHolder,
		dnsCache,
		incomingTrafficIntentsHolder,
		trafficCollector,
	)
	resolver.Register(mapperServer)

	metricsServer := echo.New()
	metricsServer.HideBanner = true
	metricsServer.GET("/metrics", echoprometheus.NewHandler())

	if viper.GetBool(config.EnableIstioCollectionKey) {
		istioWatcher, err := istiowatcher.NewWatcher(resolver.Mutation())
		if err != nil {
			logrus.WithError(err).Panic("failed to initialize istio watcher")
		}

		errgrp.Go(func() error {
			defer errorreporter.AutoNotify()
			return istioWatcher.RunForever(errGroupCtx)
		})
	}

	cloudUploaderConfig := clouduploader.ConfigFromViper()
	cloudClient, cloudEnabled, err := cloudclient.NewClient(errGroupCtx)
	if err != nil {
		logrus.WithError(err).Panic("Failed to initialize cloud client")
	}
	if cloudEnabled {
		cloudUploader := clouduploader.NewCloudUploader(intentsHolder, cloudUploaderConfig, cloudClient)

		intentsHolder.RegisterNotifyIntents(cloudUploader.NotifyIntents)
		if viper.GetBool(config.ExternalTrafficCaptureEnabledKey) {
			externalTrafficIntentsHolder.RegisterNotifyIntents(cloudUploader.NotifyExternalTrafficIntents)
			incomingTrafficIntentsHolder.RegisterNotifyIntents(cloudUploader.NotifyIncomingTrafficIntents)
		}
		awsIntentsHolder.RegisterNotifyIntents(cloudUploader.NotifyAWSIntents)
		azureIntentsHolder.RegisterNotifyIntents(cloudUploader.NotifyAzureIntents)
		trafficCollector.RegisterNotifyTraffic(cloudUploader.NotifyTrafficLevels)

		go cloudUploader.PeriodicStatusReport(errGroupCtx)

		ingressReconciler := resourcevisibility.NewIngressReconciler(mgr.GetClient(), cloudClient)
		if err := ingressReconciler.SetupWithManager(mgr); err != nil {
			logrus.WithError(err).Panic("unable to create ingress reconciler")
		}

		serviceReconciler := resourcevisibility.NewServiceReconciler(mgr.GetClient(), cloudClient, kubeFinder)
		if err := serviceReconciler.SetupWithManager(mgr); err != nil {
			logrus.WithError(err).Panic("unable to create service reconciler")
		}
	}

	if viper.GetBool(config.OTelEnabledKey) {
		otelExporter, err := metricexporter.NewMetricExporter(errGroupCtx)
		if err != nil {
			logrus.WithError(err).Panic("Failed to initialize otel exporter")
		}
		intentsHolder.RegisterNotifyIntents(otelExporter.NotifyIntents)
	}

	if dnsPublisherEnabled {
		errgrp.Go(func() error {
			defer errorreporter.AutoNotify()
			dnsPublisher.RunForever(errGroupCtx)
			return nil
		})
	}

	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		intentsHolder.PeriodicIntentsUpload(errGroupCtx, cloudUploaderConfig.UploadInterval)
		return nil
	})

	if viper.GetBool(config.ExternalTrafficCaptureEnabledKey) {
		errgrp.Go(func() error {
			defer errorreporter.AutoNotify()
			externalTrafficIntentsHolder.PeriodicIntentsUpload(errGroupCtx, cloudUploaderConfig.UploadInterval)
			return nil
		})
		errgrp.Go(func() error {
			defer errorreporter.AutoNotify()
			logrus.Info("Starting incoming traffic intents uploader")
			incomingTrafficIntentsHolder.PeriodicIntentsUpload(errGroupCtx, cloudUploaderConfig.UploadInterval)
			return nil
		})
	}

	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		awsIntentsHolder.PeriodicIntentsUpload(errGroupCtx, cloudUploaderConfig.UploadInterval)
		return nil
	})
	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		azureIntentsHolder.PeriodicIntentsUpload(errGroupCtx, cloudUploaderConfig.UploadInterval)
		return nil
	})
	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		trafficCollector.PeriodicUpload(errGroupCtx, cloudUploaderConfig.UploadInterval)
		return nil
	})

	telemetrysender.SendNetworkMapper(telemetriesgql.EventTypeStarted, 1)
	telemetrysender.NetworkMapperRunActiveReporter(errGroupCtx)

	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		go shutdownGracefullyOnCancel(errGroupCtx, metricsServer)

		return metricsServer.Start(fmt.Sprintf(":%d", viper.GetInt(sharedconfig.PrometheusMetricsPortKey)))
	})
	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		go shutdownGracefullyOnCancel(errGroupCtx, mapperServer)

		return mapperServer.Start(":9090")
	})
	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		return resolver.RunForever(errGroupCtx)
	})

	err = errgrp.Wait()
	logrus.Infof("Network Mapper stopped")

	if err != nil {
		if !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, context.Canceled) {
			logrus.WithError(err).Fatal("failed to shutdown server")
		}
	}
}

func shutdownGracefullyOnCancel(errGroupCtx context.Context, server *echo.Echo) {
	<-errGroupCtx.Done()
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownErr := server.Shutdown(timeoutCtx)

	if shutdownErr != nil {
		logrus.WithError(shutdownErr).Error("failed to shutdown server")
		_ = server.Close()

	}
}
