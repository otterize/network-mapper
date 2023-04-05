package config

import (
	"github.com/spf13/viper"
	"time"
)

const (
	NamespaceKey                 = "namespace"
	IstioReportIntervalKey       = "istio-report-interval"
	IstioReportIntervalDefault   = 30 * time.Second
	IstioCooldownIntervalKey     = "istio-cooldown-interval"
	IstioCooldownIntervalDefault = 15 * time.Second
	MetricFetchTimeoutKey        = "metric-fetch-timeout"
	MetricFetchTimeoutDefault    = 10 * time.Second
)

func init() {
	viper.SetDefault(IstioReportIntervalKey, IstioReportIntervalDefault)
	viper.SetDefault(MetricFetchTimeoutKey, MetricFetchTimeoutDefault)
	viper.SetDefault(IstioCooldownIntervalKey, IstioCooldownIntervalDefault)
	viper.SetDefault(NamespaceKey, "")
}
