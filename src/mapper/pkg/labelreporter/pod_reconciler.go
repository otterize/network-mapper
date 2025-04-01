package labelreporter

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sync"
	"time"
)

const (
	cacheTTL  = 5 * time.Hour
	cacheSize = 1000
)

type PodReconciler struct {
	client.Client
	cloudClient       cloudclient.CloudClient
	serviceIDResolver serviceidresolver.ServiceResolver
	cache             *serviceIdLabelsCache
	once              sync.Once
}

func NewPodReconciler(client client.Client, cloudClient cloudclient.CloudClient, resolver serviceidresolver.ServiceResolver) *PodReconciler {
	return &PodReconciler{
		Client:            client,
		cloudClient:       cloudClient,
		serviceIDResolver: resolver,
		cache:             newServiceIdLabelsCache(cacheSize, cacheTTL),
	}
}

func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.once.Do(func() {
		// Sync all pods in batches per namespaces
		// This way we will avoid hitting the API server with too many requests
		err := r.syncOnceAllPods(ctx)
		if err != nil {
			logrus.WithError(err).Warnf("failed to report all pod labels, will continue with individual pod reconciliation")
		}
	})
	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)
	if err != nil && client.IgnoreNotFound(err) == nil {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	if pod.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	serviceIdentity, err := r.serviceIDResolver.ResolvePodToServiceIdentity(ctx, pod)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	err = r.reportWorkloadLabelsWithCache(ctx, serviceIdentity, pod.Labels)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	return ctrl.Result{}, nil
}

func (r *PodReconciler) reportWorkloadLabelsWithCache(ctx context.Context, serviceIdentity serviceidentity.ServiceIdentity, labels map[string]string) error {
	svcIDKey, labelVal := serviceIdentityToCacheKey(serviceIdentity), checksumLabels(labels)
	cached := r.cache.IsCached(svcIDKey, labelVal)
	if cached {
		return nil
	}

	err := r.reportWorkloadLabels(ctx, serviceIdentity, labels)
	if err != nil {
		return errors.Wrap(err)
	}

	r.cache.Add(serviceIdentityToCacheKey(serviceIdentity), checksumLabels(labels))
	return nil
}

func (r *PodReconciler) reportWorkloadLabels(ctx context.Context, serviceIdentity serviceidentity.ServiceIdentity, labels map[string]string) error {
	serviceIdentityInput := serviceIdentityToServiceIdentityInput(serviceIdentity)

	labelsInput := labelsToLabelInput(labels)

	workloadLabelInput := cloudclient.ReportServiceMetadataInput{
		Identity: serviceIdentityInput,
		Metadata: cloudclient.ServiceMetadataInput{Labels: labelsInput},
	}

	return errors.Wrap(r.cloudClient.ReportWorkloadsLabels(ctx, []cloudclient.ReportServiceMetadataInput{workloadLabelInput}))
}
