package config

import (
	"github.com/amit7itz/goset"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/spf13/viper"
)

const (
	ClusterDomainKey                 = "cluster-domain"
	ClusterDomainDefault             = kubeutils.DefaultClusterDomain
	CloudApiAddrKey                  = "api-address"
	CloudApiAddrDefault              = "https://app.otterize.com/api"
	UploadIntervalSecondsKey         = "upload-interval-seconds"
	UploadIntervalSecondsDefault     = 60
	UploadBatchSizeKey               = "upload-batch-size"
	UploadBatchSizeDefault           = 500
	ExcludedNamespacesKey            = "exclude-namespaces"
	OTelExportIntervalSecondsKey     = "otel-export-interval-seconds"
	OTelExportIntervalSecondsDefault = 60
)

var excludedNamespaces *goset.Set[string]

func ExcludedNamespaces() *goset.Set[string] {
	return excludedNamespaces
}

func init() {
	viper.SetDefault(sharedconfig.DebugKey, sharedconfig.DebugDefault)
	viper.SetDefault(ClusterDomainKey, ClusterDomainDefault) // If not set by the user, the main.go of mapper will try to find the cluster domain and set it itself.
	viper.SetDefault(CloudApiAddrKey, CloudApiAddrDefault)
	viper.SetDefault(UploadIntervalSecondsKey, UploadIntervalSecondsDefault)
	viper.SetDefault(UploadBatchSizeKey, UploadBatchSizeDefault)
	viper.SetDefault(OTelExportIntervalSecondsKey, OTelExportIntervalSecondsDefault)
	excludedNamespaces = goset.FromSlice(viper.GetStringSlice(ExcludedNamespacesKey))
}
