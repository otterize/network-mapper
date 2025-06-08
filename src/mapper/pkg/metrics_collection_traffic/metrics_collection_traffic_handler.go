package metrics_collection_traffic

import (
	"bytes"
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

type MetricsCollectionTrafficHandler struct {
	client.Client
	serviceIdResolver *serviceidresolver.Resolver
	otterizeCloud     cloudclient.CloudClient
	cache             *MetricsCollectionTrafficCache
	reportToCloudLock sync.Mutex
}

func NewMetricsCollectionTrafficHandler(client client.Client, serviceIdResolver *serviceidresolver.Resolver, otterizeCloud cloudclient.CloudClient) *MetricsCollectionTrafficHandler {
	cache := NewMetricsCollectionTrafficCache()

	return &MetricsCollectionTrafficHandler{
		Client:            client,
		serviceIdResolver: serviceIdResolver,
		otterizeCloud:     otterizeCloud,
		cache:             cache,
	}
}

func (r *MetricsCollectionTrafficHandler) HandleAllPodsInNamespace(ctx context.Context, req ctrl.Request) error {
	podList := &corev1.PodList{}
	err := r.Client.List(ctx, podList, client.InNamespace(req.Namespace))
	if err != nil {
		return errors.Wrap(err)
	}

	scrapePods := lo.Filter(podList.Items, func(pod corev1.Pod, _ int) bool {
		return pod.Annotations["prometheus.io/scrape"] == "true"
	})

	podsToReport := make([]cloudclient.K8sResourceEligibleForMetricsCollectionInput, 0)
	for _, pod := range scrapePods {
		serviceId, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, &pod)
		if err != nil {
			return errors.Wrap(err)
		}
		podsToReport = append(podsToReport, cloudclient.K8sResourceEligibleForMetricsCollectionInput{Namespace: req.Namespace, Name: serviceId.Name, Kind: serviceId.Kind})
	}

	err = r.reportToCloud(ctx, req.Namespace, cloudclient.EligibleForMetricsCollectionReasonPodAnnotations, podsToReport)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (r *MetricsCollectionTrafficHandler) HandleAllServicesInNamespace(ctx context.Context, req ctrl.Request) error {
	serviceList := &corev1.ServiceList{}
	err := r.Client.List(ctx, serviceList, client.InNamespace(req.Namespace))
	if err != nil {
		return errors.Wrap(err)
	}

	scrapeServices := lo.Filter(serviceList.Items, func(pod corev1.Service, _ int) bool {
		return pod.Annotations["prometheus.io/scrape"] == "true"
	})

	podsToReport := make([]cloudclient.K8sResourceEligibleForMetricsCollectionInput, 0)

	for _, service := range scrapeServices {
		// Get all the pods relevant to this service
		endpoints := &corev1.Endpoints{}
		err = r.Client.Get(ctx, client.ObjectKey{Namespace: service.Namespace, Name: service.Name}, endpoints)
		if k8serrors.IsNotFound(err) {
			continue
		}

		if err != nil {
			return errors.Wrap(err)
		}

		endpointsPods, err := r.getEndpointsPods(ctx, endpoints)
		if err != nil {
			return errors.Wrap(err)
		}

		for _, pod := range endpointsPods {
			serviceId, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, &pod)
			if err != nil {
				return errors.Wrap(err)
			}
			podsToReport = append(podsToReport, cloudclient.K8sResourceEligibleForMetricsCollectionInput{Namespace: req.Namespace, Name: serviceId.Name, Kind: serviceId.Kind})
		}
	}

	err = r.reportToCloud(ctx, req.Namespace, cloudclient.EligibleForMetricsCollectionReasonServiceAnnotations, podsToReport)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (r *MetricsCollectionTrafficHandler) reportToCloud(ctx context.Context, namespace string, reason cloudclient.EligibleForMetricsCollectionReason, pods []cloudclient.K8sResourceEligibleForMetricsCollectionInput) error {
	r.reportToCloudLock.Lock()
	defer r.reportToCloudLock.Unlock()

	// Remove duplicates - in case we have multiple pods that indicates on the same workload
	pods = lo.UniqBy(pods, func(item cloudclient.K8sResourceEligibleForMetricsCollectionInput) string {
		return item.Name
	})

	newCacheValue, err := r.cache.GenerateValue(pods)
	if err != nil {
		return errors.Wrap(err)
	}

	currentCacheValue, found := r.cache.Get(namespace, reason)
	if found && bytes.Equal(currentCacheValue, newCacheValue) {
		// current cache value is same as the new one, no need to report
		return nil
	}

	err = r.otterizeCloud.ReportK8sResourceEligibleForMetricsCollection(ctx, namespace, reason, pods)
	if err != nil {
		return errors.Wrap(err)
	}

	r.cache.Set(namespace, reason, newCacheValue)

	return nil
}

func (r *MetricsCollectionTrafficHandler) getEndpointsPods(ctx context.Context, endpoints *corev1.Endpoints) ([]corev1.Pod, error) {
	addresses := make([]corev1.EndpointAddress, 0)
	for _, subset := range endpoints.Subsets {
		addresses = append(addresses, subset.Addresses...)
		addresses = append(addresses, subset.NotReadyAddresses...)
	}

	pods := make([]corev1.Pod, 0)
	for _, address := range addresses {
		if address.TargetRef == nil || address.TargetRef.Kind != "Pod" {
			// If we could not find the relevant pod, we just skip to the next one
			continue
		}

		pod := &corev1.Pod{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: address.TargetRef.Name, Namespace: address.TargetRef.Namespace}, pod)
		if k8serrors.IsNotFound(err) {
			continue
		}

		if err != nil {
			return make([]corev1.Pod, 0), errors.Wrap(err)
		}

		pods = append(pods, *pod)
	}
	return pods, nil
}
