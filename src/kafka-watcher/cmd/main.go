package main

import (
	"context"
	"fmt"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/config"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/logwatcher"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/mapperclient"
	"k8s.io/apimachinery/pkg/types"
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
	if viper.GetBool(config.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	kafkaServers, err := parseKafkaServers(viper.GetStringSlice(config.KafkaServersKey))
	if err != nil {
		panic(err)
	}
	mapperClient := mapperclient.NewMapperClient(viper.GetString(config.MapperApiUrlKey))
	w, err := logwatcher.NewWatcher(
		mapperClient,
		kafkaServers,
	)
	if err != nil {
		panic(err)
	}

	if err := w.RunForever(context.Background()); err != nil {
		panic(err)
	}
}
