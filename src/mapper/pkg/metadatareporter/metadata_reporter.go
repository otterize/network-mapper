package metadatareporter

import (
	"context"
	"github.com/amit7itz/goset"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

const (
	cacheTTL                    = 5 * time.Hour
	cacheSize                   = 1000
	endpointsPodNamesIndexField = "endpointsPodNames"
)

type workloadMetadata struct {
	metadataInput cloudclient.ReportServiceMetadataInput
	cacheKey      serviceIdentityKey
	cacheValue    metadataChecksum
}

type MetadataReporter struct {
	client.Client
	cloudClient       cloudclient.CloudClient
	serviceIDResolver serviceidresolver.ServiceResolver
	cache             *workloadMetadataCache
	reportCacheLock   sync.Mutex
	once              sync.Once
}

func NewMetadataReporter(
	client client.Client,
	cloudClient cloudclient.CloudClient,
	serviceIDResolver serviceidresolver.ServiceResolver,
) *MetadataReporter {
	return &MetadataReporter{
		Client:            client,
		cloudClient:       cloudClient,
		serviceIDResolver: serviceIDResolver,
		cache:             newWorkloadMetadataCache(cacheSize, cacheTTL),
	}
}

func (r *MetadataReporter) ReportMetadata(ctx context.Context, serviceIdentities []serviceidentity.ServiceIdentity) error {
	r.once.Do(func() {
		// Sync all pods in batches per namespaces
		// This way we will avoid hitting the API server with too many requests
		err := r.syncAllOnce(ctx)
		if err != nil {
			logrus.WithError(err).Warnf("failed to report all pod labels, will continue with individual pod reconciliation")
		}
	})
	return errors.Wrap(r.reportWorkloadMetadataWithCache(ctx, serviceIdentities))
}

func (r *MetadataReporter) reportWorkloadMetadataWithCache(ctx context.Context, serviceIdentities []serviceidentity.ServiceIdentity) error {
	r.reportCacheLock.Lock()
	defer r.reportCacheLock.Unlock()
	metadataToReport := make([]workloadMetadata, 0)
	for _, serviceIdentity := range serviceIdentities {
		workloadMeta, ok, err := r.serviceIdentityToWorkloadMetadata(ctx, serviceIdentity)
		if err != nil {
			return errors.Wrap(err)
		}
		if !ok {
			continue
		}
		cached := r.cache.IsCached(workloadMeta.cacheKey, workloadMeta.cacheValue)
		if cached {
			continue
		}
		metadataToReport = append(metadataToReport, workloadMeta)
	}

	err := r.reportMetadataToCloud(ctx, metadataToReport)
	if err != nil {
		return errors.Wrap(err)
	}

	for _, metadata := range metadataToReport {
		r.cache.Add(metadata.cacheKey, metadata.cacheValue)
	}

	return nil
}

func (r *MetadataReporter) reportMetadataToCloud(ctx context.Context, metadataToReport []workloadMetadata) error {
	if len(metadataToReport) == 0 {
		return nil
	}

	cloudInputs := lo.Map(metadataToReport, func(meta workloadMetadata, _ int) cloudclient.ReportServiceMetadataInput {
		return meta.metadataInput
	})

	// Sort the inputs to ensure consistent ordering
	slices.SortFunc(cloudInputs, metaDataInputSortFunc)

	err := r.cloudClient.ReportWorkloadsMetadata(ctx, cloudInputs)
	if err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func (r *MetadataReporter) serviceIdentityToWorkloadMetadata(ctx context.Context, serviceIdentity serviceidentity.ServiceIdentity) (workloadMetadata, bool, error) {
	pods, ok, err := r.serviceIDResolver.ResolveServiceIdentityToPodSlice(ctx, serviceIdentity)
	if err != nil {
		return workloadMetadata{}, false, errors.Wrap(err)
	}
	if len(pods) == 0 || !ok {
		return workloadMetadata{}, false, nil
	}

	podIps, err := r.fetchPodsIPs(pods)
	if err != nil {
		return workloadMetadata{}, false, errors.Wrap(err)
	}
	serviceIps, err := r.fetchServiceIPs(ctx, serviceIdentity, pods)
	if err != nil {
		return workloadMetadata{}, false, errors.Wrap(err)
	}

	return workloadMetadata{
		metadataInput: cloudclient.ReportServiceMetadataInput{
			Identity: serviceIdentityToServiceIdentityInput(serviceIdentity),
			Metadata: cloudclient.ServiceMetadataInput{
				Labels:     labelsToLabelInput(pods[0].Labels),
				PodIps:     podIps,
				ServiceIps: serviceIps,
			},
		},
		cacheKey:   serviceIdentityToCacheKey(serviceIdentity),
		cacheValue: checksumMetadata(pods[0].Labels, podIps, serviceIps),
	}, true, nil
}

func (r *MetadataReporter) fetchPodsIPs(pods []corev1.Pod) ([]string, error) {
	podIps := goset.NewSet[string]()
	for _, pod := range pods {
		if pod.Status.PodIP != "" {
			podIps.Add(pod.Status.PodIP)
		}
		for _, ip := range pod.Status.PodIPs {
			podIps.Add(ip.IP)
		}
	}
	return podIps.Items(), nil
}

func (r *MetadataReporter) fetchServiceIPs(ctx context.Context, serviceIdentity serviceidentity.ServiceIdentity, pods []corev1.Pod) ([]string, error) {
	serviceIps := goset.NewSet[string]()

	serviceNames := goset.NewSet[string]()
	for _, pod := range pods {
		endpointsList := &corev1.EndpointsList{}
		err := r.List(ctx, endpointsList, client.InNamespace(pod.Namespace), client.MatchingFields{endpointsPodNamesIndexField: pod.Name})
		if err != nil {
			return nil, errors.Wrap(err)
		}
		serviceNames.Add(lo.Map(endpointsList.Items, func(ep corev1.Endpoints, _ int) string { return ep.Name })...)
	}

	for _, serviceName := range serviceNames.Items() {
		service := &corev1.Service{}
		err := r.Get(ctx, client.ObjectKey{Namespace: serviceIdentity.Namespace, Name: serviceName}, service)
		if k8serrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, errors.Wrap(err)
		}
		if service.Spec.Type != corev1.ServiceTypeClusterIP {
			continue
		}

		for _, ip := range service.Spec.ClusterIPs {
			if ip != "" {
				serviceIps.Add(ip)
			}
		}
	}
	return serviceIps.Items(), nil
}

func metaDataInputSortFunc(a, b cloudclient.ReportServiceMetadataInput) int {
	if a.Identity.Name < b.Identity.Name {
		return -1
	}
	if a.Identity.Name > b.Identity.Name {
		return 1
	}
	return 0
}
