package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/bombsimon/logrusr/v3"
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/neko-neko/echo-logrus/v2/log"
	"github.com/otterize/intents-operator/src/shared/telemetries/componentinfo"
	"github.com/otterize/intents-operator/src/shared/telemetries/errorreporter"
	"github.com/otterize/network-mapper/src/mapper/pkg/externaltrafficholder"
	"github.com/otterize/network-mapper/src/shared/componentutils"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/metadata"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	ctrl "sigs.k8s.io/controller-runtime"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

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
	errorreporter.Init("network-mapper", version.Version(), viper.GetString(sharedconfig.TelemetryErrorsAPIKeyKey))
	defer bugsnag.AutoNotify(context.Background())
	if !viper.IsSet(config.ClusterDomainKey) || viper.GetString(config.ClusterDomainKey) == "" {
		clusterDomain := getClusterDomainOrDefault()
		viper.Set(config.ClusterDomainKey, clusterDomain)
	}
	mapperServer := echo.New()

	logrus.SetLevel(logrus.InfoLevel)
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	ctrl.SetLogger(logrusr.New(logrus.StandardLogger()))
	log.Logger().Logger = logrus.StandardLogger()
	mapperServer.Use(middleware.Logger())

	// start manager with operators
	mgr, err := manager.New(clientconfig.GetConfigOrDie(), manager.Options{MetricsBindAddress: "0"})
	if err != nil {
		componentutils.ExitDueToInitFailure(logrus.WithError(err), "Unable to set up overall controller manager")
	}

	errgrp, errGroupCtx := errgroup.WithContext(signals.SetupSignalHandler())
	kubeFinder, err := kubefinder.NewKubeFinder(errGroupCtx, mgr)
	if err != nil {
		componentutils.ExitDueToInitFailure(logrus.WithError(err), "Failed to initialize kube finder")
	}

	errgrp.Go(func() error {
		defer bugsnag.AutoNotify(errGroupCtx)
		logrus.Info("Starting operator manager")
		if err := mgr.Start(errGroupCtx); err != nil {
			logrus.Error(err, "unable to run manager")
			return err
		}
		return nil
	})

	metadataClient, err := metadata.NewForConfig(clientconfig.GetConfigOrDie())
	if err != nil {
		componentutils.ExitDueToInitFailure(logrus.WithError(err), "Failed to initialize metadata client")
	}
	mapping, err := mgr.GetRESTMapper().RESTMapping(schema.GroupKind{Group: "", Kind: "Namespace"}, "v1")
	if err != nil {
		componentutils.ExitDueToInitFailure(logrus.WithError(err), "unable to create Kubernetes API REST mapping")
	}
	kubeSystemUID := ""
	kubeSystemNs, err := metadataClient.Resource(mapping.Resource).Get(context.Background(), "kube-system", metav1.GetOptions{})
	if err != nil || kubeSystemNs == nil {
		logrus.Warningf("failed getting kubesystem UID: %s", err)
		kubeSystemUID = fmt.Sprintf("rand-%s", uuid.New().String())
	} else {
		kubeSystemUID = string(kubeSystemNs.UID)
	}
	componentinfo.SetGlobalContextId(telemetrysender.Anonymize(kubeSystemUID))

	// start API server
	mapperServer.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	mapperServer.Use(middleware.Logger())
	mapperServer.Use(middleware.CORS())
	mapperServer.Use(middleware.RemoveTrailingSlash())
	initCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()
	mgr.GetCache().WaitForCacheSync(initCtx) // needed to let the manager initialize before used in intentsHolder

	intentsHolder := intentsstore.NewIntentsHolder()
	externalTrafficIntentsHolder := externaltrafficholder.NewExternalTrafficIntentsHolder()

	resolver := resolvers.NewResolver(kubeFinder, serviceidresolver.NewResolver(mgr.GetClient()), intentsHolder, externalTrafficIntentsHolder)
	resolver.Register(mapperServer)

	metricsServer := echo.New()
	metricsServer.GET("/metrics", echoprometheus.NewHandler())

	cloudClient, cloudEnabled, err := cloudclient.NewClient(errGroupCtx)
	if err != nil {
		componentutils.ExitDueToInitFailure(logrus.WithError(err), "Failed to initialize cloud client")
	}

	cloudUploaderConfig := clouduploader.ConfigFromViper()
	if cloudEnabled {
		cloudUploader := clouduploader.NewCloudUploader(intentsHolder, cloudUploaderConfig, cloudClient)
		intentsHolder.RegisterNotifyIntents(cloudUploader.NotifyIntents)
		if viper.GetBool(config.ExternalTrafficCaptureEnabledKey) {
			externalTrafficIntentsHolder.RegisterNotifyIntents(cloudUploader.NotifyExternalTrafficIntents)
		}
		go cloudUploader.PeriodicStatusReport(errGroupCtx)
	}

	if viper.GetBool(config.OTelEnabledKey) {
		otelExporter, err := metricexporter.NewMetricExporter(errGroupCtx)
		intentsHolder.RegisterNotifyIntents(otelExporter.NotifyIntents)
		if err != nil {
			componentutils.ExitDueToInitFailure(logrus.WithError(err), "Failed to initialize otel exporter")
		}
	}

	errgrp.Go(func() error {
		defer bugsnag.AutoNotify(errGroupCtx)
		intentsHolder.PeriodicIntentsUpload(errGroupCtx, cloudUploaderConfig.UploadInterval)
		return nil
	})
	if viper.GetBool(config.ExternalTrafficCaptureEnabledKey) {
		errgrp.Go(func() error {
			defer bugsnag.AutoNotify(errGroupCtx)
			externalTrafficIntentsHolder.PeriodicIntentsUpload(errGroupCtx, cloudUploaderConfig.UploadInterval)
			return nil
		})
	}

	componentinfo.SetGlobalVersion(version.Version())
	telemetrysender.SendNetworkMapper(telemetriesgql.EventTypeStarted, 1)
	telemetrysender.NetworkMapperRunActiveReporter(errGroupCtx)

	errgrp.Go(func() error {
		defer bugsnag.AutoNotify(errGroupCtx)
		return metricsServer.Start(fmt.Sprintf(":%d", viper.GetInt(sharedconfig.PrometheusMetricsPortKey)))
	})
	errgrp.Go(func() error {
		defer bugsnag.AutoNotify(errGroupCtx)
		return mapperServer.Start(":9090")
	})
	errgrp.Go(func() error {
		defer bugsnag.AutoNotify(errGroupCtx)
		return resolver.RunForever(errGroupCtx)
	})

	err = errgrp.Wait()
	if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, context.Canceled) {
		componentutils.ExitDueToInitFailure(logrus.WithError(err), "Error when running server or HTTP server")
	}

}
