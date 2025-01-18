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
	HealthProbesPortKey          = "health-probes-port"
	HealthProbesPortDefault      = "57921"
	TelemetryErrorsAPIKeyKey     = "telemetry-errors-api-key"
	TelemetryErrorsAPIKeyDefault = "d86195588a41fa03aa6711993bb1c765"
	EnableTCPKey                 = "enable-tcp"
	EnableTCPSnifferDefault      = true
	EnableSocketScannerKey       = "enable-socket-scanner"
	EnableSocketScannerDefault   = true
	EnableDNSKey                 = "enable-dns"
	EnableDNSSnifferDefault      = true

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
	viper.SetDefault(HealthProbesPortKey, HealthProbesPortDefault)
	viper.SetDefault(TelemetryErrorsAPIKeyKey, TelemetryErrorsAPIKeyDefault)
	viper.SetDefault(EnableTCPKey, EnableTCPSnifferDefault)
	viper.SetDefault(EnableSocketScannerKey, EnableSocketScannerDefault)
	viper.SetDefault(EnableDNSKey, EnableDNSSnifferDefault)
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
