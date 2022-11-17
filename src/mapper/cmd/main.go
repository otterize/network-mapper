package main

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/clouduploader"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/mapper/pkg/resolvers"
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"os/signal"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"syscall"
	"time"
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
	if viper.GetBool(config.DebugKey) {
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

	kubeFinder, err := kubefinder.NewKubeFinder(mgr)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	go func() {
		logrus.Info("Starting operator manager")
		if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
			logrus.Error(err, "unable to run manager")
			os.Exit(1)

		}
	}()

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

	intentHolderCfg, err := resolvers.IntentsHolderConfigFromViper()
	if err != nil {
		logrus.Error(intentHolderCfg)
		os.Exit(1)
	}

	cloudConfig := clouduploader.ConfigFromViper()
	intentsHolder := resolvers.NewIntentsHolder(mgr.GetClient(), intentHolderCfg)
	cloudClient := clouduploader.NewCloudUploader(intentsHolder, cloudConfig, cloudclient.NewClient)
	resolver := resolvers.NewResolver(kubeFinder, serviceidresolver.NewResolver(mgr.GetClient()), intentsHolder)
	_ = resolver.LoadStore(initCtx) // loads the store from the previous run
	resolver.Register(e)

	// Temporary condition to disable cloud upload until K8s integration is ready
	if cloudConfig.Environment != "" {
		go func() {
			cloudClientCtx, cloudClientCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cloudClientCancel()
			cloudClient.PeriodicIntentsUpload(cloudClientCtx)
		}()
	}

	logrus.Info("Starting api server")
	err = e.Start("0.0.0.0:9090")
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

}
