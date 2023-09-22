package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

/*
Shared config keys between all reporter components - DNS Sniffer, Kafka watcher, Istio watcher
*/

const (
	MapperApiUrlKey              = "mapper-api-url"
	MapperApiUrlDefault          = "http://mapper:9090/query"
	DebugKey                     = "debug"
	DebugDefault                 = false
	PrometheusMetricsPortKey     = "metrics-port"
	PrometheusMetricsPortDefault = 2112

	EnvPodKey       = "pod"
	EnvNamespaceKey = "namespace"

	envPrefix = "OTTERIZE"
)

var replacer = strings.NewReplacer("-", "_")

func GetEnvVarForKey(key string) string {
	return fmt.Sprintf("%s_%s", envPrefix, replacer.Replace(key))
}

func init() {
	viper.SetDefault(MapperApiUrlKey, MapperApiUrlDefault)
	viper.SetDefault(DebugKey, DebugDefault)
	viper.SetDefault(PrometheusMetricsPortKey, PrometheusMetricsPortDefault)
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
