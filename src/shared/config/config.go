package config

import (
	"github.com/spf13/viper"
	"strings"
)

/*
Shared config keys between all reporter components - DNS Sniffer, Kafka watcher, Istio watcher
*/

const (
	MapperApiUrlKey     = "mapper-api-url"
	MapperApiUrlDefault = "http://mapper:9090/query"
	DebugKey            = "debug"
	DebugDefault        = false
	EnvPrefix           = "OTTERIZE"
)

func init() {
	viper.SetDefault(MapperApiUrlKey, MapperApiUrlDefault)
	viper.SetDefault(DebugKey, DebugDefault)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
