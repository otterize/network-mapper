package resourcevisiablity

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/injectablerecorder"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type ServiceReconciler struct {
	client.Client
	injectablerecorder.InjectableRecorder
	otterizeCloud cloudclient.CloudClient
}

func NewServiceReconciler(client client.Client, otterizeCloudClient cloudclient.CloudClient) *ServiceReconciler {
	return &ServiceReconciler{
		Client:        client,
		otterizeCloud: otterizeCloudClient,
	}
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	recorder := mgr.GetEventRecorderFor("intents-operator")
	r.InjectRecorder(recorder)

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *ServiceReconciler) InjectRecorder(recorder record.EventRecorder) {
	r.Recorder = recorder
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	return ctrl.Result{}, nil
}
