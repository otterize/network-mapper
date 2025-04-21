package metadatareporter

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type EndpointsReconciler struct {
	client.Client
	serviceIDResolver serviceidresolver.ServiceResolver
	metadataReporter  *MetadataReporter
}

func NewEndpointsReconciler(client client.Client, resolver serviceidresolver.ServiceResolver, reporter *MetadataReporter) *EndpointsReconciler {
	return &EndpointsReconciler{
		Client:            client,
		serviceIDResolver: resolver,
		metadataReporter:  reporter,
	}
}

func (r *EndpointsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Endpoints{}).
		Watches(&corev1.Service{}, handler.EnqueueRequestsFromMapFunc(r.mapServicesToEndpoints)).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *EndpointsReconciler) mapServicesToEndpoints(_ context.Context, obj client.Object) []reconcile.Request {
	service := obj.(*corev1.Service)
	logrus.Debugf("Enqueueing endpoints for service %s", service.Name)

	return []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: service.GetNamespace(), Name: service.GetName()}}}
}

func (r *EndpointsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	endpoints := &corev1.Endpoints{}
	err := r.Get(ctx, req.NamespacedName, endpoints)
	if err != nil && client.IgnoreNotFound(err) == nil {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	if endpoints.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	podNames := r.getPodNamesFromEndpoints(endpoints)
	serviceIdentities := make(map[string]serviceidentity.ServiceIdentity)
	for _, podName := range podNames {
		pod := &corev1.Pod{}
		err := r.Get(ctx, client.ObjectKey{Namespace: endpoints.Namespace, Name: podName}, pod)
		if err != nil && client.IgnoreNotFound(err) == nil {
			return ctrl.Result{}, nil
		}
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err)
		}
		serviceIdentity, err := r.serviceIDResolver.ResolvePodToServiceIdentity(ctx, pod)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err)
		}
		serviceIdentities[serviceIdentity.GetNameWithKind()] = serviceIdentity
	}

	if len(serviceIdentities) == 0 {
		return ctrl.Result{}, nil
	}

	err = r.metadataReporter.ReportMetadata(ctx, lo.Values(serviceIdentities))
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	return ctrl.Result{}, nil
}

func (r *EndpointsReconciler) getPodNamesFromEndpoints(endpoints *corev1.Endpoints) []string {
	podNames := make([]string, 0)
	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			if address.TargetRef != nil && address.TargetRef.Kind == "Pod" {
				podNames = append(podNames, address.TargetRef.Name)
			}
		}
		for _, address := range subset.NotReadyAddresses {
			if address.TargetRef != nil && address.TargetRef.Kind == "Pod" {
				podNames = append(podNames, address.TargetRef.Name)
			}
		}
	}
	return podNames
}
