package labelreporter

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
		labelsInput := labelsToLabelInput(pod.Labels)
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
