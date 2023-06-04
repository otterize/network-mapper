package main

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/shared/kubefinder"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/otterize/network-mapper/src/sniffer/pkg/sniffer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

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

	go func() {
		logrus.Info("Starting operator manager")
		if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
			logrus.Error(err, "unable to run manager")
			os.Exit(1)
		}
	}()

	mapperClient := mapperclient.NewMapperClient(viper.GetString(sharedconfig.MapperApiUrlKey))
	kubeFinder, err := kubefinder.NewKubeFinder(mgr)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	serviceIdResolver := serviceidresolver.NewResolver(mgr.GetClient())
	ipResolver := ipresolver.NewIpResolver(kubeFinder, serviceIdResolver)
	snifferInstance := sniffer.NewSniffer(mapperClient, ipResolver)
	err = snifferInstance.RunForever(context.Background())
	if err != nil {
		panic(err)
	}
}
