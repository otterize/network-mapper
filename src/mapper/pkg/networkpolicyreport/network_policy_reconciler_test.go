package networkpolicyreport

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"testing"
)

type NetworkPolicyReconcilerTestSuite struct {
	suite.Suite
	cloudClient *cloudclientmocks.MockCloudClient
	k8sClient   *mocks.K8sClient
	reconciler  *NetworkPolicyReconciler
}

func (s *NetworkPolicyReconcilerTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.cloudClient = cloudclientmocks.NewMockCloudClient(controller)
	s.k8sClient = mocks.NewK8sClient(controller)
	s.reconciler = NewNetworkPolicyReconciler(s.k8sClient, s.cloudClient)
}

func (s *NetworkPolicyReconcilerTestSuite) TestNetworkPolicyUpload() {
	resourceName := "test-networkpolicy"
	testNamespace := "test-namespace"
	expectedNetworkPolicy := networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: testNamespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-app",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": "test-app",
								},
							},
						},
					},
				},
			},
		},
	}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&networkingv1.NetworkPolicyList{}), gomock.Eq(client.InNamespace(testNamespace))).DoAndReturn(
		func(ctx context.Context, list *networkingv1.NetworkPolicyList, opts ...client.ListOption) error {
			list.Items = append(list.Items, expectedNetworkPolicy)
			return nil
		})

	expectedContent, err := yaml.Marshal(expectedNetworkPolicy)
	s.Require().NoError(err)
	cloudInput := cloudclient.NetworkPolicyInput{
		Name: resourceName,
		Yaml: string(expectedContent),
	}

	s.cloudClient.EXPECT().ReportNetworkPolicies(gomock.Any(), testNamespace, gomock.Eq([]cloudclient.NetworkPolicyInput{cloudInput})).Return(nil)

	res, err := s.reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: resourceName, Namespace: testNamespace}})
	s.NoError(err)
	s.True(res.IsZero())
}

func TestNetworkPolicyReconcilerTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkPolicyReconcilerTestSuite))
}
