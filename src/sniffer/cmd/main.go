package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/bombsimon/logrusr/v3"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared/telemetries/componentinfo"
	"github.com/otterize/intents-operator/src/shared/telemetries/errorreporter"
	"github.com/otterize/network-mapper/src/shared/componentutils"
	"github.com/otterize/network-mapper/src/shared/version"
	"golang.org/x/sync/errgroup"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/otterize/network-mapper/src/sniffer/pkg/sniffer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	errorreporter.Init("sniffer", version.Version(), viper.GetString(sharedconfig.TelemetryErrorsAPIKeyKey))
	logrus.SetLevel(logrus.InfoLevel)
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	ctrl.SetLogger(logrusr.New(logrus.StandardLogger()))
	componentutils.SetCloudClientId()
	componentinfo.SetGlobalComponentInstanceId()

	mapperClient := mapperclient.NewMapperClient(viper.GetString(sharedconfig.MapperApiUrlKey))

	healthServer := echo.New()
	healthServer.GET("/healthz", func(c echo.Context) error {
		defer errorreporter.AutoNotify()
		err := mapperClient.Health(c.Request().Context())
		if err != nil {
			return err
		}
		return c.NoContent(http.StatusOK)
	})

	metricsServer := echo.New()

	metricsServer.GET("/metrics", echoprometheus.NewHandler())
	errgrp, errGroupCtx := errgroup.WithContext(signals.SetupSignalHandler())
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
		snifferInstance := sniffer.NewSniffer(mapperClient)
		return snifferInstance.RunForever(errGroupCtx)
	})

	errgrp.Go(func() error {
		defer errorreporter.AutoNotify()
		return componentutils.WaitAndSetContextId(errGroupCtx)
	})

	err := errgrp.Wait()
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
