package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/metadata"
	"sigs.k8s.io/controller-runtime"
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
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if !viper.IsSet(config.ClusterDomainKey) || viper.GetString(config.ClusterDomainKey) == "" {
		clusterDomain := getClusterDomainOrDefault()
		viper.Set(config.ClusterDomainKey, clusterDomain)
	}

	// start manager with operators
	mgr, err := manager.New(clientconfig.GetConfigOrDie(), manager.Options{MetricsBindAddress: "0"})
	if err != nil {
		logrus.Errorf("unable to set up overall controller manager: %s", err)
		os.Exit(1)
	}

	stopCtx := signals.SetupSignalHandler()

	kubeFinder, err := kubefinder.NewKubeFinder(stopCtx, mgr)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	go func() {
		logrus.Info("Starting operator manager")
		if err := mgr.Start(stopCtx); err != nil {
			logrus.Error(err, "unable to run manager")
			os.Exit(1)

		}
	}()

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
	telemetrysender.SetGlobalContextId(telemetrysender.Anonymize(kubeSystemUID))

	// start API server
	e := echo.New()
	e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())
	e.Use(middleware.RemoveTrailingSlash())
	initCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()
	mgr.GetCache().WaitForCacheSync(initCtx) // needed to let the manager initialize before used in intentsHolder

	intentsHolder := intentsstore.NewIntentsHolder()

	resolver := resolvers.NewResolver(kubeFinder, serviceidresolver.NewResolver(mgr.GetClient()), intentsHolder)
	resolver.Register(e)

	signalHandlerCtx := controllerruntime.SetupSignalHandler()
	cloudClient, cloudEnabled, err := cloudclient.NewClient(signalHandlerCtx)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize cloud client")
	}

	cloudUploaderConfig := clouduploader.ConfigFromViper()
	if cloudEnabled {
		cloudUploader := clouduploader.NewCloudUploader(intentsHolder, cloudUploaderConfig, cloudClient)
		intentsHolder.RegisterGetCallback(cloudUploader.GetIntentCallback)
		go cloudUploader.PeriodicStatusReport(signalHandlerCtx)
	}

	if viper.GetBool(config.OTelEnabledKey) {
		otelExporter, err := metricexporter.NewMetricExporter(signalHandlerCtx)
		intentsHolder.RegisterGetCallback(otelExporter.GetIntentCallback)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to initialize otel exporter")
		}
	}

	go intentsHolder.PeriodicIntentsUpload(signalHandlerCtx, cloudUploaderConfig.UploadInterval)

	telemetrysender.SetGlobalVersion(version.Version())
	telemetrysender.SendNetworkMapper(telemetriesgql.EventTypeStarted, 1)
	telemetrysender.NetworkMapperRunActiveReporter(signalHandlerCtx)

	logrus.Info("Starting api server")
	err = e.Start("0.0.0.0:9090")
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

}
