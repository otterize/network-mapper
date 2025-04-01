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

type ServiceReconciler struct {
	client.Client
	injectablerecorder.InjectableRecorder
	metricsCollectionTrafficHandler *MetricsCollectionTrafficHandler
}

func NewServiceReconciler(metricsCollectionTrafficHandler *MetricsCollectionTrafficHandler) *ServiceReconciler {
	return &ServiceReconciler{
		metricsCollectionTrafficHandler: metricsCollectionTrafficHandler,
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
	err := r.metricsCollectionTrafficHandler.HandleAllServicesInNamespace(ctx, req)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	return ctrl.Result{}, nil
}
