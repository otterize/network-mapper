package resourcevisibility

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/mocks"
	"github.com/otterize/nilable"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"

	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

const (
	testNamespace = "test-namespace-a"
)

type ServiceVisibilityTestSuite struct {
	suite.Suite
	cloudClient *cloudclientmocks.MockCloudClient
	k8sClient   *mocks.K8sClient
	reconciler  *ServiceReconciler
	kubeFinder  *mocks.MockKubeFinder
}

func (s *ServiceVisibilityTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.cloudClient = cloudclientmocks.NewMockCloudClient(controller)
	s.k8sClient = mocks.NewK8sClient(controller)
	s.kubeFinder = mocks.NewMockKubeFinder(controller)
	s.reconciler = NewServiceReconciler(s.k8sClient, s.cloudClient, s.kubeFinder)
}

func (s *ServiceVisibilityTestSuite) TearDownTest() {
	s.cloudClient = nil
	s.k8sClient = nil
	s.reconciler = nil
	s.kubeFinder = nil
}

func (s *ServiceVisibilityTestSuite) TestServiceUpload() {
	deploymentName := "my-server"
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: testNamespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": deploymentName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(8080),
				},
			},
		},
	}

	emptyList := corev1.ServiceList{}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&emptyList), gomock.Eq(client.InNamespace(testNamespace))).DoAndReturn(
		func(ctx context.Context, list *corev1.ServiceList, opts ...client.ListOption) error {
			list.Items = []corev1.Service{service}
			return nil
		})

	serviceIdentity := model.OtterizeServiceIdentity{
		Name:              deploymentName,
		Namespace:         testNamespace,
		PodOwnerKind:      nil,
		KubernetesService: lo.ToPtr(service.Name),
	}
	s.kubeFinder.EXPECT().ResolveOtterizeIdentityForService(gomock.Any(), &service, gomock.Any()).Return(serviceIdentity, true, nil)

	// Expect pod listing for TargetNamedPorts
	podList := corev1.PodList{}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&podList), gomock.Any()).DoAndReturn(
		func(ctx context.Context, list *corev1.PodList, opts ...client.ListOption) error {
			list.Items = []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: testNamespace,
						Labels:    map[string]string{"app": deploymentName},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "container-1",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 8080,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			}
			return nil
		})

	serviceInput := cloudclient.K8sServiceInput{
		OtterizeServer: deploymentName,
		Namespace:      testNamespace,
		ResourceName:   service.Name,
		Service: cloudclient.K8sResourceServiceInput{
			Spec: cloudclient.K8sResourceServiceSpecInput{
				Ports: []cloudclient.K8sServicePort{
					{
						Port:       80,
						Protocol:   nilable.From(cloudclient.K8sPortProtocolTcp),
						TargetPort: nilable.From(cloudclient.IntOrStringInput{IntVal: nilable.From(8080), IsInt: true}),
					},
				},
				Selector: []cloudclient.SelectorKeyValueInput{{Key: nilable.From("app"), Value: nilable.From(deploymentName)}},
			},
			TargetNamedPorts: []cloudclient.NamedPortInput{
				{
					Name:     "http",
					Port:     8080,
					Protocol: cloudclient.K8sPortProtocol(corev1.ProtocolTCP),
				},
			},
		},
	}
	s.cloudClient.EXPECT().ReportK8sServices(gomock.Any(), testNamespace, gomock.Any()).DoAndReturn(
		func(ctx context.Context, namespace string, services []cloudclient.K8sServiceInput) error {
			s.Require().Len(services, 1)
			s.Require().Equal(serviceInput, services[0])
			return nil
		})

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Namespace: testNamespace,
			Name:      "endpoint-for-service",
		},
	}

	res, err := s.reconciler.Reconcile(context.Background(), req)
	s.Require().NoError(err)
	s.Require().Equal(ctrl.Result{}, res)

	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&emptyList), gomock.Eq(client.InNamespace(testNamespace))).DoAndReturn(
		func(ctx context.Context, list *corev1.ServiceList, opts ...client.ListOption) error {
			list.Items = []corev1.Service{service}
			return nil
		})

	s.kubeFinder.EXPECT().ResolveOtterizeIdentityForService(gomock.Any(), &service, gomock.Any()).Return(serviceIdentity, true, nil)

	// Expect pod listing for TargetNamedPorts again
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&podList), gomock.Any()).DoAndReturn(
		func(ctx context.Context, list *corev1.PodList, opts ...client.ListOption) error {
			list.Items = []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: testNamespace,
						Labels:    map[string]string{"app": deploymentName},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "container-1",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 8080,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			}
			return nil
		})

	// Reconcile again should not upload cause re-upload due to caching
	res, err = s.reconciler.Reconcile(context.Background(), req)
	s.Require().NoError(err)
	s.Require().Equal(ctrl.Result{}, res)
}

func (s *ServiceVisibilityTestSuite) TestServiceReUploadOnIdentityChange() {
	deploymentName := "my-server"
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: testNamespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": deploymentName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(8080),
				},
			},
		},
	}

	emptyList := corev1.ServiceList{}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&emptyList), gomock.Eq(client.InNamespace(testNamespace))).DoAndReturn(
		func(ctx context.Context, list *corev1.ServiceList, opts ...client.ListOption) error {
			list.Items = []corev1.Service{service}
			return nil
		})

	serviceIdentity := model.OtterizeServiceIdentity{
		Name:              deploymentName,
		Namespace:         testNamespace,
		PodOwnerKind:      nil,
		KubernetesService: lo.ToPtr(service.Name),
	}
	s.kubeFinder.EXPECT().ResolveOtterizeIdentityForService(gomock.Any(), &service, gomock.Any()).Return(serviceIdentity, true, nil)

	// Expect pod listing for TargetNamedPorts
	podList := corev1.PodList{}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&podList), gomock.Any()).DoAndReturn(
		func(ctx context.Context, list *corev1.PodList, opts ...client.ListOption) error {
			list.Items = []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: testNamespace,
						Labels:    map[string]string{"app": deploymentName},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "container-1",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 8080,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			}
			return nil
		})

	serviceInput := cloudclient.K8sServiceInput{
		OtterizeServer: deploymentName,
		Namespace:      testNamespace,
		ResourceName:   service.Name,
		Service: cloudclient.K8sResourceServiceInput{
			Spec: cloudclient.K8sResourceServiceSpecInput{
				Type: nilable.From(cloudclient.K8sServiceTypeClusterIp),
				Ports: []cloudclient.K8sServicePort{
					{
						Port:       80,
						Protocol:   nilable.From(cloudclient.K8sPortProtocolTcp),
						TargetPort: nilable.From(cloudclient.IntOrStringInput{IntVal: nilable.From(8080), IsInt: true}),
					},
				},
				Selector: []cloudclient.SelectorKeyValueInput{{Key: nilable.From("app"), Value: nilable.From(deploymentName)}},
			},
			TargetNamedPorts: []cloudclient.NamedPortInput{
				{
					Name:     "http",
					Port:     8080,
					Protocol: cloudclient.K8sPortProtocol(corev1.ProtocolTCP),
				},
			},
		},
	}
	s.cloudClient.EXPECT().ReportK8sServices(gomock.Any(), testNamespace, gomock.Any()).DoAndReturn(
		func(ctx context.Context, namespace string, services []cloudclient.K8sServiceInput) error {
			s.Require().Len(services, 1)
			s.Require().Equal(serviceInput, services[0])
			return nil
		})

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Namespace: testNamespace,
			Name:      "endpoint-for-service",
		},
	}

	res, err := s.reconciler.Reconcile(context.Background(), req)
	s.Require().NoError(err)
	s.Require().Equal(ctrl.Result{}, res)

	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&emptyList), gomock.Eq(client.InNamespace(testNamespace))).DoAndReturn(
		func(ctx context.Context, list *corev1.ServiceList, opts ...client.ListOption) error {
			list.Items = []corev1.Service{service}
			return nil
		})

	newIdentity := model.OtterizeServiceIdentity{
		Name:              "another-server",
		Namespace:         testNamespace,
		PodOwnerKind:      nil,
		KubernetesService: lo.ToPtr(service.Name),
	}

	s.kubeFinder.EXPECT().ResolveOtterizeIdentityForService(gomock.Any(), &service, gomock.Any()).Return(newIdentity, true, nil)

	// Expect pod listing for TargetNamedPorts again
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&podList), gomock.Any()).DoAndReturn(
		func(ctx context.Context, list *corev1.PodList, opts ...client.ListOption) error {
			list.Items = []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: testNamespace,
						Labels:    map[string]string{"app": deploymentName},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "container-1",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 8080,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			}
			return nil
		})

	newServiceInput := cloudclient.K8sServiceInput{
		OtterizeServer: "another-server",
		Namespace:      testNamespace,
		ResourceName:   service.Name,
		Service: cloudclient.K8sResourceServiceInput{
			Spec: cloudclient.K8sResourceServiceSpecInput{
				Type: nilable.From(cloudclient.K8sServiceTypeClusterIp),
				Ports: []cloudclient.K8sServicePort{
					{
						Port:       80,
						Protocol:   nilable.From(cloudclient.K8sPortProtocolTcp),
						TargetPort: nilable.From(cloudclient.IntOrStringInput{IntVal: nilable.From(8080), IsInt: true}),
					},
				},
				Selector: []cloudclient.SelectorKeyValueInput{{Key: nilable.From("app"), Value: nilable.From(deploymentName)}},
			},
			TargetNamedPorts: []cloudclient.NamedPortInput{
				{
					Name:     "http",
					Port:     8080,
					Protocol: cloudclient.K8sPortProtocol(corev1.ProtocolTCP),
				},
			},
		},
	}

	s.cloudClient.EXPECT().ReportK8sServices(gomock.Any(), testNamespace, gomock.Any()).DoAndReturn(
		func(ctx context.Context, namespace string, services []cloudclient.K8sServiceInput) error {
			s.Require().Len(services, 1)
			s.Require().Equal(newServiceInput, services[0])
			return nil
		})

	res, err = s.reconciler.Reconcile(context.Background(), req)
	s.Require().NoError(err)
	s.Require().Equal(ctrl.Result{}, res)

	nodePortService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: testNamespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"app": deploymentName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(8080),
				},
			},
		},
	}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&emptyList), gomock.Eq(client.InNamespace(testNamespace))).DoAndReturn(
		func(ctx context.Context, list *corev1.ServiceList, opts ...client.ListOption) error {
			list.Items = []corev1.Service{nodePortService}
			return nil
		})

	s.kubeFinder.EXPECT().ResolveOtterizeIdentityForService(gomock.Any(), &nodePortService, gomock.Any()).Return(newIdentity, true, nil)

	// Expect pod listing for TargetNamedPorts for nodePortService
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&podList), gomock.Any()).DoAndReturn(
		func(ctx context.Context, list *corev1.PodList, opts ...client.ListOption) error {
			list.Items = []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: testNamespace,
						Labels:    map[string]string{"app": deploymentName},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "container-1",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 8080,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			}
			return nil
		})

	nodePortServiceInput := cloudclient.K8sServiceInput{
		OtterizeServer: "another-server",
		Namespace:      testNamespace,
		ResourceName:   service.Name,
		Service: cloudclient.K8sResourceServiceInput{
			Spec: cloudclient.K8sResourceServiceSpecInput{
				Type: nilable.From(cloudclient.K8sServiceTypeNodePort),
				Ports: []cloudclient.K8sServicePort{
					{
						Port:       80,
						Protocol:   nilable.From(cloudclient.K8sPortProtocolTcp),
						TargetPort: nilable.From(cloudclient.IntOrStringInput{IntVal: nilable.From(8080), IsInt: true}),
					},
				},
				Selector: []cloudclient.SelectorKeyValueInput{{Key: nilable.From("app"), Value: nilable.From(deploymentName)}},
			},
			TargetNamedPorts: []cloudclient.NamedPortInput{
				{
					Name:     "http",
					Port:     8080,
					Protocol: cloudclient.K8sPortProtocol(corev1.ProtocolTCP),
				},
			},
		},
	}

	s.cloudClient.EXPECT().ReportK8sServices(gomock.Any(), testNamespace, gomock.Any()).DoAndReturn(
		func(ctx context.Context, namespace string, services []cloudclient.K8sServiceInput) error {
			s.Require().Len(services, 1)
			s.Require().Equal(nodePortServiceInput, services[0])
			return nil

		})

	res, err = s.reconciler.Reconcile(context.Background(), req)
	s.Require().NoError(err)
	s.Require().Equal(ctrl.Result{}, res)
}

func (s *ServiceVisibilityTestSuite) TestUploadEmptyNamespaces() {
	emptyList := corev1.ServiceList{}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&emptyList), gomock.Eq(client.InNamespace(testNamespace))).DoAndReturn(
		func(ctx context.Context, list *corev1.ServiceList, opts ...client.ListOption) error {
			return nil
		})

	s.cloudClient.EXPECT().ReportK8sServices(gomock.Any(), testNamespace, gomock.Eq([]cloudclient.K8sServiceInput{})).Return(nil)

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Namespace: testNamespace,
			Name:      "endpoint-for-service",
		},
	}

	res, err := s.reconciler.Reconcile(context.Background(), req)
	s.Require().NoError(err)
	s.Require().Equal(ctrl.Result{}, res)
}

func (s *ServiceVisibilityTestSuite) TestUploadEmptyNamespacesWhenNoPods() {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: testNamespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "my-server",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(8080),
				},
			},
		},
	}

	emptyList := corev1.ServiceList{}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&emptyList), gomock.Eq(client.InNamespace(testNamespace))).DoAndReturn(
		func(ctx context.Context, list *corev1.ServiceList, opts ...client.ListOption) error {
			list.Items = []corev1.Service{service}
			return nil
		})

	s.kubeFinder.EXPECT().ResolveOtterizeIdentityForService(gomock.Any(), &service, gomock.Any()).Return(model.OtterizeServiceIdentity{}, false, nil)

	s.cloudClient.EXPECT().ReportK8sServices(gomock.Any(), testNamespace, gomock.Eq([]cloudclient.K8sServiceInput{})).Return(nil)

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Namespace: testNamespace,
			Name:      "endpoint-for-service",
		},
	}

	res, err := s.reconciler.Reconcile(context.Background(), req)
	s.Require().NoError(err)
	s.Require().Equal(ctrl.Result{}, res)
}

func (s *ServiceVisibilityTestSuite) TestUploadEmptyNamespacesWhenServiceDeleted() {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-service",
			Namespace:         testNamespace,
			DeletionTimestamp: &metav1.Time{Time: time.Date(2020, 12, 1, 17, 14, 0, 0, time.UTC)},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "my-server",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(8080),
				},
			},
		},
	}

	emptyList := corev1.ServiceList{}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&emptyList), gomock.Eq(client.InNamespace(testNamespace))).DoAndReturn(
		func(ctx context.Context, list *corev1.ServiceList, opts ...client.ListOption) error {
			list.Items = []corev1.Service{service}
			return nil
		})

	s.cloudClient.EXPECT().ReportK8sServices(gomock.Any(), testNamespace, gomock.Eq([]cloudclient.K8sServiceInput{})).Return(nil)

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Namespace: testNamespace,
			Name:      "endpoint-for-service",
		},
	}

	res, err := s.reconciler.Reconcile(context.Background(), req)
	s.Require().NoError(err)
	s.Require().Equal(ctrl.Result{}, res)
}

func TestServiceVisibilityTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceVisibilityTestSuite))
}
