package config

import (
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/spf13/viper"
	"strings"
)

const (
	EnvPrefix             = "OTTERIZE"
	ClusterDomainKey      = "cluster-domain"
	ClusterDomainDefault  = kubeutils.DefaultClusterDomain
	DebugKey              = "debug"
	DebugDefault          = false
	StoreConfigMapKey     = "store-config-map"
	StoreConfigMapDefault = "otterize-network-mapper-store"
	NamespaceKey          = "namespace" // what namespace the mapper is running at. needed for development
	NamespaceDefault      = ""
)

func init() {
	viper.SetDefault(DebugKey, DebugDefault)
	viper.SetDefault(ClusterDomainKey, ClusterDomainDefault) // If not set by the user, the main.go of mapper will try to find the cluster domain and set it itself.
	viper.SetDefault(StoreConfigMapKey, StoreConfigMapDefault)
	viper.SetDefault(NamespaceKey, NamespaceDefault)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
