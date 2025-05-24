package webhook_traffic

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/mocks"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

const (
	validatingWebhookName       = "validating-webhook"
	mutatingWebhookName         = "mutating-webhook"
	conversionWebhookName       = "conversion-webhook"
	serviceName                 = "test-service"
	serviceNamespace            = "test-namespace"
	otterizeServiceIdentityName = "test-service-deployment"
)

var ValidatingWebhook = admissionv1.ValidatingWebhookConfiguration{
	ObjectMeta: metav1.ObjectMeta{Name: validatingWebhookName},
	Webhooks: []admissionv1.ValidatingWebhook{
		{
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Name:      serviceName,
					Namespace: serviceNamespace,
					Port:      lo.ToPtr(int32(1479)),
				},
			},
		},
	},
}

var MutatingWebhook = admissionv1.MutatingWebhookConfiguration{
	ObjectMeta: metav1.ObjectMeta{Name: mutatingWebhookName},
	Webhooks: []admissionv1.MutatingWebhook{
		{
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Name:      serviceName,
					Namespace: serviceNamespace,
					Port:      lo.ToPtr(int32(1479)),
				},
			},
		},
	},
}

var ConversionWebhook = apiextensionsv1.CustomResourceDefinition{
	ObjectMeta: metav1.ObjectMeta{Name: conversionWebhookName},
	Spec: apiextensionsv1.CustomResourceDefinitionSpec{
		Conversion: &apiextensionsv1.CustomResourceConversion{
			Webhook: &apiextensionsv1.WebhookConversion{
				ClientConfig: &apiextensionsv1.WebhookClientConfig{
					Service: &apiextensionsv1.ServiceReference{
						Name:      serviceName,
						Namespace: serviceNamespace,
						Port:      lo.ToPtr(int32(1479)),
					},
				},
			},
		},
	},
}

type WebhookServiceHandlerTestSuite struct {
	suite.Suite
	cloudClient *cloudclientmocks.MockCloudClient
	k8sClient   *mocks.K8sClient
	kubeFinder  *mocks.MockKubeFinder
}

func (s *WebhookServiceHandlerTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.cloudClient = cloudclientmocks.NewMockCloudClient(controller)
	s.k8sClient = mocks.NewK8sClient(controller)
	s.kubeFinder = mocks.NewMockKubeFinder(controller)
}

func (s *WebhookServiceHandlerTestSuite) TearDownTest() {
	s.cloudClient = nil
	s.k8sClient = nil
	s.kubeFinder = nil
}

func (s *WebhookServiceHandlerTestSuite) TestReportingAllWebhooks() {
	webhookServiceHandler := NewWebhookServicesHandler(s.k8sClient, s.cloudClient, s.kubeFinder)

	s.mockValidatingWebhookServices([]admissionv1.ValidatingWebhookConfiguration{ValidatingWebhook})
	s.mockResolvingWebhookService(serviceName, serviceNamespace, otterizeServiceIdentityName)
	s.mockMutatingWebhookServices([]admissionv1.MutatingWebhookConfiguration{MutatingWebhook})
	s.mockResolvingWebhookService(serviceName, serviceNamespace, otterizeServiceIdentityName)
	s.mockCRDs([]apiextensionsv1.CustomResourceDefinition{ConversionWebhook})
	s.mockResolvingWebhookService(serviceName, serviceNamespace, otterizeServiceIdentityName)

	expected := []cloudclient.K8sWebhookServiceInput{
		{
			Namespace:   serviceNamespace,
			Name:        otterizeServiceIdentityName,
			Kind:        "",
			WebhookName: ValidatingWebhook.Name,
			WebhookType: cloudclient.WebhookTypeValidatingWebhook,
		},
		{
			Namespace:   serviceNamespace,
			Name:        otterizeServiceIdentityName,
			Kind:        "",
			WebhookName: MutatingWebhook.Name,
			WebhookType: cloudclient.WebhookTypeMutatingWebhook,
		},
		{
			Namespace:   serviceNamespace,
			Name:        otterizeServiceIdentityName,
			Kind:        "",
			WebhookName: ConversionWebhook.Name,
			WebhookType: cloudclient.WebhookTypeConversionWebhook,
		},
	}
	s.cloudClient.EXPECT().ReportK8sWebhookServices(gomock.Any(), gomock.Eq(expected)).Return(nil)

	err := webhookServiceHandler.HandleAll(context.Background())
	s.Require().NoError(err)
}

func (s *WebhookServiceHandlerTestSuite) TestReportingAllWebhooks_SameWebhookNameDifferentTypes() {
	webhookServiceHandler := NewWebhookServicesHandler(s.k8sClient, s.cloudClient, s.kubeFinder)

	s.mockValidatingWebhookServices([]admissionv1.ValidatingWebhookConfiguration{ValidatingWebhook})
	s.mockResolvingWebhookService(serviceName, serviceNamespace, otterizeServiceIdentityName)
	mutatingWebhookCopy := MutatingWebhook.DeepCopy()
	mutatingWebhookCopy.Name = ValidatingWebhook.Name
	s.mockMutatingWebhookServices([]admissionv1.MutatingWebhookConfiguration{*mutatingWebhookCopy})
	s.mockResolvingWebhookService(serviceName, serviceNamespace, otterizeServiceIdentityName)
	s.mockCRDs([]apiextensionsv1.CustomResourceDefinition{})

	expected := []cloudclient.K8sWebhookServiceInput{
		{
			Namespace:   serviceNamespace,
			Name:        otterizeServiceIdentityName,
			Kind:        "",
			WebhookName: ValidatingWebhook.Name,
			WebhookType: cloudclient.WebhookTypeValidatingWebhook,
		},
		{
			Namespace:   serviceNamespace,
			Name:        otterizeServiceIdentityName,
			Kind:        "",
			WebhookName: ValidatingWebhook.Name,
			WebhookType: cloudclient.WebhookTypeMutatingWebhook,
		},
	}
	s.cloudClient.EXPECT().ReportK8sWebhookServices(gomock.Any(), gomock.Eq(expected)).Return(nil)

	err := webhookServiceHandler.HandleAll(context.Background())
	s.Require().NoError(err)
}

func (s *WebhookServiceHandlerTestSuite) TestReportingAllWebhooks_SameServiceTwiceForTheSameWebhook() {
	webhookServiceHandler := NewWebhookServicesHandler(s.k8sClient, s.cloudClient, s.kubeFinder)

	validatingWebhookCopy := ValidatingWebhook.DeepCopy()
	validatingWebhookCopy.Webhooks = append(validatingWebhookCopy.Webhooks, validatingWebhookCopy.Webhooks...)
	s.mockValidatingWebhookServices([]admissionv1.ValidatingWebhookConfiguration{*validatingWebhookCopy})
	s.mockResolvingWebhookService(serviceName, serviceNamespace, otterizeServiceIdentityName)
	s.mockResolvingWebhookService(serviceName, serviceNamespace, otterizeServiceIdentityName)
	s.mockMutatingWebhookServices([]admissionv1.MutatingWebhookConfiguration{})
	s.mockCRDs([]apiextensionsv1.CustomResourceDefinition{})

	expected := []cloudclient.K8sWebhookServiceInput{
		{
			Namespace:   serviceNamespace,
			Name:        otterizeServiceIdentityName,
			Kind:        "",
			WebhookName: ValidatingWebhook.Name,
			WebhookType: cloudclient.WebhookTypeValidatingWebhook,
		},
	}
	s.cloudClient.EXPECT().ReportK8sWebhookServices(gomock.Any(), gomock.Eq(expected)).Return(nil)

	err := webhookServiceHandler.HandleAll(context.Background())
	s.Require().NoError(err)
}

func (s *WebhookServiceHandlerTestSuite) mockResolvingWebhookService(serviceName string, serviceNamespace string, otterizeName string) {

	mockService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: serviceNamespace,
		},
	}

	s.k8sClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: serviceName, Namespace: serviceNamespace}, gomock.Eq(&corev1.Service{})).
		DoAndReturn(
			func(ctx context.Context, key client.ObjectKey, service *corev1.Service, _ ...any) error {
				*service = *mockService
				return nil
			})

	mockServiceIdentity := model.OtterizeServiceIdentity{
		Name:              otterizeName,
		Namespace:         serviceNamespace,
		PodOwnerKind:      nil,
		KubernetesService: lo.ToPtr(serviceNamespace),
	}
	s.kubeFinder.EXPECT().ResolveOtterizeIdentityForService(gomock.Any(), gomock.Eq(mockService), gomock.Any()).Return(mockServiceIdentity, true, nil)
}

func (s *WebhookServiceHandlerTestSuite) mockValidatingWebhookServices(webhooks []admissionv1.ValidatingWebhookConfiguration) {
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&admissionv1.ValidatingWebhookConfigurationList{})).DoAndReturn(
		func(ctx context.Context, list *admissionv1.ValidatingWebhookConfigurationList, opts ...client.ListOption) error {
			list.Items = webhooks
			return nil
		})
}

func (s *WebhookServiceHandlerTestSuite) mockMutatingWebhookServices(webhooks []admissionv1.MutatingWebhookConfiguration) {
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&admissionv1.MutatingWebhookConfigurationList{})).DoAndReturn(
		func(ctx context.Context, list *admissionv1.MutatingWebhookConfigurationList, opts ...client.ListOption) error {
			list.Items = webhooks
			return nil
		})
}

func (s *WebhookServiceHandlerTestSuite) mockCRDs(crds []apiextensionsv1.CustomResourceDefinition) {
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&apiextensionsv1.CustomResourceDefinitionList{})).DoAndReturn(
		func(ctx context.Context, list *apiextensionsv1.CustomResourceDefinitionList, opts ...client.ListOption) error {
			list.Items = crds
			return nil
		})
}

func TestWebhookServiceHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(WebhookServiceHandlerTestSuite))
}
