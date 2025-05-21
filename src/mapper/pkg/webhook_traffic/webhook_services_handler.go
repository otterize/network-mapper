package webhook_traffic

import (
	"context"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type KubeFinder interface {
	ResolveOtterizeIdentityForService(ctx context.Context, service *corev1.Service, now time.Time) (model.OtterizeServiceIdentity, bool, error)
}

type WebhookServicesHandler struct {
	client.Client
	otterizeCloud cloudclient.CloudClient
	kubeFinder    KubeFinder
	cache         *WebhookServicesCache
}

func NewWebhookServicesHandler(client client.Client, otterizeCloud cloudclient.CloudClient, kubeFinder KubeFinder) *WebhookServicesHandler {
	cache := NewWebhookServicesCache()

	return &WebhookServicesHandler{
		Client:        client,
		otterizeCloud: otterizeCloud,
		cache:         cache,
		kubeFinder:    kubeFinder,
	}
}

func (h *WebhookServicesHandler) HandleAll(ctx context.Context) error {
	validatingWebhookServices, err := h.collectValidatingWebhooksServices(ctx)
	if err != nil {
		return errors.Wrap(err)
	}

	err = h.otterizeCloud.ReportK8sWebhookServices(ctx, validatingWebhookServices)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (h *WebhookServicesHandler) collectValidatingWebhooksServices(ctx context.Context) ([]cloudclient.K8sWebhookServiceInput, error) {
	validatingWebhookConfigurationList := &admissionv1.ValidatingWebhookConfigurationList{}
	err := h.Client.List(ctx, validatingWebhookConfigurationList)
	if err != nil {
		return make([]cloudclient.K8sWebhookServiceInput, 0), errors.Wrap(err)
	}

	validatingWebhookServices := make([]cloudclient.K8sWebhookServiceInput, 0)
	for _, webhookConfiguration := range validatingWebhookConfigurationList.Items {
		for _, webhook := range webhookConfiguration.Webhooks {
			if webhook.ClientConfig.Service != nil {

				identity, found, err := h.getServiceIdentity(ctx, webhook.ClientConfig.Service.Name, webhook.ClientConfig.Service.Namespace)
				if err != nil {
					return make([]cloudclient.K8sWebhookServiceInput, 0), errors.Wrap(err)
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

	return validatingWebhookServices, nil
}

func (h *WebhookServicesHandler) getServiceIdentity(ctx context.Context, name string, namespace string) (model.OtterizeServiceIdentity, bool, error) {
	service := corev1.Service{}
	err := h.Client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &service)
	if k8sErr := &(k8serrors.StatusError{}); errors.As(err, &k8sErr) {
		if k8serrors.IsNotFound(k8sErr) {
			return model.OtterizeServiceIdentity{}, false, nil
		}
	}

	if err != nil {
		return model.OtterizeServiceIdentity{}, false, errors.Wrap(err)
	}

	identity, found, err := h.kubeFinder.ResolveOtterizeIdentityForService(ctx, &service, time.Now())
	if err != nil {
		return model.OtterizeServiceIdentity{}, false, errors.Wrap(err)
	}

	if !found {
		return model.OtterizeServiceIdentity{}, false, nil
	}

	return identity, true, nil
}
