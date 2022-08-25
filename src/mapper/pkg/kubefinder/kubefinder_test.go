package kubefinder

import (
	"context"
	"fmt"
	"github.com/otterize/otternose/shared/testutils"
	"github.com/stretchr/testify/suite"
	"testing"
)

type KubeFinderTestSuite struct {
	testutils.ManagerTestSuite
}

func (suite *KubeFinderTestSuite) TestResolveIpToPod() {
	kf, err := NewKubeFinder(suite.Mgr)
	suite.Require().NoError(err)

	pod, err := kf.ResolveIpToPod(context.Background(), "1.1.1.1")
	suite.Require().Nil(pod)
	suite.Require().Error(err)

	suite.AddPod("some-pod", "2.2.2.2")
	suite.AddPod("test-pod", "1.1.1.1")
	suite.AddPod("pod-with-no-ip", "")

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

	podIp1 := "1.1.1.1"
	podIp2 := "1.1.1.2"
	podIp3 := "1.1.1.3"
	suite.AddPod("pod1", podIp1)
	suite.AddPod("pod2", podIp1)
	suite.AddPod("pod3", podIp1)
	suite.AddEndpoints("service0", []string{podIp2, podIp3})
	suite.AddEndpoints("service1", []string{podIp1, podIp2})
	suite.AddService("service0")
	suite.AddService("service1")

	ips, err := kf.ResolveServiceAddressToIps(context.Background(), fmt.Sprintf("service1.%s.svc.cluster.local", suite.TestNamespace))
	suite.Require().NoError(err)
	suite.Require().ElementsMatch(ips, []string{podIp1, podIp2})
	suite.Require().NotContains(ips, podIp3)

	// make sure we don't fail on the longer forms of k8s service addresses, like this: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-hostname-and-subdomain-fields
	ips, err = kf.ResolveServiceAddressToIps(context.Background(), fmt.Sprintf("longer.address.%s.svc.cluster.local", suite.TestNamespace))
	suite.Require().NoError(err)
	suite.Require().ElementsMatch(ips, []string{podIp1, podIp2})
	suite.Require().NotContains(ips, podIp3)

	ips, err = kf.ResolveServiceAddressToIps(context.Background(), fmt.Sprintf("4-4-4-4.%s.pod.cluster.local", suite.TestNamespace))
	suite.Require().NoError(err)
	suite.Require().ElementsMatch(ips, []string{"4.4.4.4"})
}

func TestKubeFinderTestSuite(t *testing.T) {
	suite.Run(t, new(KubeFinderTestSuite))
}
