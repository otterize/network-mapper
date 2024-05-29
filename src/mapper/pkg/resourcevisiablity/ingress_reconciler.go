package resourcevisiablity

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/injectablerecorder"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type IngressReconciler struct {
	client.Client
	injectablerecorder.InjectableRecorder
	otterizeCloud cloudclient.CloudClient
}

func NewIngressReconciler(client client.Client, otterizeCloudClient cloudclient.CloudClient) *IngressReconciler {
	return &IngressReconciler{
		Client:        client,
		otterizeCloud: otterizeCloudClient,
	}
}

func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	recorder := mgr.GetEventRecorderFor("intents-operator")
	r.InjectRecorder(recorder)

	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *IngressReconciler) InjectRecorder(recorder record.EventRecorder) {
	r.Recorder = recorder
}

func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	return ctrl.Result{}, nil
}
