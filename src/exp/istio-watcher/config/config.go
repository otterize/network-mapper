package config

import (
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	EnvPrefix               = "OTTERIZE"
	MapperApiUrlKey         = "mapper-api-url"
	MapperApiUrlDefault     = "http://mapper:9090/query"
	ReportIntervalKey       = "report-interval"
	ReportIntervalDefault   = 10 * time.Second
	CallsTimeoutKey         = "calls-timeout"
	CallsTimeoutDefault     = 5 * time.Second
	CooldownIntervalKey     = "cooldown-interval"
	CooldownIntervalDefault = 10 * time.Second
	DebugKey                = "debug"
	DebugDefault            = false
	NamespaceKey            = "namespace"
)

func init() {
	viper.SetDefault(MapperApiUrlKey, MapperApiUrlDefault)
	viper.SetDefault(ReportIntervalKey, ReportIntervalDefault)
	viper.SetDefault(CallsTimeoutKey, CallsTimeoutDefault)
	viper.SetDefault(CooldownIntervalKey, CooldownIntervalDefault)
	viper.SetDefault(DebugKey, DebugDefault)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
