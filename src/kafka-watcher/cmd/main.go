package main

import (
	"context"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/client"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/config"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/logwatcher"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	if viper.GetBool(config.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if !viper.IsSet(config.KafkaNameKey) || !viper.IsSet(config.KafkaNamespaceKey) {
		logrus.Panic("Kafka pod name and namespace must be specified")
	}

	mapperClient := client.NewMapperClient(viper.GetString(config.MapperApiUrlKey))
	w, err := logwatcher.NewWatcher(
		mapperClient,
		viper.GetString(config.KafkaNameKey),
		viper.GetString(config.KafkaNamespaceKey),
	)
	if err != nil {
		panic(err)
	}

	if err := w.RunForever(context.Background()); err != nil {
		panic(err)
	}
}
