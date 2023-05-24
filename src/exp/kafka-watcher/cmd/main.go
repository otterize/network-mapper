package main

import (
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/logwatcher"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/mapperclient"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	mapperClient := mapperclient.NewMapperClient(viper.GetString(sharedconfig.MapperApiUrlKey))
	logPath := viper.GetString(sharedconfig.AuthZLogPath)

	w, err := logwatcher.NewWatcher(
		mapperClient,
		logPath,
	)

	if err != nil {
		logrus.WithError(err).Panic()
	}

	serverName := types.NamespacedName{
		Namespace: viper.GetString(sharedconfig.EnvNamespaceKey),
		Name:      viper.GetString(sharedconfig.EnvPodKey),
	}

	sigHandlerCtx := signals.SetupSignalHandler()

	if err := w.RunForever(sigHandlerCtx, serverName); err != nil {
		logrus.WithError(err).Panic()
	}
}
