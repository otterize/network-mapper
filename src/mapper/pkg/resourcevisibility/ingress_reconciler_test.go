package resourcevisibility

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/mocks"
	"github.com/otterize/nilable"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

type IngressVisibilityTestSuite struct {
	suite.Suite
	cloudClient *cloudclientmocks.MockCloudClient
	k8sClient   *mocks.K8sClient
	reconciler  *IngressReconciler
}

func (s *IngressVisibilityTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.cloudClient = cloudclientmocks.NewMockCloudClient(controller)
	s.k8sClient = mocks.NewK8sClient(controller)
	s.reconciler = NewIngressReconciler(s.k8sClient, s.cloudClient)
}

func (s *IngressVisibilityTestSuite) TearDownTest() {
	s.cloudClient = nil
	s.k8sClient = nil
	s.reconciler = nil
}

func (s *IngressVisibilityTestSuite) TestIngressUpload() {
	resourceName := "test-ingress"
	ingress := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: testNamespace,
		},
		Spec: networkingv1.IngressSpec{
			DefaultBackend: &networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: "default-backend",
					Port: networkingv1.ServiceBackendPort{
						Number: 9091,
					},
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: "app.otter.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									PathType: lo.ToPtr(networkingv1.PathTypePrefix),
									Path:     "/",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "test-service",
											Port: networkingv1.ServiceBackendPort{
												Name: "http",
											},
										},
									},
								},
								{
									PathType: lo.ToPtr(networkingv1.PathTypePrefix),
									Path:     "/api",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "api-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "food.otter.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									PathType: lo.ToPtr(networkingv1.PathTypeExact),
									Path:     "/send-to-default-backend",
								},
							},
						},
					},
				},
			},
		},
	}

	deletedIngress := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "deleted-ingress",
			Namespace:         testNamespace,
			DeletionTimestamp: &metav1.Time{Time: time.Date(2020, 12, 1, 17, 14, 0, 0, time.UTC)},
		},
		Spec: networkingv1.IngressSpec{
			DefaultBackend: &networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: "default-backend",
					Port: networkingv1.ServiceBackendPort{
						Number: 9091,
					},
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: "app.otter.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									PathType: lo.ToPtr(networkingv1.PathTypePrefix),
									Path:     "/this-want-be-uploaded",
								},
							},
						},
					},
				},
			},
		},
	}

	emptyList := networkingv1.IngressList{}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&emptyList), gomock.Eq(client.InNamespace(testNamespace))).DoAndReturn(
		func(ctx context.Context, list *networkingv1.IngressList, opts ...client.ListOption) error {
			list.Items = []networkingv1.Ingress{ingress, deletedIngress}
			return nil
		})

	ingressInput := cloudclient.K8sIngressInput{
		Namespace: testNamespace,
		Name:      resourceName,
		Ingress: cloudclient.K8sResourceIngressInput{
			Spec: cloudclient.K8sResourceIngressSpecInput{
				DefaultBackend: nilable.From(cloudclient.K8sIngressBackendInput{Service: nilable.From(cloudclient.K8sIngressServiceBackendInput{
					Name: "default-backend",
					Port: cloudclient.ServiceBackendPortInput{
						Number: nilable.From(9091),
					},
				})}),
				Rules: []cloudclient.K8sIngressRuleInput{
					{
						Host: nilable.From("app.otter.com"),
						HttpPaths: []cloudclient.K8sIngressHttpPathInput{
							{
								PathType: nilable.From(cloudclient.PathTypePrefix),
								Path:     nilable.From("/"),
								Backend: cloudclient.K8sIngressBackendInput{
									Service: nilable.From(cloudclient.K8sIngressServiceBackendInput{
										Name: "test-service",
										Port: cloudclient.ServiceBackendPortInput{
											Name: nilable.From("http"),
										},
									}),
								},
							},
							{
								PathType: nilable.From(cloudclient.PathTypePrefix),
								Path:     nilable.From("/api"),
								Backend: cloudclient.K8sIngressBackendInput{
									Service: nilable.From(cloudclient.K8sIngressServiceBackendInput{
										Name: "api-service",
										Port: cloudclient.ServiceBackendPortInput{
											Number: nilable.From(8080),
										},
									}),
								},
							},
						},
					},
					{
						Host: nilable.From("food.otter.com"),
						HttpPaths: []cloudclient.K8sIngressHttpPathInput{
							{
								PathType: nilable.From(cloudclient.PathTypeExact),
								Path:     nilable.From("/send-to-default-backend"),
							},
						},
					},
				},
			},
		},
	}

	s.cloudClient.EXPECT().ReportK8sIngresses(gomock.Any(), testNamespace, gomock.Any()).DoAndReturn(
		func(ctx context.Context, namespace string, ingresses []cloudclient.K8sIngressInput) error {
			s.Require().Len(ingresses, 1)
			s.Require().Equal(ingressInput, ingresses[0])
			return nil
		})

	req := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: resourceName}}
	res, err := s.reconciler.Reconcile(context.Background(), req)
	s.NoError(err)
	s.Equal(ctrl.Result{}, res)
}

func (s *IngressVisibilityTestSuite) TestEmptyNamespace() {
	emptyList := networkingv1.IngressList{}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&emptyList), gomock.Eq(client.InNamespace(testNamespace))).Return(nil)

	s.cloudClient.EXPECT().ReportK8sIngresses(gomock.Any(), testNamespace, gomock.Eq([]cloudclient.K8sIngressInput{})).Return(nil)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: "deleted-ingress"}}
	res, err := s.reconciler.Reconcile(context.Background(), req)
	s.NoError(err)
	s.Equal(ctrl.Result{}, res)
}

func TestIngressVisibilityTestSuite(t *testing.T) {
	suite.Run(t, new(IngressVisibilityTestSuite))
}
