package config

import (
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/spf13/viper"
	"strings"
)

const (
	EnvPrefix            = "OTTERIZE"
	ClusterDomainKey     = "cluster-domain"
	ClusterDomainDefault = kubeutils.DefaultClusterDomain
	DebugKey             = "debug"
	DebugDefault         = false
	StoreFileKey         = "store-file"
	StoreFileDefault     = "/data/store.json"
)

func init() {
	viper.SetDefault(DebugKey, DebugDefault)
	viper.SetDefault(ClusterDomainKey, ClusterDomainDefault) // If not set by the user, the main.go of mapper will try to find the cluster domain and set it itself.
	viper.SetDefault(StoreFileKey, StoreFileDefault)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
