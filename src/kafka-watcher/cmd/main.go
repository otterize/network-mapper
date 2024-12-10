package main

import (
	"context"
	"fmt"
	"github.com/bombsimon/logrusr/v3"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared"
	"github.com/otterize/intents-operator/src/shared/clusterutils"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/telemetries/componentinfo"
	"github.com/otterize/intents-operator/src/shared/telemetries/errorreporter"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetriesgql"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetrysender"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/config"
	logwatcher2 "github.com/otterize/network-mapper/src/kafka-watcher/pkg/logwatcher"
	"github.com/otterize/network-mapper/src/mapperclient"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/shared/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"strings"
	"time"
)

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	errgrp, errGroupCtx := errgroup.WithContext(signals.SetupSignalHandler())
	clusterUID := clusterutils.GetOrCreateClusterUID(errGroupCtx)
	componentinfo.SetGlobalContextId(telemetrysender.Anonymize(clusterUID))
	errorreporter.Init(telemetriesgql.TelemetryComponentTypeNetworkMapper, version.Version())
	defer errorreporter.AutoNotify()
	shared.RegisterPanicHandlers()

	ctrl.SetLogger(logrusr.New(logrus.StandardLogger()))

	mapperClient := mapperclient.New(viper.GetString(sharedconfig.MapperApiUrlKey))

	mode := viper.GetString(config.KafkaLogReadModeKey)

	var watcher logwatcher2.Watcher
	var err error

	switch mode {
	case config.FileReadMode:
		logPath := viper.GetString(config.KafkaAuthZLogPathKey)

		if logPath == "" {
			logrus.Panicf("Kafka log path is not set - please set %s", sharedconfig.GetEnvVarForKey(config.KafkaAuthZLogPathKey))

		}

		logrus.Infof("Kafka watcher: reading from filesystem - %s", logPath)

		serverName := types.NamespacedName{
			Namespace: viper.GetString(sharedconfig.EnvNamespaceKey),
			Name:      viper.GetString(sharedconfig.EnvPodKey),
		}

		watcher, err = logwatcher2.NewLogFileWatcher(mapperClient, logPath, serverName)
		if err != nil {
			logrus.WithError(err).Panic("could not initialize log file watcher")
		}
	case config.KubernetesLogReadMode:
		kafkaServers, parseErr := parseKafkaServers(viper.GetStringSlice(config.KafkaServersKey))
		logrus.Infof("Reading from k8s logs - %d servers", len(kafkaServers))

		if parseErr != nil {
			logrus.WithError(err).Panic("could not parse Kafka servers list")
		}

		watcher, err = logwatcher2.NewKubernetesLogWatcher(mapperClient, kafkaServers)
		if err != nil {
			logrus.WithError(err).Panic("could not initialize Kubernetes log watcher")
		}
	case "":
		logrus.Panicf("Kafka watcher mode is not set - please set %s", sharedconfig.GetEnvVarForKey(config.KafkaLogReadModeKey))
	default:
		logrus.Panicf("Kafka watcher mode (%s) is not set to a valid mode", mode)
	}

	healthServer := echo.New()
	healthServer.HideBanner = true
	healthServer.GET("/healthz", func(c echo.Context) error {
		err := mapperClient.Health(c.Request().Context())
		if err != nil {
			return errors.Wrap(err)
		}
		return c.NoContent(http.StatusOK)
	})

	metricsServer := echo.New()
	metricsServer.HideBanner = true

	metricsServer.GET("/metrics", echoprometheus.NewHandler())
	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		return metricsServer.Start(fmt.Sprintf(":%d", viper.GetInt(sharedconfig.PrometheusMetricsPortKey)))
	})
	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		return healthServer.Start(":9090")
	})

	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		err := watcher.RunForever(errGroupCtx)
		return errors.Wrap(err)
	})

	err = errgrp.Wait()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logrus.WithError(err).Panic("Error when running server or HTTP server")
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = healthServer.Shutdown(timeoutCtx)
	if err != nil {
		logrus.WithError(err).Panic("Error when shutting down")
	}

	err = metricsServer.Shutdown(timeoutCtx)
	if err != nil {
		logrus.WithError(err).Panic("Error when shutting down")
	}
}

func parseKafkaServers(serverNames []string) ([]types.NamespacedName, error) {
	var servers []types.NamespacedName
	for _, serverName := range serverNames {
		nameParts := strings.Split(serverName, ".")
		if len(nameParts) != 2 {
			return nil, errors.Errorf("error parsing server pod name %s - should be formatted as 'name.namespace'", serverName)
		}
		servers = append(servers, types.NamespacedName{
			Name:      nameParts[0],
			Namespace: nameParts[1],
		})
	}
	return servers, nil
}
