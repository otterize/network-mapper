package sniffer

import (
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	EnvPrefix             = "OTTERNOSE"
	mapperApiUrlKey       = "mapper-api-url"
	mapperApiUrlDefault   = "http://mapper:9090/query"
	reportIntervalKey     = "report-interval"
	reportIntervalDefault = 10 * time.Second
	callsTimeoutKey       = "calls-timeout"
	callsTimeoutDefault   = 5 * time.Second
)

func init() {
	viper.SetDefault(mapperApiUrlKey, mapperApiUrlDefault)
	viper.SetDefault(reportIntervalKey, reportIntervalDefault)
	viper.SetDefault(callsTimeoutKey, callsTimeoutDefault)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
