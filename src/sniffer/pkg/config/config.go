package config

import (
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	MaxResultsPerUploadKey        = "max-results-per-upload"
	MaxResultsPerUploadDefault    = 100
	HostProcDirKey                = "host-proc-dir"
	HostProcDirDefault            = "/hostproc"
	CallsTimeoutKey               = "calls-timeout"
	CallsTimeoutDefault           = 5 * time.Second
	SnifferReportIntervalKey      = "sniffer-report-interval"
	SnifferReportIntervalDefault  = 10 * time.Second
	SnifferResolveIntervalKey     = "sniffer-resolve-interval"
	SnifferResolveIntervalDefault = 1 * time.Second
	SnifferResolveLocallyKey      = "exp-sniffer-resolve-locally"
	SnifferResolveLocallyDefault  = false
)

func init() {
	viper.SetDefault(MaxResultsPerUploadKey, MaxResultsPerUploadDefault)
	viper.SetDefault(SnifferReportIntervalKey, SnifferReportIntervalDefault)
	viper.SetDefault(SnifferResolveIntervalKey, SnifferResolveIntervalDefault)
	viper.SetDefault(SnifferResolveLocallyKey, SnifferResolveLocallyDefault)
	viper.SetDefault(CallsTimeoutKey, CallsTimeoutDefault)
	viper.SetDefault(HostProcDirKey, HostProcDirDefault)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
