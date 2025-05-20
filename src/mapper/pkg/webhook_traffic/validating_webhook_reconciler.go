package webhook_traffic

import (
	"context"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/injectablerecorder"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"time"
)

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch

type KubeFinder interface {
	ResolveOtterizeIdentityForService(ctx context.Context, service *corev1.Service, now time.Time) (model.OtterizeServiceIdentity, bool, error)
}

type ValidatingWebhookReconciler struct {
	client.Client
	injectablerecorder.InjectableRecorder
	kubeFinder          KubeFinder
	otterizeCloudClient cloudclient.CloudClient
}

func NewValidatingWebhookReconciler(client client.Client, otterizeCloudClient cloudclient.CloudClient, kubeFinder KubeFinder) *ValidatingWebhookReconciler {
	return &ValidatingWebhookReconciler{Client: client, kubeFinder: kubeFinder, otterizeCloudClient: otterizeCloudClient}
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
	validatingWebhookConfigurationList := &admissionv1.ValidatingWebhookConfigurationList{}
	err := r.Client.List(ctx, validatingWebhookConfigurationList)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	validatingWebhookServices := make([]cloudclient.K8sWebhookServiceInput, 0)
	for _, webhookConfiguration := range validatingWebhookConfigurationList.Items {
		for _, webhook := range webhookConfiguration.Webhooks {
			if webhook.ClientConfig.Service != nil {
				service := corev1.Service{}
				err := r.Client.Get(ctx, types.NamespacedName{Name: webhook.ClientConfig.Service.Name, Namespace: webhook.ClientConfig.Service.Namespace}, &service)
				if k8sErr := &(k8serrors.StatusError{}); errors.As(err, &k8sErr) {
					if k8serrors.IsNotFound(k8sErr) {
						continue
					}
				}

				if err != nil {
					return ctrl.Result{}, errors.Wrap(err)
				}

				identity, found, err := r.kubeFinder.ResolveOtterizeIdentityForService(ctx, &service, time.Now())
				if err != nil {
					return ctrl.Result{}, errors.Wrap(err)
				}

				if !found {
					continue
				}

				validatingWebhookServices = append(validatingWebhookServices, cloudclient.K8sWebhookServiceInput{
					OtterizeName: identity.Name,
					ServiceName:  webhook.ClientConfig.Service.Name,
					Namespace:    webhook.ClientConfig.Service.Namespace,
					WebhookName:  webhookConfiguration.Name,
					WebhookType:  cloudclient.WebhookTypeValidatingWebhook,
				})
			}
		}
	}

	validatingWebhookServices = lo.UniqBy(validatingWebhookServices, func(service cloudclient.K8sWebhookServiceInput) string {
		return fmt.Sprintf("%s#%s#%s#%s", service.Namespace, service.ServiceName, service.WebhookName, service.WebhookType)

	})
	err = r.otterizeCloudClient.ReportK8sWebhookServices(ctx, validatingWebhookServices)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	return ctrl.Result{}, nil
}
