package labelreporter

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/nilable"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type NamespaceReconciler struct {
	client.Client
	cloudClient cloudclient.CloudClient
}

func NewNamespaceReconciler(client client.Client) *NamespaceReconciler {
	return &NamespaceReconciler{
		Client: client,
	}
}

func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	namespace := &corev1.Namespace{}
	err := r.Get(ctx, req.NamespacedName, namespace)
	if err != nil && client.IgnoreNotFound(err) == nil {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	if len(namespace.Labels) == 0 {
		return ctrl.Result{}, nil
	}

	labels := make([]cloudclient.LabelInput, 0, len(namespace.Labels))
	for key, value := range namespace.Labels {
		labels = append(labels, cloudclient.LabelInput{Key: key, Value: nilable.From(value)})
	}

	err = r.cloudClient.ReportNamespaceLabels(ctx, namespace.Name, labels)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}
	return ctrl.Result{}, nil
}
