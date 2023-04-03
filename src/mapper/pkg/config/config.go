package config

import (
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/spf13/viper"
	"strings"
)

const (
	ClusterDomainKey             = "cluster-domain"
	ClusterDomainDefault         = kubeutils.DefaultClusterDomain
	CloudApiAddrKey              = "api-address"
	CloudApiAddrDefault          = "https://app.otterize.com/api"
	UploadIntervalSecondsKey     = "upload-interval-seconds"
	UploadIntervalSecondsDefault = 60
)

func init() {
	viper.SetDefault(sharedconfig.DebugKey, sharedconfig.DebugDefault)
	viper.SetDefault(ClusterDomainKey, ClusterDomainDefault) // If not set by the user, the main.go of mapper will try to find the cluster domain and set it itself.
	viper.SetDefault(CloudApiAddrKey, CloudApiAddrDefault)
	viper.SetDefault(UploadIntervalSecondsKey, UploadIntervalSecondsDefault)
	viper.SetEnvPrefix(sharedconfig.EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
