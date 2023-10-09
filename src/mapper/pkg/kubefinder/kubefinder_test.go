package kubefinder

import (
	"context"
	"fmt"
	"github.com/otterize/network-mapper/src/shared/testbase"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

type KubeFinderTestSuite struct {
	testbase.ControllerManagerTestSuiteBase
	kubeFinder *KubeFinder
}

func (s *KubeFinderTestSuite) SetupTest() {
	s.ControllerManagerTestSuiteBase.SetupTest()
	var err error
	s.kubeFinder, err = NewKubeFinder(context.Background(), s.Mgr)
	s.Require().NoError(err)
}

func (s *KubeFinderTestSuite) TestResolveIpToPod() {
	pod, err := s.kubeFinder.ResolveIPToPod(context.Background(), "1.1.1.1")
	s.Require().Nil(pod)
	s.Require().Error(err)

	s.AddPod("some-pod", "2.2.2.2", nil, nil)
	s.AddPod("test-pod", "1.1.1.1", nil, nil)
	s.AddPod("pod-with-no-ip", "", nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	pod, err = s.kubeFinder.ResolveIPToPod(context.Background(), "1.1.1.1")
	s.Require().NoError(err)
	s.Require().Equal("test-pod", pod.Name)

}

func (s *KubeFinderTestSuite) TestResolveServiceAddressToIps() {
	_, _, err := s.kubeFinder.ResolveServiceAddressToIps(context.Background(), "www.google.com")
	s.Require().Error(err)

	_, _, err = s.kubeFinder.ResolveServiceAddressToIps(context.Background(), fmt.Sprintf("svc-service1.%s.svc.cluster.local", s.TestNamespace))
	s.Require().Error(err)

	podIp0 := "1.1.1.1"
	podIp1 := "1.1.1.2"
	podIp2 := "1.1.1.3"
	s.Require().NoError(s.Mgr.GetClient().List(context.Background(), &corev1.EndpointsList{})) // Workaround: make then client start caching Endpoints, so when we do "WaitForCacheSync" it will actually sync cache"
	s.AddDeploymentWithService("service0", []string{podIp0}, map[string]string{"app": "service0"}, "10.0.0.10")
	s.AddDeploymentWithService("service1", []string{podIp1, podIp2}, map[string]string{"app": "service1"}, "10.0.0.11")
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	ips, serviceName, err := s.kubeFinder.ResolveServiceAddressToIps(context.Background(), fmt.Sprintf("svc-service1.%s.svc.cluster.local", s.TestNamespace))
	s.Require().NoError(err)
	s.Require().Equal("svc-service1", serviceName)
	s.Require().ElementsMatch(ips, []string{podIp1, podIp2})

	// make sure we don't fail on the longer forms of k8s service addresses, listed on this page: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service
	ips, serviceName, err = s.kubeFinder.ResolveServiceAddressToIps(context.Background(), fmt.Sprintf("4-4-4-4.svc-service1.%s.svc.cluster.local", s.TestNamespace))
	s.Require().Equal("svc-service1", serviceName)
	s.Require().NoError(err)
	s.Require().ElementsMatch(ips, []string{podIp1, podIp2})

	ips, serviceName, err = s.kubeFinder.ResolveServiceAddressToIps(context.Background(), fmt.Sprintf("4-4-4-4.%s.pod.cluster.local", s.TestNamespace))
	s.Require().NoError(err)
	s.Require().Empty(serviceName)
	s.Require().ElementsMatch(ips, []string{"4.4.4.4"})
}

func TestKubeFinderTestSuite(t *testing.T) {
	suite.Run(t, new(KubeFinderTestSuite))
}
