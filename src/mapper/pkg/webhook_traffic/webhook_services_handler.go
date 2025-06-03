package webhook_traffic

import (
	"bytes"
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/nilable"
	"github.com/samber/lo"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

	mutatingWebhookServices, err := h.collectMutatingWebhooksServices(ctx)
	if err != nil {
		return errors.Wrap(err)
	}

	conversionWebhookServices, err := h.collectConversionWebhooksServices(ctx)
	if err != nil {
		return errors.Wrap(err)
	}

	allWebhookServices := append(validatingWebhookServices, mutatingWebhookServices...)
	allWebhookServices = append(allWebhookServices, conversionWebhookServices...)

	err = h.reportToCloud(ctx, allWebhookServices)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (h *WebhookServicesHandler) reportToCloud(ctx context.Context, allWebhookServices []cloudclient.K8sWebhookServiceInput) error {
	// dedup
	allWebhookServices = lo.UniqBy(allWebhookServices, K8sWebhookServiceInputKey)

	newCacheValue, err := h.cache.GenerateValue(allWebhookServices)
	if err != nil {
		return errors.Wrap(err)
	}

	currentCacheValue, found := h.cache.Get()
	if found && bytes.Equal(currentCacheValue, newCacheValue) {
		// current cache value is same as the new one, no need to report
		return nil
	}

	err = h.otterizeCloud.ReportK8sWebhookServices(ctx, allWebhookServices)
	if err != nil {
		return errors.Wrap(err)
	}

	h.cache.Set(newCacheValue)

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
				input, found, err := h.createWebhookServiceInput(ctx, webhookConfiguration.Name, cloudclient.WebhookTypeValidatingWebhook, webhook.ClientConfig.Service.Name, webhook.ClientConfig.Service.Namespace)
				if err != nil {
					return make([]cloudclient.K8sWebhookServiceInput, 0), errors.Wrap(err)
				}

				if found {
					validatingWebhookServices = append(validatingWebhookServices, input)
				}
			}
		}
	}

	return validatingWebhookServices, nil
}

func (h *WebhookServicesHandler) collectMutatingWebhooksServices(ctx context.Context) ([]cloudclient.K8sWebhookServiceInput, error) {
	mutatingWebhookConfigurationList := &admissionv1.MutatingWebhookConfigurationList{}
	err := h.Client.List(ctx, mutatingWebhookConfigurationList)
	if err != nil {
		return make([]cloudclient.K8sWebhookServiceInput, 0), errors.Wrap(err)
	}

	mutatingWebhookServices := make([]cloudclient.K8sWebhookServiceInput, 0)
	for _, webhookConfiguration := range mutatingWebhookConfigurationList.Items {
		for _, webhook := range webhookConfiguration.Webhooks {
			if webhook.ClientConfig.Service != nil {
				input, found, err := h.createWebhookServiceInput(ctx, webhookConfiguration.Name, cloudclient.WebhookTypeMutatingWebhook, webhook.ClientConfig.Service.Name, webhook.ClientConfig.Service.Namespace)
				if err != nil {
					return make([]cloudclient.K8sWebhookServiceInput, 0), errors.Wrap(err)
				}

				if found {
					mutatingWebhookServices = append(mutatingWebhookServices, input)
				}
			}
		}
	}

	return mutatingWebhookServices, nil
}

func (h *WebhookServicesHandler) collectConversionWebhooksServices(ctx context.Context) ([]cloudclient.K8sWebhookServiceInput, error) {
	crdsList := &apiextensionsv1.CustomResourceDefinitionList{}
	err := h.Client.List(ctx, crdsList)
	if err != nil {
		return make([]cloudclient.K8sWebhookServiceInput, 0), errors.Wrap(err)
	}

	conversionWebhookConfigurationList := lo.Filter(crdsList.Items, func(crd apiextensionsv1.CustomResourceDefinition, _ int) bool {
		return crd.Spec.Conversion != nil && crd.Spec.Conversion.Webhook != nil && crd.Spec.Conversion.Webhook.ClientConfig != nil
	})

	conversionWebhookServices := make([]cloudclient.K8sWebhookServiceInput, 0)
	for _, webhookCRD := range conversionWebhookConfigurationList {
		webhookCRDService := webhookCRD.Spec.Conversion.Webhook.ClientConfig.Service
		if webhookCRDService != nil {
			input, found, err := h.createWebhookServiceInput(ctx, webhookCRD.Name, cloudclient.WebhookTypeConversionWebhook, webhookCRDService.Name, webhookCRDService.Namespace)
			if err != nil {
				return make([]cloudclient.K8sWebhookServiceInput, 0), errors.Wrap(err)
			}

			if found {
				conversionWebhookServices = append(conversionWebhookServices, input)
			}
		}

	}

	return conversionWebhookServices, nil
}

func (h *WebhookServicesHandler) createWebhookServiceInput(ctx context.Context, webhookName string, webhookType cloudclient.WebhookType, name string, namespace string) (cloudclient.K8sWebhookServiceInput, bool, error) {
	identity, found, err := h.getServiceIdentity(ctx, name, namespace)
	if err != nil {
		return cloudclient.K8sWebhookServiceInput{}, false, errors.Wrap(err)
	}

	if !found {
		return cloudclient.K8sWebhookServiceInput{}, false, nil
	}

	input := cloudclient.K8sWebhookServiceInput{
		Identity: cloudclient.ServiceIdentityInput{
			Name:      identity.Name,
			Namespace: namespace,
		},
		WebhookName: webhookName,
		WebhookType: webhookType,
	}

	if identity.PodOwnerKind != nil {
		input.Identity.Kind = identity.PodOwnerKind.Kind
	}

	if identity.NameResolvedUsingAnnotation != nil {
		input.Identity.NameResolvedUsingAnnotation = nilable.From(*identity.NameResolvedUsingAnnotation)
	}

	return input, true, nil
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
