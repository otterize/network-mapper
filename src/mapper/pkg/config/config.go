package config

import (
	"github.com/spf13/viper"
	"strings"
)

const (
	EnvPrefix        = "OTTERNOSE"
	ClusterDomainKey = "cluster-domain"
	DebugKey         = "debug"
	DebugDefault     = false
)

func init() {
	viper.SetDefault(DebugKey, DebugDefault)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
