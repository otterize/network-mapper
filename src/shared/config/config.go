package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
	"time"
)

/*
Shared config keys between all reporter components - DNS Sniffer, Kafka watcher, Istio watcher
*/

const (
	MapperApiUrlKey                            = "mapper-api-url"
	MapperApiUrlDefault                        = "http://mapper:9090/query"
	DebugKey                                   = "debug"
	DebugDefault                               = false
	PrometheusMetricsPortKey                   = "metrics-port"
	PrometheusMetricsPortDefault               = 2112
	TelemetryErrorsAPIKeyKey                   = "telemetry-errors-api-key"
	TelemetryErrorsAPIKeyDefault               = "d86195588a41fa03aa6711993bb1c765"
	ComponentMetadataConfigmapNameKey          = "component-metadata-configmap-name"
	componentMetadataConfigmapNameDefault      = "otterize-network-mapper-component-config-map"
	ReadContextIdIntervalKey                   = "read-context-id-interval"
	ReadContextIdIntervalDefault               = 5 * time.Second
	ComponentMetadataConfigmapMountPathKey     = "component-configmap-path"
	ComponentMetadataConfigmapMountPathDefault = "/etc/otterize"
	ComponentMetadataContextIdKeyKey           = "context-id-key"
	ComponentMetadataContextIdKeyDefault       = "CONTEXT_ID"
	EnvPodKey                                  = "pod"
	EnvNamespaceKey                            = "namespace"

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
	viper.SetDefault(TelemetryErrorsAPIKeyKey, TelemetryErrorsAPIKeyDefault)
	viper.SetDefault(ComponentMetadataConfigmapNameKey, componentMetadataConfigmapNameDefault)
	viper.SetDefault(ReadContextIdIntervalKey, ReadContextIdIntervalDefault)
	viper.SetDefault(ComponentMetadataConfigmapMountPathKey, ComponentMetadataConfigmapMountPathDefault)
	viper.SetDefault(ComponentMetadataContextIdKeyKey, ComponentMetadataContextIdKeyDefault)
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
