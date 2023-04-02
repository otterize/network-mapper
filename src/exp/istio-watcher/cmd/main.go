package main

import (
	"context"
	"github.com/otterize/network-mapper/src/exp/istio-watcher/config"
	"github.com/otterize/network-mapper/src/exp/istio-watcher/mapperclient"
	"github.com/otterize/network-mapper/src/exp/istio-watcher/pkg/watcher"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	if viper.GetBool(config.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	mapperClient := mapperclient.NewMapperClient(viper.GetString(config.MapperApiUrlKey))
	istioWatcher, err := istiowatcher.NewWatcher(mapperClient)
	if err != nil {
		logrus.WithError(err).Panic()
	}

	if err := istioWatcher.RunForever(context.Background()); err != nil {
		logrus.WithError(err).Panic()
	}
}
