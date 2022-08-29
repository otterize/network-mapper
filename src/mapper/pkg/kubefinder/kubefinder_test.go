package kubefinder

import (
	"context"
	"fmt"
	"github.com/otterize/otternose/mapper/pkg/graph/model"
	"github.com/otterize/otternose/shared/testbase"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

type KubeFinderTestSuite struct {
	testbase.ControllerManagerTestSuiteBase
}

func (suite *KubeFinderTestSuite) TestResolveIpToPod() {
	kf, err := NewKubeFinder(suite.Mgr)
	suite.Require().NoError(err)

	pod, err := kf.ResolveIpToPod(context.Background(), "1.1.1.1")
	suite.Require().Nil(pod)
	suite.Require().Error(err)

	suite.AddPod("some-pod", "2.2.2.2", nil)
	suite.AddPod("test-pod", "1.1.1.1", nil)
	suite.AddPod("pod-with-no-ip", "", nil)
	suite.Mgr.GetCache().WaitForCacheSync(context.Background())

	pod, err = kf.ResolveIpToPod(context.Background(), "1.1.1.1")
	suite.Require().NoError(err)
	suite.Require().Equal("test-pod", pod.Name)

}

func (suite *KubeFinderTestSuite) TestResolveServiceAddressToIps() {
	kf, err := NewKubeFinder(suite.Mgr)
	suite.Require().NoError(err)

	_, err = kf.ResolveServiceAddressToIps(context.Background(), "www.google.com")
	suite.Require().Error(err)

	_, err = kf.ResolveServiceAddressToIps(context.Background(), fmt.Sprintf("service1.%s.svc.cluster.local", suite.TestNamespace))
	suite.Require().Error(err)

	podIp0 := "1.1.1.1"
	podIp1 := "1.1.1.2"
	podIp2 := "1.1.1.3"
	suite.AddDeploymentWithService("service0", []string{podIp0}, map[string]string{"app": "service0"})
	suite.AddDeploymentWithService("service1", []string{podIp1, podIp2}, map[string]string{"app": "service1"})
	suite.Mgr.GetCache().WaitForCacheSync(context.Background())

	ips, err := kf.ResolveServiceAddressToIps(context.Background(), fmt.Sprintf("service1.%s.svc.cluster.local", suite.TestNamespace))
	suite.Require().NoError(err)
	suite.Require().ElementsMatch(ips, []string{podIp1, podIp2})

	// make sure we don't fail on the longer forms of k8s service addresses, listed on this page: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service
	ips, err = kf.ResolveServiceAddressToIps(context.Background(), fmt.Sprintf("4-4-4-4.service1.%s.svc.cluster.local", suite.TestNamespace))
	suite.Require().NoError(err)
	suite.Require().ElementsMatch(ips, []string{podIp1, podIp2})

	ips, err = kf.ResolveServiceAddressToIps(context.Background(), fmt.Sprintf("4-4-4-4.%s.pod.cluster.local", suite.TestNamespace))
	suite.Require().NoError(err)
	suite.Require().ElementsMatch(ips, []string{"4.4.4.4"})
}

func (suite *KubeFinderTestSuite) TestResolvePodToOtterizeServiceIdentity() {
	kf, err := NewKubeFinder(suite.Mgr)
	suite.Require().NoError(err)

	suite.AddDeploymentWithService("service0", []string{"1.1.1.1", "1.1.1.2"}, map[string]string{"app": "test"})
	suite.Mgr.GetCache().WaitForCacheSync(context.Background())

	podList := &corev1.PodList{}
	err = suite.Mgr.GetClient().List(context.Background(), podList, client.HasLabels{"app=test"}, client.InNamespace(suite.TestNamespace))
	suite.Require().NoError(err)
	suite.Require().NotEmpty(podList.Items)

	for _, pod := range podList.Items {
		identity, err := kf.ResolvePodToOtterizeServiceIdentity(context.Background(), &pod)
		suite.Require().NoError(err)
		suite.Require().Equal(model.OtterizeServiceIdentity{Name: "service0", Namespace: suite.TestNamespace}, identity)
	}
}

func TestKubeFinderTestSuite(t *testing.T) {
	suite.Run(t, new(KubeFinderTestSuite))
}
