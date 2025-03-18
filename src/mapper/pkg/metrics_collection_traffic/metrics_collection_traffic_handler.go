package metrics_collection_traffic

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MetricsCollectionTrafficHandler struct {
	client.Client
	serviceIdResolver *serviceidresolver.Resolver
	otterizeCloud     cloudclient.CloudClient
}

func NewMetricsCollectionTrafficHandler(client client.Client, serviceIdResolver *serviceidresolver.Resolver, otterizeCloud cloudclient.CloudClient) *MetricsCollectionTrafficHandler {
	return &MetricsCollectionTrafficHandler{
		Client:            client,
		serviceIdResolver: serviceIdResolver,
		otterizeCloud:     otterizeCloud,
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

	// Remove duplicates - in case we have multiple pods that indicates on the same workload
	podsToReport = lo.UniqBy(podsToReport, func(item cloudclient.K8sResourceEligibleForMetricsCollectionInput) string {
		return item.Name
	})

	// TODO: Add cache and report to cloud only if something changed

	err = r.otterizeCloud.ReportK8sResourceEligibleForMetricsCollection(ctx, req.Namespace, cloudclient.EligibleForMetricsCollectionReasonPodAnnotations, podsToReport)
	if err != nil {
		return errors.Wrap(err)
	}
	return nil
}
