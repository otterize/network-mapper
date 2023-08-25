package main

import (
	"fmt"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/config"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/logwatcher"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/mapperclient"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"strings"
)

func main() {
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	mapperClient := mapperclient.NewMapperClient(viper.GetString(sharedconfig.MapperApiUrlKey))

	mode := viper.GetString(config.KafkaLogReadModeKey)

	var err error
	var watcher logwatcher.Watcher

	switch mode {
	case config.FileReadMode:
		logPath := viper.GetString(config.KafkaAuthZLogPathKey)

		if logPath == "" {
			logrus.Panic("Kafka log path is not set - please set OTTERIZE_KAFKA_AUTHZ_LOG_PATH")
		}

		logrus.Infof("Reading from filesystem - %s", logPath)

		serverName := types.NamespacedName{
			Namespace: viper.GetString(sharedconfig.EnvNamespaceKey),
			Name:      viper.GetString(sharedconfig.EnvPodKey),
		}

		watcher, err = logwatcher.NewLogFileWatcher(mapperClient, logPath, serverName)
	case config.KubernetesLogReadMode:
		kafkaServers, parseErr := parseKafkaServers(viper.GetStringSlice(config.KafkaServersKey))
		logrus.Infof("Reading from k8s logs - %d servers", len(kafkaServers))

		if parseErr != nil {
			logrus.WithError(err).Panic()
		}

		watcher, err = logwatcher.NewKubernetesLogWatcher(mapperClient, kafkaServers)
	case "":
		logrus.Panicf("Kafka watcher mode is not set - please set %s", sharedconfig.GetEnvVarForKey(config.KafkaLogReadModeKey))
	default:
		logrus.Panicf("Kafka watcher mode (%s) is not set to a valid mode", mode)
	}

	if err != nil {
		logrus.WithError(err).Panic()
	}

	watcher.RunForever(signals.SetupSignalHandler())
}

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
