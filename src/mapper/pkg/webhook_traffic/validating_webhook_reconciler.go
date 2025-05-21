package webhook_traffic

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/injectablerecorder"
	"github.com/samber/lo"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch

type ValidatingWebhookReconciler struct {
	client.Client
	injectablerecorder.InjectableRecorder
	handler *WebhookServicesHandler
}

func NewValidatingWebhookReconciler(handler *WebhookServicesHandler) *ValidatingWebhookReconciler {
	return &ValidatingWebhookReconciler{handler: handler}
}

func (r *ValidatingWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	recorder := mgr.GetEventRecorderFor("intents-operator")
	r.InjectRecorder(recorder)

	return ctrl.NewControllerManagedBy(mgr).
		For(&admissionv1.ValidatingWebhookConfiguration{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *ValidatingWebhookReconciler) InjectRecorder(recorder record.EventRecorder) {
	r.Recorder = recorder
}

func (r *ValidatingWebhookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := r.handler.HandleAll(ctx)

	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	return ctrl.Result{}, nil
}
