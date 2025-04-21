package metadatareporter

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Setup(client client.Client, cloudClient cloudclient.CloudClient, resolver serviceidresolver.ServiceResolver, mgr ctrl.Manager) error {
	reporter := NewMetadataReporter(client, cloudClient, resolver)

	// Initialize the EndpointsReconciler
	endpointsReconciler := NewEndpointsReconciler(client, resolver, reporter)
	if err := endpointsReconciler.SetupWithManager(mgr); err != nil {
		return errors.Wrap(err)
	}

	// Initialize the PodReconciler
	podReconciler := NewPodReconciler(client, resolver, reporter)
	if err := podReconciler.SetupWithManager(mgr); err != nil {
		return errors.Wrap(err)
	}

	// Initialize the NamespaceReconciler
	namespaceReconciler := NewNamespaceReconciler(mgr.GetClient(), cloudClient)
	if err := namespaceReconciler.SetupWithManager(mgr); err != nil {
		return errors.Wrap(err)
	}

	// Initialize indexes
	if err := initIndexes(mgr); err != nil {
		return errors.Wrap(err)
	}

	return nil
}
func initIndexes(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&corev1.Endpoints{},
		endpointsPodNamesIndexField,
		func(object client.Object) []string {
			var res []string
			endpoints := object.(*corev1.Endpoints)
			addresses := make([]corev1.EndpointAddress, 0)
			for _, subset := range endpoints.Subsets {
				addresses = append(addresses, subset.Addresses...)
				addresses = append(addresses, subset.NotReadyAddresses...)
			}

			for _, address := range addresses {
				if address.TargetRef == nil || address.TargetRef.Kind != "Pod" {
					continue
				}

				res = append(res, address.TargetRef.Name)
			}

			return res
		}); err != nil {
		return errors.Wrap(err)
	}
	return nil
}
