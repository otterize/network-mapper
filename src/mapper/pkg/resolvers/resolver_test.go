package resolvers

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/externaltrafficholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/mapper/pkg/resolvers/test_gql_client"
	"github.com/otterize/network-mapper/src/shared/testbase"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http/httptest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"testing"
	"time"
)

type ResolverTestSuite struct {
	testbase.ControllerManagerTestSuiteBase
	server                       *httptest.Server
	client                       graphql.Client
	kubeFinder                   *kubefinder.KubeFinder
	intentsHolder                *intentsstore.IntentsHolder
	externalTrafficIntentsHolder *externaltrafficholder.ExternalTrafficIntentsHolder
}

func (s *ResolverTestSuite) SetupTest() {
	s.ControllerManagerTestSuiteBase.SetupTest()
	e := echo.New()
	var err error
	s.kubeFinder, err = kubefinder.NewKubeFinder(context.Background(), s.Mgr)
	s.Require().NoError(err)
	s.intentsHolder = intentsstore.NewIntentsHolder()
	s.externalTrafficIntentsHolder = externaltrafficholder.NewExternalTrafficIntentsHolder()
	resolver := NewResolver(s.kubeFinder, serviceidresolver.NewResolver(s.Mgr.GetClient()), s.intentsHolder, s.externalTrafficIntentsHolder)
	resolver.Register(e)
	s.server = httptest.NewServer(e)
	s.client = graphql.NewClient(s.server.URL+"/query", s.server.Client())
}

func (s *ResolverTestSuite) TestReportCaptureResults() {
	s.AddDeploymentWithService("service1", []string{"1.1.1.1"}, map[string]string{"app": "service1"}, "10.0.0.16")
	s.AddDeploymentWithService("service2", []string{"1.1.1.2"}, map[string]string{"app": "service2"}, "10.0.0.17")
	s.AddDaemonSetWithService("service3", []string{"1.1.1.3"}, map[string]string{"app": "service3"}, "10.0.0.18")
	s.AddPod("pod4", "1.1.1.4", nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	packetTime := time.Now().Add(time.Minute)
	_, err := test_gql_client.ReportCaptureResults(context.Background(), s.client, test_gql_client.CaptureResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "1.1.1.1",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.1.3",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service1.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.1.4",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", "service1"),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", "service3"),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service1"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service1",
				},
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "pod4",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "",
					Kind:    "Pod",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
	})
}

func (s *ResolverTestSuite) TestReportCaptureResultsHostnameMismatch() {
	s.AddDeploymentWithService("service1", []string{"1.1.1.1"}, map[string]string{"app": "service1"}, "10.0.0.16")
	s.AddDeploymentWithService("service2", []string{"1.1.1.2"}, map[string]string{"app": "service2"}, "10.0.0.17")
	s.AddDaemonSetWithService("service3", []string{"1.1.1.3"}, map[string]string{"app": "service3"}, "10.0.0.18")
	s.AddPod("pod4", "1.1.1.4", nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	packetTime := time.Now().Add(time.Minute)
	_, err := test_gql_client.ReportCaptureResults(context.Background(), s.client, test_gql_client.CaptureResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "1.1.1.1",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.1.3",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service1.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			// should be discarded - hostname mismatch
			{
				SrcIp:       "1.1.1.4",
				SrcHostname: "pod5",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", "service1"),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", "service3"),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service1"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service1",
				},
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
	})
}

func (s *ResolverTestSuite) TestReportCaptureResultsPodDeletion() {
	s.AddDeploymentWithService("service1", []string{"1.1.1.1"}, map[string]string{"app": "service1"}, "10.0.0.16")
	s.AddDeploymentWithService("service2", []string{"1.1.1.2"}, map[string]string{"app": "service2"}, "10.0.0.17")
	s.AddDaemonSetWithService("service3", []string{"1.1.1.3"}, map[string]string{"app": "service3"}, "10.0.0.18")
	pod := s.AddPod("pod4", "1.1.1.4", nil, nil)
	var podToUpdate v1.Pod
	err := s.Mgr.GetClient().Get(context.Background(), types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}, &podToUpdate)
	s.Require().NoError(err)
	s.Require().True(controllerutil.AddFinalizer(&podToUpdate, "intents.otterize.com/finalizer-so-that-object-cant-be-deleted-for-this-test"))
	err = s.Mgr.GetClient().Update(context.Background(), &podToUpdate)
	s.Require().NoError(err)
	s.Require().NoError(wait.PollImmediate(1*time.Second, 10*time.Second, func() (done bool, err error) {
		var readPod v1.Pod
		err = s.Mgr.GetClient().Get(context.Background(), types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}, &readPod)
		if errors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		if !slices.Contains(readPod.Finalizers, "intents.otterize.com/finalizer-so-that-object-cant-be-deleted-for-this-test") {
			return false, nil
		}
		return true, nil
	}))

	err = s.Mgr.GetClient().Delete(context.Background(), pod)
	s.Require().NoError(err)

	packetTime := time.Now().Add(time.Minute)
	_, err = test_gql_client.ReportCaptureResults(context.Background(), s.client, test_gql_client.CaptureResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "1.1.1.1",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.1.3",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service1.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			// should be discarded - deleted pod
			{
				SrcIp: "1.1.1.4",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", "service1"),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", "service3"),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service1"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service1",
				},
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
	})
}

func (s *ResolverTestSuite) TestReportCaptureResultsIPReuse() {
	s.AddDeploymentWithService("service1", []string{"1.1.1.1"}, map[string]string{"app": "service1"}, "10.0.0.16")
	s.AddDeploymentWithService("service2", []string{"1.1.1.2"}, map[string]string{"app": "service2"}, "10.0.0.17")
	s.AddDaemonSetWithService("service3", []string{"1.1.1.3"}, map[string]string{"app": "service3"}, "10.0.0.18")
	s.AddPod("pod4", "1.1.1.4", nil, nil)
	// intentionally reusing Pod IP
	s.AddDaemonSetWithService("network-sniffer", []string{"1.1.1.5"}, map[string]string{"app": "network-sniffer"}, "10.0.0.19")
	s.AddDaemonSetWithService("network-sniffer-2", []string{"1.1.1.5"}, map[string]string{"app": "network-sniffer"}, "10.0.0.20")
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	packetTime := time.Now().Add(time.Minute)
	_, err := test_gql_client.ReportCaptureResults(context.Background(), s.client, test_gql_client.CaptureResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "1.1.1.1",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.1.3",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service1.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.1.4",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			// should be discarded - IP belongs to more than one pod
			{
				SrcIp: "1.1.1.5",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", "service1"),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", "service3"),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service1"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service1",
				},
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "pod4",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "",
					Kind:    "Pod",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
	})
}

func (s *ResolverTestSuite) TestReportCaptureResultsIgnoreOldPacket() {
	s.AddDeploymentWithService("service1", []string{"1.1.1.1"}, map[string]string{"app": "service1"}, "10.0.0.19")
	s.AddDeploymentWithService("service2", []string{"1.1.1.2"}, map[string]string{"app": "service2"}, "10.0.0.20")
	s.AddDaemonSetWithService("service3", []string{"1.1.1.3"}, map[string]string{"app": "service3"}, "10.0.0.21")
	s.AddPod("pod4", "1.1.1.4", nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	packetTime := time.Now().Add(-1 * time.Minute)
	_, err := test_gql_client.ReportCaptureResults(context.Background(), s.client, test_gql_client.CaptureResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "1.1.1.1",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{})
}

func (s *ResolverTestSuite) TestSocketScanResults() {
	const (
		service1podIP = "1.1.2.1"
		service2podIP = "1.1.2.2"
		service3podIP = "1.1.2.3"
		service4podIP = "1.1.2.4"
		service1IP    = "10.0.0.22"
		service2IP    = "10.0.0.23"
		service3IP    = "10.0.0.24"
	)
	s.AddDaemonSetWithService("service1", []string{service1podIP}, map[string]string{"app": "service1"}, service1IP)
	s.AddDeploymentWithService("service2", []string{service2podIP}, map[string]string{"app": "service2"}, service2IP)
	s.AddDeploymentWithService("service3", []string{service3podIP}, map[string]string{"app": "service3"}, service3IP)
	s.AddPod("pod4", service4podIP, nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	packetTime := time.Now().Add(time.Minute)

	_, err := test_gql_client.ReportSocketScanResults(context.Background(), s.client, test_gql_client.SocketScanResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: service1podIP,
				Destinations: []test_gql_client.Destination{
					{
						Destination: service2IP,
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: service3podIP,
				Destinations: []test_gql_client.Destination{
					{
						Destination: service1IP,
						LastSeen:    packetTime,
					},
					{
						Destination: service2IP,
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: service4podIP,
				Destinations: []test_gql_client.Destination{
					{
						Destination: service2IP,
						LastSeen:    packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "daemonset-service1",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              "deployment-service2",
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "deployment-service3",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              "daemonset-service1",
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service1",
				},
				{
					Name:              "deployment-service2",
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "pod4",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "",
					Kind:    "Pod",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              "deployment-service2",
					Namespace:         s.TestNamespace,
					KubernetesService: "svc-service2",
				},
			},
		},
	})
}

func (s *ResolverTestSuite) TestIntents() {
	s.AddDeploymentWithService("service1", []string{"1.1.21.1"}, map[string]string{"app": "service1"}, "10.0.0.10")
	s.AddDeploymentWithService("service2", []string{"1.1.21.2"}, map[string]string{"app": "service2"}, "10.0.0.11")
	s.AddDaemonSetWithService("service3", []string{"1.1.21.3"}, map[string]string{"app": "service3"}, "10.0.0.12")
	s.AddPod("pod4", "1.1.21.4", nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	packetTime := time.Now().Add(time.Minute)
	_, err := test_gql_client.ReportCaptureResults(context.Background(), s.client, test_gql_client.CaptureResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "1.1.21.1",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.21.3",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service1.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.21.4",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	logrus.Info("Testing Intents query")
	res, err := test_gql_client.Intents(context.Background(), s.client, nil, nil, nil, true, nil)
	s.Require().NoError(err)
	logrus.Info("Testing Intents query done")
	logrus.Infof("Intents: %v", res.Intents)

	expectedIntents := []test_gql_client.IntentsIntentsIntent{
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      "deployment-service1",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      "deployment-service2",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
				KubernetesService: "svc-service2",
			},
		},
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      "daemonset-service3",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      "deployment-service1",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
				KubernetesService: "svc-service1",
			},
		}, {
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      "daemonset-service3",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      "deployment-service2",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
				KubernetesService: "svc-service2",
			},
		},
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      "pod4",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "",
					Kind:    "Pod",
					Version: "v1",
				},
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      "deployment-service2",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
				KubernetesService: "svc-service2",
			},
		},
	}
	s.Require().ElementsMatch(res.Intents, expectedIntents)
}

func (s *ResolverTestSuite) TestIntentsFilterByServer() {
	service1Name := "service1"
	service1IP := "1.1.18.1"
	s.AddDeploymentWithService(service1Name, []string{service1IP}, map[string]string{"app": service1Name}, "10.0.0.13")
	service2Name := "service2"
	service2IP := "1.1.18.2"
	s.AddDeploymentWithService(service2Name, []string{service2IP}, map[string]string{"app": service2Name}, "10.0.0.14")
	service3Name := "service3"
	service3IP := "1.1.18.3"
	s.AddDaemonSetWithService(service3Name, []string{service3IP}, map[string]string{"app": service3Name}, "10.0.0.15")
	podServiceName := "pod4"
	podIP := "1.1.18.4"
	s.AddPod(podServiceName, podIP, nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	packetTime := time.Now().Add(time.Minute)
	_, err := test_gql_client.ReportCaptureResults(context.Background(), s.client, test_gql_client.CaptureResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: service1IP,
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: service3IP,
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service1.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: podIP,
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("svc-service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	logrus.Info("Waiting for report to be processed")
	serverFilter := &test_gql_client.ServerFilter{
		Name:      fmt.Sprintf("deployment-%s", service1Name),
		Namespace: s.TestNamespace,
	}
	res, err := test_gql_client.Intents(context.Background(), s.client, []string{s.TestNamespace}, nil, nil, true, serverFilter)
	s.Require().NoError(err)
	logrus.Info("Report processed")
	logrus.Infof("Intents: %v", res.Intents)

	expectedIntents := []test_gql_client.IntentsIntentsIntent{
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", service3Name),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", service1Name),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
				KubernetesService: "svc-service1",
			},
		}, {
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", service3Name),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", service2Name),
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
				KubernetesService: "svc-service2",
			},
		},
	}
	s.Require().ElementsMatch(res.Intents, expectedIntents)
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(ResolverTestSuite))
}
