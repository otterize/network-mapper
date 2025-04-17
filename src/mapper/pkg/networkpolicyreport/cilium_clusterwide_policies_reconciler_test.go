package networkpolicyreport

import (
	"context"
	v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
)

type CiliumClusterwidePolicyReconcilerTestSuite struct {
	suite.Suite
	cloudClient *cloudclientmocks.MockCloudClient
	k8sClient   *mocks.K8sClient
	reconciler  *CiliumClusterWideNetworkPolicyReconciler
}

func (s *CiliumClusterwidePolicyReconcilerTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.cloudClient = cloudclientmocks.NewMockCloudClient(controller)
	s.k8sClient = mocks.NewK8sClient(controller)
	s.reconciler = NewCiliumClusterWideNetworkPolicyReconciler(s.k8sClient, s.cloudClient)
}

func (s *CiliumClusterwidePolicyReconcilerTestSuite) TestCiliumClusterWidePolicyUpload() {
	resourceName := "test-cilium-policy"
	policy := v2.CiliumClusterwideNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
			Annotations: map[string]string{
				"keyLarge": strings.Repeat("a", 1000),
				"keySmall": "value",
			},
		},
		Spec: &api.Rule{
			Ingress: make([]api.IngressRule, 0),
		},
	}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Eq(&v2.CiliumClusterwideNetworkPolicyList{})).DoAndReturn(
		func(ctx context.Context, list *v2.CiliumClusterwideNetworkPolicyList, opts ...client.ListOption) error {
			list.Items = append(list.Items, policy)
			return nil
		})

	expectedPolicy := policy.DeepCopy()
	// filter large annotation
	delete(expectedPolicy.Annotations, "keyLarge")
	expectedContent, err := yaml.Marshal(expectedPolicy)
	s.Require().NoError(err)
	cloudInput := cloudclient.NetworkPolicyInput{
		Name: resourceName,
		Yaml: string(expectedContent),
	}

	s.cloudClient.EXPECT().ReportCiliumClusterWideNetworkPolicies(gomock.Any(), gomock.Eq([]cloudclient.NetworkPolicyInput{cloudInput})).Return(nil)

	res, err := s.reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: resourceName}})
	s.NoError(err)
	s.True(res.IsZero())
}

func TestCiliumClusterwidePolicyReconcilerTestSuite(t *testing.T) {
	suite.Run(t, new(CiliumClusterwidePolicyReconcilerTestSuite))
}
