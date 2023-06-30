package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	HostProcDirKey               = "host-proc-dir"
	HostProcDirDefault           = "/hostproc"
	CallsTimeoutKey              = "calls-timeout"
	CallsTimeoutDefault          = 60 * time.Second
	SnifferReportIntervalKey     = "sniffer-report-interval"
	SnifferReportIntervalDefault = 10 * time.Second
	PacketsBufferLengthKey       = "packets-buffer-length"
	PacketsBufferLengthDefault   = 4096
)

func init() {
	viper.SetDefault(SnifferReportIntervalKey, SnifferReportIntervalDefault)
	viper.SetDefault(PacketsBufferLengthKey, PacketsBufferLengthDefault)
	viper.SetDefault(CallsTimeoutKey, CallsTimeoutDefault)
	viper.SetDefault(HostProcDirKey, HostProcDirDefault)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
