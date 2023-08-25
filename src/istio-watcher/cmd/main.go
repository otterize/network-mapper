package main

import (
	"github.com/otterize/network-mapper/src/istio-watcher/mapperclient"
	"github.com/otterize/network-mapper/src/istio-watcher/pkg/watcher"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	mapperClient := mapperclient.NewMapperClient(viper.GetString(sharedconfig.MapperApiUrlKey))
	istioWatcher, err := istiowatcher.istiowatcher.NewWatcher(mapperClient)
	if err != nil {
		logrus.WithError(err).Panic()
	}

	if err := istioWatcher.RunForever(signals.SetupSignalHandler()); err != nil {
		logrus.WithError(err).Panic()
	}
}
