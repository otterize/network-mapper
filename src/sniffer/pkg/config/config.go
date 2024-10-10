package config

import (
	"github.com/spf13/viper"
	"time"
)

const (
	HostProcDirKey                     = "host-proc-dir"
	HostProcDirDefault                 = "/hostproc"
	CallsTimeoutKey                    = "calls-timeout"
	CallsTimeoutDefault                = 60 * time.Second
	SnifferReportIntervalKey           = "sniffer-report-interval"
	SnifferReportIntervalDefault       = 1 * time.Second
	PacketsBufferLengthKey             = "packets-buffer-length"
	PacketsBufferLengthDefault         = 4096
	HostsMappingRefreshIntervalKey     = "hosts-mapping-refresh-interval"
	HostsMappingRefreshIntervalDefault = 500 * time.Millisecond
	UseExtendedProcfsResolutionKey     = "use-extended-procfs-resolution"
	UseExtendedProcfsResolutionDefault = false
)

func init() {
	viper.SetDefault(SnifferReportIntervalKey, SnifferReportIntervalDefault)
	viper.SetDefault(PacketsBufferLengthKey, PacketsBufferLengthDefault)
	viper.SetDefault(CallsTimeoutKey, CallsTimeoutDefault)
	viper.SetDefault(HostProcDirKey, HostProcDirDefault)
	viper.SetDefault(HostsMappingRefreshIntervalKey, HostsMappingRefreshIntervalDefault)
	viper.SetDefault(UseExtendedProcfsResolutionKey, UseExtendedProcfsResolutionDefault)
}
