package resourcevisibility

import (
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"

	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

const (
	testNamespace = "test-namespace-a"
)

type ResourceVisibilityTestSuite struct {
	suite.Suite
	cloudClient *cloudclientmocks.MockCloudClient
	k8sClient   *mocks.K8sClient
	reconciler  *ServiceReconciler
}

func (s *ResourceVisibilityTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.cloudClient = cloudclientmocks.NewMockCloudClient(controller)
	s.k8sClient = mocks.NewK8sClient(controller)
	s.reconciler = NewServiceReconciler(s.k8sClient, s.cloudClient, nil)
}

func (s *ResourceVisibilityTestSuite) TearDownTest() {
	s.cloudClient = nil
	s.k8sClient = nil
}

func (s *ResourceVisibilityTestSuite) TestServiceUpload() {
	s.T().Skip()
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
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&emptyList), gomock.Any()).DoAndReturn(
		func(ctx context.Context, list *corev1.ServiceList, opts ...client.ListOption) error {
			list.Items = []corev1.Service{service}
			return nil
		})

	s.cloudClient.EXPECT().ReportK8sServices(gomock.Any(), testNamespace, gomock.Any()).Return(nil)
}

func (s *ResourceVisibilityTestSuite) TestUploadEmptyNamespaces() {

}

func (s *ResourceVisibilityTestSuite) TestUploadEmptyNamespacesWhenNoPods() {

}

func (s *ResourceVisibilityTestSuite) TestUploadEmptyNamespacesWhenServiceDeleted() {

}

func TestResourceVisibilityTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceVisibilityTestSuite))
}
