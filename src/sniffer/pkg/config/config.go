package config

import (
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	MaxResultsPerUploadKey       = "max-results-per-upload"
	MaxResultsPerUploadDefault   = 100
	HostProcDirKey               = "host-proc-dir"
	HostProcDirDefault           = "/hostproc"
	CallsTimeoutKey              = "calls-timeout"
	CallsTimeoutDefault          = 5 * time.Second
	SnifferReportIntervalKey     = "sniffer-report-interval"
	SnifferReportIntervalDefault = 10 * time.Second
)

func init() {
	viper.SetDefault(MaxResultsPerUploadKey, MaxResultsPerUploadDefault)
	viper.SetDefault(SnifferReportIntervalKey, SnifferReportIntervalDefault)
	viper.SetDefault(CallsTimeoutKey, CallsTimeoutDefault)
	viper.SetDefault(HostProcDirKey, HostProcDirDefault)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
