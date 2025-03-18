package metrics_collection_traffic

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/injectablerecorder"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type PodReconciler struct {
	client.Client
	injectablerecorder.InjectableRecorder
	serviceIdResolver *serviceidresolver.Resolver
	otterizeCloud     cloudclient.CloudClient
}

func NewPodReconciler(client client.Client, serviceIdResolver *serviceidresolver.Resolver, otterizeCloud cloudclient.CloudClient) *PodReconciler {
	return &PodReconciler{
		Client:            client,
		serviceIdResolver: serviceIdResolver,
		otterizeCloud:     otterizeCloud,
	}
}

func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	recorder := mgr.GetEventRecorderFor("intents-operator")
	r.InjectRecorder(recorder)

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *PodReconciler) InjectRecorder(recorder record.EventRecorder) {
	r.Recorder = recorder
}

func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	podList := &corev1.PodList{}
	err := r.Client.List(ctx, podList, client.InNamespace(req.Namespace))
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	scrapePods := lo.Filter(podList.Items, func(pod corev1.Pod, _ int) bool {
		return pod.Annotations["prometheus.io/scrape"] == "true"

	})

	podsToReport := make([]cloudclient.K8sResourceEligibleForMetricsCollectionInput, 0)
	for _, pod := range scrapePods {
		serviceId, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, &pod)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err)
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
		return ctrl.Result{}, errors.Wrap(err)
	}
	return ctrl.Result{}, nil
}
