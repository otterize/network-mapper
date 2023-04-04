package config

import (
	"github.com/otterize/network-mapper/src/shared/config"
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	HostProcDirKey      = "host-proc-dir"
	HostProcDirDefault  = "/hostproc"
	CallsTimeoutKey     = "calls-timeout"
	CallsTimeoutDefault = 5 * time.Second
)

func init() {
	viper.SetDefault(config.MapperApiUrlKey, config.MapperApiUrlDefault)
	viper.SetDefault(config.ReportIntervalKey, config.ReportIntervalDefault)
	viper.SetDefault(config.DebugKey, config.DebugDefault)
	viper.SetEnvPrefix(config.EnvPrefix)

	// Sniffer specific flags
	viper.SetDefault(CallsTimeoutKey, CallsTimeoutDefault)
	viper.SetDefault(HostProcDirKey, HostProcDirDefault)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
