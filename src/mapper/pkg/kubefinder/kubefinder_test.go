package kubefinder

import (
	"context"
	"fmt"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/shared/testbase"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	"net"
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

func (s *KubeFinderTestSuite) TestResolveIpToControlPlane() {
	endpoints := s.GetAPIServerEndpoints()
	endpointIP := endpoints.Subsets[0].Addresses[0].IP
	pod, found, err := s.kubeFinder.ResolveIPToControlPlane(context.Background(), endpointIP)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal("kubernetes", pod.Name)
	s.Require().Equal("default", pod.Namespace)
}

func (s *KubeFinderTestSuite) TestResolveIpToControlPlaneSubnet() {
	endpoints := s.GetAPIServerEndpoints()
	endpointIP := endpoints.Subsets[0].Addresses[0].IP
	viper.Set(config.ControlPlaneIPv4CidrPrefixLength, "28")
	defer func() {
		viper.Set(config.ControlPlaneIPv4CidrPrefixLength, config.ControlPlaneIPv4CidrPrefixLengthDefault)
	}()

	_, subnet, err := net.ParseCIDR(fmt.Sprintf("%s/28", endpointIP))
	s.Require().NoError(err)

	// iterate over IPs in the same subnet (only increment the last byte for simplicity)
	for ip := subnet.IP.Mask(subnet.Mask).To4(); subnet.Contains(ip); ip[3]++ {
		pod, found, err := s.kubeFinder.ResolveIPToControlPlane(context.Background(), ip.String())
		s.Require().NoError(err)
		s.Require().True(found)
		s.Require().Equal("kubernetes", pod.Name)
		s.Require().Equal("default", pod.Namespace)
	}
}

func (s *KubeFinderTestSuite) TestResolveServiceAddressToIps() {
	_, _, err := s.kubeFinder.ResolveServiceAddressToPods(context.Background(), "www.google.com")
	s.Require().Error(err)

	_, _, err = s.kubeFinder.ResolveServiceAddressToPods(context.Background(), fmt.Sprintf("svc-service1.%s.svc.cluster.local", s.TestNamespace))
	s.Require().Error(err)

	podIp0 := "1.1.1.1"
	podIp1 := "1.1.1.2"
	podIp2 := "1.1.1.3"
	s.Require().NoError(s.Mgr.GetClient().List(context.Background(), &corev1.EndpointsList{})) // Workaround: make then client start caching Endpoints, so when we do "WaitForCacheSync" it will actually sync cache"
	s.AddDeploymentWithService("service0", []string{podIp0}, map[string]string{"app": "service0"}, "10.0.0.10")
	_, _, retPods := s.AddDeploymentWithService("service1", []string{podIp1, podIp2}, map[string]string{"app": "service1"}, "10.0.0.11")
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	pods, service, err := s.kubeFinder.ResolveServiceAddressToPods(context.Background(), fmt.Sprintf("svc-service1.%s.svc.cluster.local", s.TestNamespace))
	s.Require().NoError(err)
	s.Require().Equal("svc-service1", service.Name)
	s.Require().ElementsMatch(lo.Map(pods, func(p corev1.Pod, _ int) string { return p.Status.PodIP }), lo.Map(retPods, func(p *corev1.Pod, _ int) string { return p.Status.PodIP }))

	// make sure we don't fail on the longer forms of k8s service addresses, listed on this page: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service
	pods, service, err = s.kubeFinder.ResolveServiceAddressToPods(context.Background(), fmt.Sprintf("4-4-4-4.svc-service1.%s.svc.cluster.local", s.TestNamespace))
	s.Require().Equal("svc-service1", service.Name)
	s.Require().NoError(err)
	s.Require().ElementsMatch(lo.Map(pods, func(p corev1.Pod, _ int) string { return p.Status.PodIP }), lo.Map(retPods, func(p *corev1.Pod, _ int) string { return p.Status.PodIP }))

	_, _, err = s.kubeFinder.ResolveServiceAddressToPods(context.Background(), fmt.Sprintf("4-4-4-4.%s.pod.cluster.local", s.TestNamespace))
	s.Require().Error(err)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	_, pods4444 := s.AddDeployment("depl", []string{"4.4.4.4"}, map[string]string{"app": "4444"})
	pods, service, err = s.kubeFinder.ResolveServiceAddressToPods(context.Background(), fmt.Sprintf("4-4-4-4.%s.pod.cluster.local", s.TestNamespace))
	s.Require().NoError(err)
	s.Require().Empty(service)
	s.Require().ElementsMatch(lo.Map(pods, func(p corev1.Pod, _ int) string { return p.Status.PodIP }), lo.Map(pods4444, func(p *corev1.Pod, _ int) string { return p.Status.PodIP }))
}

func (s *KubeFinderTestSuite) TestIsSrcIpClusterInternal() {
	pod := s.AddPod("test-pod", "1.1.1.1", nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	// Check with existing pod's ip
	isInternal, err := s.kubeFinder.IsSrcIpClusterInternal(context.Background(), "1.1.1.1")
	s.Require().NoError(err)
	s.Require().True(isInternal)

	// Check with non-existing pod's ip
	isInternal, err = s.kubeFinder.IsSrcIpClusterInternal(context.Background(), "8.8.8.8")
	s.Require().NoError(err)
	s.Require().False(isInternal)

	err = s.Mgr.GetClient().Delete(context.Background(), pod)
	s.Require().NoError(err)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	// Check pod doesn't exist in the manager's cache
	s.WaitForObjectToBeDeleted(pod)

	// Check isInternal with the deleted pod's ip
	isInternal, err = s.kubeFinder.IsSrcIpClusterInternal(context.Background(), "1.1.1.1")
	s.Require().NoError(err)
	s.Require().True(isInternal)

	// Reset the cache
	s.kubeFinder.initSeenIPsCache()

	// Check isInternal with the deleted pod's ip after cache reset
	isInternal, err = s.kubeFinder.IsSrcIpClusterInternal(context.Background(), "1.1.1.1")
	s.Require().NoError(err)
	s.Require().False(isInternal)
}

func TestKubeFinderTestSuite(t *testing.T) {
	suite.Run(t, new(KubeFinderTestSuite))
}
