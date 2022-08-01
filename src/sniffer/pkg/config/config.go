package config

import (
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	EnvPrefix             = "OTTERNOSE"
	MapperApiUrlKey       = "mapper-api-url"
	MapperApiUrlDefault   = "http://mapper:9090/query"
	ReportIntervalKey     = "report-interval"
	ReportIntervalDefault = 10 * time.Second
	CallsTimeoutKey       = "calls-timeout"
	CallsTimeoutDefault   = 5 * time.Second
	HostProcDirKey        = "host-proc-dir"
	HostProcDirDefault    = "/hostproc"
)

func init() {
	viper.SetDefault(MapperApiUrlKey, MapperApiUrlDefault)
	viper.SetDefault(ReportIntervalKey, ReportIntervalDefault)
	viper.SetDefault(CallsTimeoutKey, CallsTimeoutDefault)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
