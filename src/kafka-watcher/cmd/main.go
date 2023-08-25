package main

import (
	"fmt"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/config"
	logwatcher2 "github.com/otterize/network-mapper/src/kafka-watcher/pkg/logwatcher"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/mapperclient"
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
	var watcher logwatcher2.Watcher

	switch mode {
	case config.FileReadMode:
		logPath := viper.GetString(config.KafkaAuthZLogPathKey)

		if logPath == "" {
			logrus.Fatalf("Kafka log path is not set - please set %s", sharedconfig.GetEnvVarForKey(config.KafkaAuthZLogPathKey))

		}

		logrus.Infof("Kafka watcher: reading from filesystem - %s", logPath)

		serverName := types.NamespacedName{
			Namespace: viper.GetString(sharedconfig.EnvNamespaceKey),
			Name:      viper.GetString(sharedconfig.EnvPodKey),
		}

		watcher, err = logwatcher2.NewLogFileWatcher(mapperClient, logPath, serverName)
		if err != nil {
			logrus.WithError(err).Fatal("could not initialize log file watcher")
		}
	case config.KubernetesLogReadMode:
		kafkaServers, parseErr := parseKafkaServers(viper.GetStringSlice(config.KafkaServersKey))
		logrus.Infof("Reading from k8s logs - %d servers", len(kafkaServers))

		if parseErr != nil {
			logrus.WithError(err).Fatal("could not parse Kafka servers list")
		}

		watcher, err = logwatcher2.NewKubernetesLogWatcher(mapperClient, kafkaServers)
		if err != nil {
			logrus.WithError(err).Fatal("could not initialize Kubernetes log watcher")
		}
	case "":
		logrus.Fatalf("Kafka watcher mode is not set - please set %s", sharedconfig.GetEnvVarForKey(config.KafkaLogReadModeKey))
	default:
		logrus.Fatalf("Kafka watcher mode (%s) is not set to a valid mode", mode)
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
