package metrics_collection_traffic

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/injectablerecorder"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type EndpointsReconciler struct {
	client.Client
	injectablerecorder.InjectableRecorder
	metricsCollectionTrafficHandler *MetricsCollectionTrafficHandler
}

func NewEndpointsReconciler(metricsCollectionTrafficHandler *MetricsCollectionTrafficHandler) *EndpointsReconciler {
	return &EndpointsReconciler{
		metricsCollectionTrafficHandler: metricsCollectionTrafficHandler,
	}
}

func (r *EndpointsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	recorder := mgr.GetEventRecorderFor("intents-operator")
	r.InjectRecorder(recorder)

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Endpoints{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *EndpointsReconciler) InjectRecorder(recorder record.EventRecorder) {
	r.Recorder = recorder
}

func (r *EndpointsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := r.metricsCollectionTrafficHandler.HandleAllServicesInNamespace(ctx, req)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	return ctrl.Result{}, nil
}
