package config

import (
	"github.com/spf13/viper"
	"strings"
)

const (
	EnvPrefix        = "OTTERNOSE"
	ClusterDomainKey = "cluster-domain"
)

func init() {
	viper.SetDefault(ClusterDomainKey, "")
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
