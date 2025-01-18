package main

import (
	"context"
	"fmt"
	"github.com/bombsimon/logrusr/v3"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared"
	"github.com/otterize/intents-operator/src/shared/clusterutils"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/telemetries/componentinfo"
	"github.com/otterize/intents-operator/src/shared/telemetries/errorreporter"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetriesgql"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetrysender"
	"github.com/otterize/network-mapper/src/mapperclient"
	"github.com/otterize/network-mapper/src/shared/version"
	"golang.org/x/sync/errgroup"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/sniffer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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
	healthProbesPort := viper.GetInt(sharedconfig.HealthProbesPortKey)

	healthServer := echo.New()
	healthServer.HideBanner = true
	healthServer.GET("/healthz", func(c echo.Context) error {
		defer errorreporter.AutoNotify()
		err := mapperClient.Health(c.Request().Context())
		if err != nil {
			return errors.Wrap(err)
		}
		return c.NoContent(http.StatusOK)
	})

	metricsServer := echo.New()
	metricsServer.HideBanner = true

	metricsServer.GET("/metrics", echoprometheus.NewHandler())

	componentinfo.SetGlobalContextId(telemetrysender.Anonymize(clusterUID))
	logrus.Debug("Starting metrics server")
	errgrp.Go(func() error {
		logrus.Debug("Started metrics server")
		defer errorreporter.AutoNotify()
		err := metricsServer.Start(fmt.Sprintf(":%d", viper.GetInt(sharedconfig.PrometheusMetricsPortKey)))
		if err != nil {
			logrus.WithError(err).Error("Error when starting metrics server, however not returning an error as this may be due to the same port being used on the host node, and the sniffer runs in hostNetwork mode.")
		}
		return nil
	})
	logrus.Debug("Starting health server")
	errgrp.Go(func() error {
		logrus.Debug("Started health server")
		defer errorreporter.AutoNotify()
		return healthServer.Start(fmt.Sprintf(":%d", healthProbesPort))
	})

	logrus.Debug("Starting sniffer")

	errgrp.Go(func() error {
		logrus.Debug("Started sniffer")
		defer errorreporter.AutoNotify()
		snifferInstance := sniffer.NewSniffer(mapperClient)
		return snifferInstance.RunForever(errGroupCtx)
	})
	<-errGroupCtx.Done()
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := healthServer.Shutdown(timeoutCtx)
	if err != nil {
		logrus.WithError(err).Panic("Error when shutting down")
	}

	err = metricsServer.Shutdown(timeoutCtx)
	if err != nil {
		logrus.WithError(err).Panic("Error when shutting down")
	}

	err = errgrp.Wait()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logrus.WithError(err).Panic("Error when running server or HTTP server")
	}

	logrus.Info("Sniffer stopped")
}
