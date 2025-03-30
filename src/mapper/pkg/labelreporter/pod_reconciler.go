package labelreporter

import (
	"context"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/nilable"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	"hash/crc32"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sort"
	"sync"
)

type serviceIdentityKey string
type labelsChecksum uint32

type PodReconciler struct {
	client.Client
	cloudClient       cloudclient.CloudClient
	serviceIDResolver serviceidresolver.ServiceResolver
	cache             *lru.Cache[serviceIdentityKey, labelsChecksum]
	once              sync.Once
}

func NewPodReconciler(client client.Client, cloudClient cloudclient.CloudClient, resolver serviceidresolver.ServiceResolver) (*PodReconciler, error) {
	cache, err := lru.New[serviceIdentityKey, labelsChecksum](1000)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return &PodReconciler{
		Client:            client,
		cloudClient:       cloudClient,
		serviceIDResolver: resolver,
		cache:             cache,
	}, nil
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
		labelsInput := make([]cloudclient.LabelInput, 0)
		for key, value := range pod.Labels {
			labelsInput = append(labelsInput, cloudclient.LabelInput{Key: key, Value: nilable.From(value)})
		}
		input := cloudclient.ReportServiceMetadataInput{
			Identity: identityInput,
			Metadata: cloudclient.ServiceMetadataInput{Labels: labelsInput},
		}
		serviceIdentityToReportInput[serviceIdentityToCacheKey(serviceIdentity)] = input
	}

	err = r.cloudClient.ReportWorkloadsLabels(ctx, lo.Values(serviceIdentityToReportInput))
	if err != nil {
		return errors.Wrap(err)
	}

	for key, input := range serviceIdentityToReportInput {
		labels := lo.SliceToMap(input.Metadata.Labels, func(label cloudclient.LabelInput) (string, string) {
			return label.Key, label.Value.Item
		})
		err := r.addToCache(key, r.checksumLabels(labels))
		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func (r *PodReconciler) reportWorkloadLabelsWithCache(ctx context.Context, serviceIdentity serviceidentity.ServiceIdentity, labels map[string]string) error {
	cached, err := r.isCached(serviceIdentity, labels)
	if err != nil {
		return errors.Wrap(err)
	}
	if cached {
		return nil
	}

	err = r.reportWorkloadLabels(ctx, serviceIdentity, labels)
	if err != nil {
		return errors.Wrap(err)
	}

	err = r.addToCache(serviceIdentityToCacheKey(serviceIdentity), r.checksumLabels(labels))
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (r *PodReconciler) reportWorkloadLabels(ctx context.Context, serviceIdentity serviceidentity.ServiceIdentity, labels map[string]string) error {
	serviceIdentityInput := serviceIdentityToServiceIdentityInput(serviceIdentity)

	labelsInput := make([]cloudclient.LabelInput, 0)
	for key, value := range labels {
		labelsInput = append(labelsInput, cloudclient.LabelInput{Key: key, Value: nilable.From(value)})
	}

	slices.SortFunc(labelsInput, func(a, b cloudclient.LabelInput) bool {
		return a.Key < b.Key
	})

	workloadLabelInput := cloudclient.ReportServiceMetadataInput{
		Identity: serviceIdentityInput,
		Metadata: cloudclient.ServiceMetadataInput{Labels: labelsInput},
	}

	return errors.Wrap(r.cloudClient.ReportWorkloadsLabels(ctx, []cloudclient.ReportServiceMetadataInput{workloadLabelInput}))
}

func (r *PodReconciler) isCached(serviceIdentity serviceidentity.ServiceIdentity, labels map[string]string) (bool, error) {
	serviceIdentityKey := serviceIdentityToCacheKey(serviceIdentity)

	labelHash := r.checksumLabels(labels)

	val, found := r.cache.Get(serviceIdentityKey)

	if found && val == labelHash {
		return true, nil
	}

	return false, nil
}

func (r *PodReconciler) checksumLabels(labels map[string]string) labelsChecksum {
	labelKeys := make([]string, 0, len(labels))
	for key := range labels {
		labelKeys = append(labelKeys, key)
	}
	sort.Strings(labelKeys)
	labelString := ""
	for _, key := range labelKeys {
		labelString += key + labels[key]
	}

	labelHash := crc32.ChecksumIEEE([]byte(labelString))
	return labelsChecksum(labelHash)
}

func (r *PodReconciler) addToCache(key serviceIdentityKey, checksum labelsChecksum) error {
	r.cache.Add(key, checksum)
	return nil
}

func serviceIdentityToServiceIdentityInput(identity serviceidentity.ServiceIdentity) cloudclient.ServiceIdentityInput {
	wi := cloudclient.ServiceIdentityInput{
		Namespace: identity.Namespace,
		Name:      identity.Name,
		Kind:      identity.Kind,
	}
	if identity.ResolvedUsingOverrideAnnotation != nil {
		wi.NameResolvedUsingAnnotation = nilable.From(*identity.ResolvedUsingOverrideAnnotation)
	}

	return wi
}

func serviceIdentityToCacheKey(identity serviceidentity.ServiceIdentity) serviceIdentityKey {
	return serviceIdentityKey(identity.GetNameWithKind())
}
