package config

import (
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/spf13/viper"
	"strings"
)

func init() {
	viper.SetDefault(sharedconfig.MapperApiUrlKey, sharedconfig.MapperApiUrlDefault)
	viper.SetDefault(sharedconfig.ReportIntervalKey, sharedconfig.ReportIntervalDefault)
	viper.SetDefault(sharedconfig.CallsTimeoutKey, sharedconfig.CallsTimeoutDefault)
	viper.SetDefault(sharedconfig.CooldownIntervalKey, sharedconfig.CooldownIntervalDefault)
	viper.SetDefault(sharedconfig.DebugKey, sharedconfig.DebugDefault)
	viper.SetEnvPrefix(sharedconfig.EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
