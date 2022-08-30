package config

import (
	"github.com/otterize/otternose/shared/kubeutils"
	"github.com/spf13/viper"
	"strings"
)

const (
	EnvPrefix            = "OTTERNOSE"
	ClusterDomainKey     = "cluster-domain"
	ClusterDomainDefault = kubeutils.DefaultClusterDomain
	DebugKey             = "debug"
	DebugDefault         = false
)

func init() {
	viper.SetDefault(DebugKey, DebugDefault)
	viper.SetDefault(ClusterDomainKey, ClusterDomainDefault) // If not set by the user, the main.go of mapper will try to find the cluster domain and set it itself.
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
