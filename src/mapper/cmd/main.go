package main

import (
	"context"
	"fmt"
	"github.com/bombsimon/logrusr/v3"
	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/echoprometheus"
	operatorwebhooks "github.com/otterize/intents-operator/src/operator/webhooks"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/telemetries/componentinfo"
	"github.com/otterize/intents-operator/src/shared/telemetries/errorreporter"
	"github.com/otterize/network-mapper/src/mapper/pkg/awsintentsholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnscache"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnsintentspublisher"
	"github.com/otterize/network-mapper/src/mapper/pkg/externaltrafficholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/pod_webhook"
	"github.com/otterize/network-mapper/src/shared/echologrus"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/metadata"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	otterizev1alpha2 "github.com/otterize/intents-operator/src/operator/api/v1alpha2"
	otterizev1alpha3 "github.com/otterize/intents-operator/src/operator/api/v1alpha3"
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
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(otterizev1alpha2.AddToScheme(scheme))
	utilruntime.Must(otterizev1alpha3.AddToScheme(scheme))
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
	echologrus.Logger = logrus.StandardLogger()
	mapperServer.Use(middleware.Logger())

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

	errgrp, errGroupCtx := errgroup.WithContext(signals.SetupSignalHandler())

	dnsCache := dnscache.NewDNSCache()
	dnsPublisher := dnsintentspublisher.NewPublisher(mgr.GetClient(), dnsCache)
	err = dnsPublisher.InitIndices(errGroupCtx, mgr)
	if err != nil {
		logrus.WithError(err).Panic("unable to initialize DNS publisher")
	}
	kubeFinder, err := kubefinder.NewKubeFinder(errGroupCtx, mgr)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	if viper.GetBool(config.CreateWebhookCertificateKey) {
		// create webhook server certificate
		logrus.Infoln("Creating self signing certs")
		podNamespace, err := kubeutils.GetCurrentNamespace()

		if err != nil {
			logrus.WithError(err).Panic("unable to get pod namespace")
		}

		certBundle, err :=
			operatorwebhooks.GenerateSelfSignedCertificate("otterize-network-mapper-webhook-service", podNamespace)
		if err != nil {
			logrus.WithError(err).Panic("unable to create self signed certs for webhook")
		}
		err = operatorwebhooks.WriteCertToFiles(certBundle)
		if err != nil {
			logrus.WithError(err).Panic("failed writing certs to file system")
		}

		err = operatorwebhooks.UpdateMutationWebHookCA(context.Background(),
			"otterize-aws-visibility-mutating-webhook-configuration", certBundle.CertPem)
		if err != nil {
			logrus.WithError(err).Panic("updating validation webhook certificate failed")
		}
	}

	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		logrus.Info("Starting operator manager")
		if err := mgr.Start(errGroupCtx); err != nil {
			logrus.Error(err, "unable to run manager")
			return errors.Wrap(err)
		}
		return nil
	})

	metadataClient, err := metadata.NewForConfig(clientconfig.GetConfigOrDie())
	if err != nil {
		logrus.WithError(err).Panic("unable to create metadata client")
	}
	mapping, err := mgr.GetRESTMapper().RESTMapping(schema.GroupKind{Group: "", Kind: "Namespace"}, "v1")
	if err != nil {
		logrus.WithError(err).Panic("unable to create Kubernetes API REST mapping")
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
	mapperServer.Logger = echologrus.GetEchoLogger()
	mapperServer.Use(echologrus.Hook())
	mapperServer.Use(middleware.CORS())
	mapperServer.Use(middleware.RemoveTrailingSlash())
	initCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()
	mgr.GetCache().WaitForCacheSync(initCtx) // needed to let the manager initialize before used in intentsHolder

	if viper.GetBool(config.EnableAWSVisibilityKeyWebHook) {
		webhookHandler, err := pod_webhook.NewInjectDNSConfigToPodWebhook(
			mgr.GetClient(),
			admission.NewDecoder(mgr.GetScheme()),
		)

		if err != nil {
			logrus.WithError(err).Panic("unable to create webhook handler")
		}

		mgr.GetWebhookServer().Register(
			"/mutate-v1-pod",
			&webhook.Admission{
				Handler: webhookHandler,
			},
		)
	}

	intentsHolder := intentsstore.NewIntentsHolder()
	externalTrafficIntentsHolder := externaltrafficholder.NewExternalTrafficIntentsHolder()
	awsIntentsHolder := awsintentsholder.New()

	resolver := resolvers.NewResolver(kubeFinder, serviceidresolver.NewResolver(mgr.GetClient()), intentsHolder, externalTrafficIntentsHolder, awsIntentsHolder, dnsCache)
	resolver.Register(mapperServer)

	metricsServer := echo.New()
	metricsServer.GET("/metrics", echoprometheus.NewHandler())

	cloudClient, cloudEnabled, err := cloudclient.NewClient(errGroupCtx)
	if err != nil {
		logrus.WithError(err).Panic("Failed to initialize cloud client")
	}

	cloudUploaderConfig := clouduploader.ConfigFromViper()
	if cloudEnabled {
		cloudUploader := clouduploader.NewCloudUploader(intentsHolder, cloudUploaderConfig, cloudClient)

		intentsHolder.RegisterNotifyIntents(cloudUploader.NotifyIntents)
		if viper.GetBool(config.ExternalTrafficCaptureEnabledKey) {
			externalTrafficIntentsHolder.RegisterNotifyIntents(cloudUploader.NotifyExternalTrafficIntents)
		}
		awsIntentsHolder.RegisterNotifyIntents(cloudUploader.NotifyAWSIntents)

		go cloudUploader.PeriodicStatusReport(errGroupCtx)
	}

	if viper.GetBool(config.OTelEnabledKey) {
		otelExporter, err := metricexporter.NewMetricExporter(errGroupCtx)
		if err != nil {
			logrus.WithError(err).Panic("Failed to initialize otel exporter")
		}
		intentsHolder.RegisterNotifyIntents(otelExporter.NotifyIntents)
	}

	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		dnsPublisher.RunForever(errGroupCtx)
		return nil
	})

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
	}

	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		awsIntentsHolder.PeriodicIntentsUpload(errGroupCtx, cloudUploaderConfig.UploadInterval)
		return nil
	})

	componentinfo.SetGlobalVersion(version.Version())
	telemetrysender.SendNetworkMapper(telemetriesgql.EventTypeStarted, 1)
	telemetrysender.NetworkMapperRunActiveReporter(errGroupCtx)

	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		return metricsServer.Start(fmt.Sprintf(":%d", viper.GetInt(sharedconfig.PrometheusMetricsPortKey)))
	})
	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		return mapperServer.Start(":9090")
	})

	err = errgrp.Wait()
	if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, context.Canceled) {
		logrus.WithError(err).Panic("Error when running server or HTTP server")
	}

}
