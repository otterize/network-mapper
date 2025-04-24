package metadatareporter

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/mocks"
	"github.com/otterize/nilable"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

type NamespaceReconcilerTestSuite struct {
	suite.Suite
	cloudClient *cloudclientmocks.MockCloudClient
	k8sClient   *mocks.K8sClient
	reconciler  *NamespaceReconciler
}

func (s *NamespaceReconcilerTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.cloudClient = cloudclientmocks.NewMockCloudClient(controller)
	s.k8sClient = mocks.NewK8sClient(controller)
	s.reconciler = NewNamespaceReconciler(s.k8sClient, s.cloudClient)
}

func (s *NamespaceReconcilerTestSuite) TestNamespaceReconciler_Reconcile() {
	testNamespaceName := "test-namespace"
	testNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespaceName,
			Labels: map[string]string{"key1": "value1", "key2": "value2"},
		},
	}

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{Name: testNamespaceName},
	}

	s.k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, gomock.Any()).DoAndReturn(
		func(ctx context.Context, name types.NamespacedName, namespace *corev1.Namespace, _ ...any) error {
			testNS.DeepCopyInto(namespace)
			return nil
		})

	s.cloudClient.EXPECT().ReportNamespaceLabels(gomock.Any(), testNamespaceName, []cloudclient.LabelInput{
		{Key: "key1", Value: nilable.From("value1")},
		{Key: "key2", Value: nilable.From("value2")},
	}).Return(nil)

	res, err := s.reconciler.Reconcile(context.Background(), req)
	s.NoError(err)
	s.Require().True(res.IsZero())
}

func TestNamespaceReconcilerTestSuite(t *testing.T) {
	suite.Run(t, new(NamespaceReconcilerTestSuite))
}
