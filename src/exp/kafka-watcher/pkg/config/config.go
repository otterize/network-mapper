package config

import (
	"github.com/otterize/network-mapper/src/shared/config"
	"github.com/spf13/viper"
	"strings"
)

const (
	KafkaServersKey = "kafka-servers"
)

func init() {
	viper.SetDefault(config.MapperApiUrlKey, config.MapperApiUrlDefault)
	viper.SetDefault(config.ReportIntervalKey, config.ReportIntervalDefault)
	viper.SetDefault(config.CallsTimeoutKey, config.CallsTimeoutDefault)
	viper.SetDefault(config.CooldownIntervalKey, config.CooldownIntervalDefault)
	viper.SetDefault(config.DebugKey, config.DebugDefault)
	viper.SetEnvPrefix(config.EnvPrefix)

	// Kafka watcher specific flags
	viper.SetDefault(KafkaServersKey, []string{})
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
