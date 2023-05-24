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
	AuthZLogPath        = "authz-log-path"
	AuthZLogPathDefault = "/opt/otterize/kafka-watcher/authz.log"

	EnvPodKey       = "pod"
	EnvNamespaceKey = "namespace"

	EnvPrefix = "OTTERIZE"
)

func init() {
	viper.SetDefault(MapperApiUrlKey, MapperApiUrlDefault)
	viper.SetDefault(DebugKey, DebugDefault)
	viper.SetDefault(AuthZLogPath, AuthZLogPathDefault)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
