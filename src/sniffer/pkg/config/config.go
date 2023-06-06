package config

import (
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	HostProcDirKey               = "host-proc-dir"
	HostProcDirDefault           = "/hostproc"
	CallsTimeoutKey              = "calls-timeout"
	CallsTimeoutDefault          = 60 * time.Second
	SnifferReportIntervalKey     = "sniffer-report-interval"
	SnifferReportIntervalDefault = 10 * time.Second
)

func init() {
	viper.SetDefault(SnifferReportIntervalKey, SnifferReportIntervalDefault)
	viper.SetDefault(CallsTimeoutKey, CallsTimeoutDefault)
	viper.SetDefault(HostProcDirKey, HostProcDirDefault)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
