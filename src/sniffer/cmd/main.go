package main

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/shared/kubefinder"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/otterize/network-mapper/src/sniffer/pkg/sniffer"
	localresolution2 "github.com/otterize/network-mapper/src/sniffer/pkg/sniffer/exp/localresolution"
	"github.com/otterize/network-mapper/src/sniffer/pkg/socketscanner"
	"github.com/otterize/network-mapper/src/sniffer/pkg/socketscanner/exp/localresolution"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"os"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

type Runnable interface {
	RunForever(ctx context.Context) error
}

func main() {
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	mgr, err := manager.New(clientconfig.GetConfigOrDie(), manager.Options{MetricsBindAddress: "0"})
	if err != nil {
		logrus.Errorf("unable to set up overall controller manager: %s", err)
		os.Exit(1)
	}

	logrus.Info("Manager created")

	logrus.Info("Starting operator manager")

	errGroup, ctx := errgroup.WithContext(signals.SetupSignalHandler())
	errGroup.Go(func() error {
		return mgr.Start(ctx)
	})
	mgr.GetCache().WaitForCacheSync(ctx)

	mapperClient := mapperclient.NewMapperClient(viper.GetString(sharedconfig.MapperApiUrlKey))
	kubeFinder, err := kubefinder.NewKubeFinder(mgr)
	if err != nil {
		logrus.WithError(err).Fatal("unable to start kubefinder")
	}

	serviceIdResolver := serviceidresolver.NewResolver(mgr.GetClient())
	ipResolver := ipresolver.NewPodResolver(kubeFinder, serviceIdResolver)
	var socketScanner Runnable = socketscanner.NewSocketScanner(mapperClient)
	var snifferInstance Runnable = sniffer.NewSniffer(mapperClient)
	if viper.GetBool(config.SnifferResolveLocallyKey) {
		socketScanner = localresolution.NewSocketScanner(mapperClient, ipResolver)
		snifferInstance = localresolution2.NewSniffer(mapperClient, ipResolver)
	}
	errGroup.Go(func() error {
		return socketScanner.RunForever(ctx)
	})
	errGroup.Go(func() error {
		return snifferInstance.RunForever(ctx)

	})

	err = errGroup.Wait()
	if err != nil {
		logrus.WithError(err).Fatal("critical component exited or failed to start: %s", err.Error())
	}
}
