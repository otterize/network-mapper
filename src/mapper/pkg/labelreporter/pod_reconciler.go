package labelreporter

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/nilable"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
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

func (r *PodReconciler) syncOnceAllPods(ctx context.Context) error {
	allNamespaces := &corev1.NamespaceList{}
	err := r.List(ctx, allNamespaces)
	if err != nil {
		return errors.Wrap(err)
	}
	for _, namespace := range allNamespaces.Items {
		err := r.syncPodsInNamespace(ctx, namespace.Name)
		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func (r *PodReconciler) syncPodsInNamespace(ctx context.Context, namespace string) error {
	pods := &corev1.PodList{}
	err := r.List(ctx, pods, client.InNamespace(namespace))
	if err != nil {
		return errors.Wrap(err)
	}

	serviceIdentityToReportInput := make(map[serviceIdentityKey]cloudclient.ReportServiceMetadataInput)

	for _, pod := range pods.Items {
		serviceIdentity, err := r.serviceIDResolver.ResolvePodToServiceIdentity(ctx, &pod)
		if err != nil {
			return errors.Wrap(err)
		}
		if _, ok := serviceIdentityToReportInput[serviceIdentityToCacheKey(serviceIdentity)]; ok {
			continue
		}
		identityInput := serviceIdentityToServiceIdentityInput(serviceIdentity)
		labelsInput := r.labelsToLabelInput(pod.Labels)
		input := cloudclient.ReportServiceMetadataInput{
			Identity: identityInput,
			Metadata: cloudclient.ServiceMetadataInput{Labels: labelsInput},
		}
		serviceIdentityToReportInput[serviceIdentityToCacheKey(serviceIdentity)] = input
	}

	inputs := lo.Values(serviceIdentityToReportInput)
	slices.SortFunc(inputs, func(a, b cloudclient.ReportServiceMetadataInput) bool {
		return a.Identity.Name < b.Identity.Name
	})

	err = r.cloudClient.ReportWorkloadsLabels(ctx, inputs)
	if err != nil {
		return errors.Wrap(err)
	}

	for key, input := range serviceIdentityToReportInput {
		labels := lo.SliceToMap(input.Metadata.Labels, func(label cloudclient.LabelInput) (string, string) {
			return label.Key, label.Value.Item
		})
		r.cache.Add(key, checksumLabels(labels))
	}

	return nil
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

	labelsInput := r.labelsToLabelInput(labels)

	workloadLabelInput := cloudclient.ReportServiceMetadataInput{
		Identity: serviceIdentityInput,
		Metadata: cloudclient.ServiceMetadataInput{Labels: labelsInput},
	}

	return errors.Wrap(r.cloudClient.ReportWorkloadsLabels(ctx, []cloudclient.ReportServiceMetadataInput{workloadLabelInput}))
}

func (r *PodReconciler) labelsToLabelInput(labels map[string]string) []cloudclient.LabelInput {
	labelsInput := make([]cloudclient.LabelInput, 0)
	for key, value := range labels {
		labelsInput = append(labelsInput, cloudclient.LabelInput{Key: key, Value: nilable.From(value)})
	}

	slices.SortFunc(labelsInput, func(a, b cloudclient.LabelInput) bool {
		return a.Key < b.Key
	})
	return labelsInput
}

func serviceIdentityToCacheKey(identity serviceidentity.ServiceIdentity) serviceIdentityKey {
	return serviceIdentityKey(identity.GetNameWithKind())
}
