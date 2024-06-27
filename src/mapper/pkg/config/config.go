package config

import (
	"github.com/amit7itz/goset"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/spf13/viper"
	"time"
)

const (
	ClusterDomainKey                      = "cluster-domain"
	ClusterDomainDefault                  = kubeutils.DefaultClusterDomain
	CloudApiAddrKey                       = "api-address"
	CloudApiAddrDefault                   = "https://app.otterize.com/api"
	UploadIntervalSecondsKey              = "upload-interval-seconds"
	UploadIntervalSecondsDefault          = 60
	UploadBatchSizeKey                    = "upload-batch-size"
	UploadBatchSizeDefault                = 500
	ExcludedNamespacesKey                 = "exclude-namespaces"
	OTelEnabledKey                        = "enable-otel-export"
	OTelEnabledDefault                    = false
	OTelMetricKey                         = "otel-metric-name"
	OTelMetricDefault                     = "traces_service_graph_request_total" // same as expected in otel-collector-contrib's servicegraphprocessor
	ExternalTrafficCaptureEnabledKey      = "capture-external-traffic-enabled"
	ExternalTrafficCaptureEnabledDefault  = true
	CreateWebhookCertificateKey           = "create-webhook-certificate"
	CreateWebhookCertificateDefault       = true
	EnableAWSVisibilityWebHookKey         = "enable-aws-visibility-webhook"
	EnableAWSVisibilityWebHookDefault     = false
	DNSCacheItemsMaxCapacityKey           = "dns-cache-items-max-capacity"
	DNSCacheItemsMaxCapacityDefault       = 100000
	DNSClientIntentsUpdateIntervalKey     = "dns-client-intents-update-interval"
	DNSClientIntentsUpdateIntervalDefault = 1 * time.Second
	DNSClientIntentsUpdateEnabledKey      = "dns-client-intents-update-enabled"
	DNSClientIntentsUpdateEnabledDefault  = true
	ServiceCacheTTLDurationKey            = "service-cache-ttl-duration"
	ServiceCacheTTLDurationDefault        = 1 * time.Minute
	ServiceCacheSizeKey                   = "service-cache-size"
	ServiceCacheSizeDefault               = 10000

	EnableIstioCollectionKey           = "enable-istio-collection"
	EnableIstioCollectionDefault       = false
	IstioRestrictCollectionToNamespace = "istio-restrict-collection-to-namespace"
	IstioReportIntervalKey             = "istio-report-interval"
	IstioReportIntervalDefault         = 30 * time.Second
	IstioCooldownIntervalKey           = "istio-cooldown-interval"
	IstioCooldownIntervalDefault       = 15 * time.Second
	MetricFetchTimeoutKey              = "istio-metric-fetch-timeout"
	MetricFetchTimeoutDefault          = 10 * time.Second
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
	viper.SetDefault(OTelEnabledKey, OTelEnabledDefault)
	viper.SetDefault(OTelMetricKey, OTelMetricDefault)
	viper.SetDefault(ExternalTrafficCaptureEnabledKey, ExternalTrafficCaptureEnabledDefault)
	viper.SetDefault(CreateWebhookCertificateKey, CreateWebhookCertificateDefault)
	viper.SetDefault(EnableAWSVisibilityWebHookKey, EnableAWSVisibilityWebHookDefault)
	viper.SetDefault(DNSCacheItemsMaxCapacityKey, DNSCacheItemsMaxCapacityDefault)
	viper.SetDefault(DNSClientIntentsUpdateIntervalKey, DNSClientIntentsUpdateIntervalDefault)
	viper.SetDefault(DNSClientIntentsUpdateEnabledKey, DNSClientIntentsUpdateEnabledDefault)
	viper.SetDefault(IstioReportIntervalKey, IstioReportIntervalDefault)
	viper.SetDefault(MetricFetchTimeoutKey, MetricFetchTimeoutDefault)
	viper.SetDefault(IstioCooldownIntervalKey, IstioCooldownIntervalDefault)
	viper.SetDefault(IstioRestrictCollectionToNamespace, "")
	viper.SetDefault(EnableIstioCollectionKey, EnableIstioCollectionDefault)
	viper.SetDefault(ServiceCacheTTLDurationKey, ServiceCacheTTLDurationDefault)
	viper.SetDefault(ServiceCacheSizeKey, ServiceCacheSizeDefault)
	excludedNamespaces = goset.FromSlice(viper.GetStringSlice(ExcludedNamespacesKey))
}
