package metadatareporter

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *MetadataReporter) syncAllOnce(ctx context.Context) error {
	allNamespaces := &corev1.NamespaceList{}
	err := r.List(ctx, allNamespaces)
	if err != nil {
		return errors.Wrap(err)
	}
	for _, namespace := range allNamespaces.Items {
		err := r.syncNamespace(ctx, namespace.Name)
		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func (r *MetadataReporter) syncNamespace(ctx context.Context, namespace string) error {
	pods := &corev1.PodList{}
	err := r.List(ctx, pods, client.InNamespace(namespace))
	if err != nil {
		return errors.Wrap(err)
	}

	serviceIdentityToReportInput := make(map[serviceIdentityKey]serviceidentity.ServiceIdentity)

	for _, pod := range pods.Items {
		serviceIdentity, err := r.serviceIDResolver.ResolvePodToServiceIdentity(ctx, &pod)
		if err != nil {
			return errors.Wrap(err)
		}
		if _, ok := serviceIdentityToReportInput[serviceIdentityToCacheKey(serviceIdentity)]; ok {
			// For multi-pod workloads, we only need to report the metadata once
			continue
		}
		serviceIdentityToReportInput[serviceIdentityToCacheKey(serviceIdentity)] = serviceIdentity
	}

	identities := lo.Values(serviceIdentityToReportInput)
	return errors.Wrap(r.reportWorkloadMetadataWithCache(ctx, identities))
}
