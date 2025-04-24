package metadatareporter

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type PodReconciler struct {
	client.Client
	serviceIDResolver serviceidresolver.ServiceResolver
	metadataReporter  *MetadataReporter
}

func NewPodReconciler(client client.Client, resolver serviceidresolver.ServiceResolver, reporter *MetadataReporter) *PodReconciler {
	return &PodReconciler{
		Client:            client,
		serviceIDResolver: resolver,
		metadataReporter:  reporter,
	}
}

func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	err = r.metadataReporter.ReportMetadata(ctx, []serviceidentity.ServiceIdentity{serviceIdentity})
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	return ctrl.Result{}, nil
}
