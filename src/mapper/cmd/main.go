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
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/metadata"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		logrus.Errorf("unable to set up overall controller manager: %s", err)
		os.Exit(1)
	}

	errgrp, errGroupCtx := errgroup.WithContext(signals.SetupSignalHandler())
	kubeFinder, err := kubefinder.NewKubeFinder(errGroupCtx, mgr)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
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
		logrus.WithError(err).Fatal("unable to create metadata client")
	}
	mapping, err := mgr.GetRESTMapper().RESTMapping(schema.GroupKind{Group: "", Kind: "Namespace"}, "v1")
	if err != nil {
		logrus.WithError(err).Fatal("unable to create Kubernetes API REST mapping")
	}
	kubeSystemUID := ""
	kubeSystemNs, err := metadataClient.Resource(mapping.Resource).Get(context.Background(), "kube-system", metav1.GetOptions{})
	if err != nil || kubeSystemNs == nil {
		logrus.Warningf("failed getting kubesystem UID: %s", err)
		kubeSystemUID = fmt.Sprintf("rand-%s", uuid.New().String())
	} else {
		kubeSystemUID = string(kubeSystemNs.UID)
	}
	contextID := telemetrysender.Anonymize(kubeSystemUID)
	componentinfo.SetGlobalContextId(contextID)
	err = WriteContextIDToConfigMap(mgr.GetClient(), contextID)
	if err != nil {
		logrus.WithError(err).Error("unable to write context id to config map")
	}

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
		logrus.WithError(err).Fatal("Failed to initialize cloud client")
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
			logrus.WithError(err).Fatal("Failed to initialize otel exporter")
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

	err = errgrp.Wait()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logrus.WithError(err).Fatal("Error when running server or HTTP server")
	}

}

func WriteContextIDToConfigMap(k8sClient client.Client, contextID string) error {
	namespace, err := kubeutils.GetCurrentNamespace()
	if err != nil {
		return err
	}

	configMapName := viper.GetString(sharedconfig.ComponentMetadataConfigmapNameKey)
	var configMap corev1.ConfigMap
	err = k8sClient.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: configMapName}, &configMap)
	if err != nil {
		return err
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	contextIdKey := viper.GetString(sharedconfig.ComponentMetadataContextIdKeyKey)
	configMap.Data[contextIdKey] = contextID
	err = k8sClient.Update(context.Background(), &configMap)
	if err != nil {
		return err
	}

	logrus.Info("Successfully wrote context id to config map")
	return nil
}
