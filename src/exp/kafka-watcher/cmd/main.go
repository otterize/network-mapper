package main

import (
	"errors"
	"fmt"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/config"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/logwatcher"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/mapperclient"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func parseKafkaServers(serverNames []string) ([]types.NamespacedName, error) {
	var servers []types.NamespacedName
	for _, serverName := range serverNames {
		nameParts := strings.Split(serverName, ".")
		if len(nameParts) != 2 {
			return nil, fmt.Errorf("error parsing server pod name %s - should be formatted as 'name.namespace'", serverName)
		}
		servers = append(servers, types.NamespacedName{
			Name:      nameParts[0],
			Namespace: nameParts[1],
		})
	}
	return servers, nil
}

func main() {
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	kafkaServers, err := parseKafkaServers(viper.GetStringSlice(config.KafkaServersKey))
	if err != nil {
		logrus.WithError(err).Panic()
	}

	if len(kafkaServers) == 0 {
		logrus.WithFields(
			logrus.Fields{
				"KafkaServers": viper.GetStringSlice(config.KafkaServersKey),
			}).WithError(errors.New("no valid Kafka servers parsed from environment variable")).Panic()
	}

	mapperClient := mapperclient.NewMapperClient(viper.GetString(sharedconfig.MapperApiUrlKey))
	w, err := logwatcher.NewWatcher(
		mapperClient,
		kafkaServers,
	)
	if err != nil {
		logrus.WithError(err).Panic()
	}

	sigHandlerCtx := signals.SetupSignalHandler()
	if err = w.ValidateKafkaServers(sigHandlerCtx); err != nil {
		logrus.WithError(err).Panic()
	}

	if err := w.RunForever(sigHandlerCtx); err != nil {
		logrus.WithError(err).Panic()
	}
}
