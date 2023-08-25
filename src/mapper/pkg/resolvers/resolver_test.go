package resolvers

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/mapper/pkg/resolvers/test_gql_client"
	"github.com/otterize/network-mapper/src/shared/testbase"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"testing"
	"time"
)

type ResolverTestSuite struct {
	testbase.ControllerManagerTestSuiteBase
	server        *httptest.Server
	client        graphql.Client
	kubeFinder    *kubefinder.KubeFinder
	intentsHolder *intentsstore.IntentsHolder
}

func (s *ResolverTestSuite) SetupTest() {
	s.ControllerManagerTestSuiteBase.SetupTest()
	e := echo.New()
	var err error
	s.kubeFinder, err = kubefinder.NewKubeFinder(s.Mgr)
	s.Require().NoError(err)
	s.intentsHolder = intentsstore.NewIntentsHolder()
	resolver := NewResolver(s.kubeFinder, serviceidresolver.NewResolver(s.Mgr.GetClient()), s.intentsHolder)
	resolver.Register(e)
	s.server = httptest.NewServer(e)
	s.client = graphql.NewClient(s.server.URL+"/query", s.server.Client())
}

func (s *ResolverTestSuite) TestReportCaptureResults() {
	s.AddDeploymentWithService("service1", []string{"1.1.1.1"}, map[string]string{"app": "service1"})
	s.AddDeploymentWithService("service2", []string{"1.1.1.2"}, map[string]string{"app": "service2"})
	s.AddDaemonSetWithService("service3", []string{"1.1.1.3"}, map[string]string{"app": "service3"})
	s.AddPod("pod4", "1.1.1.4", nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	packetTime := time.Now().Add(time.Minute)
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
			{
				SrcIp: "1.1.1.3",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("service1.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
					{
						Destination: fmt.Sprintf("service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.1.4",
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
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "service1",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:      "service2",
					Namespace: s.TestNamespace,
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "service3",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:      "service1",
					Namespace: s.TestNamespace,
				},
				{
					Name:      "service2",
					Namespace: s.TestNamespace,
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
					Name:      "service2",
					Namespace: s.TestNamespace,
				},
			},
		},
	})
}

func (s *ResolverTestSuite) TestReportCaptureResultsIgnoreOldPacket() {
	s.AddDeploymentWithService("service1", []string{"1.1.1.1"}, map[string]string{"app": "service1"})
	s.AddDeploymentWithService("service2", []string{"1.1.1.2"}, map[string]string{"app": "service2"})
	s.AddDaemonSetWithService("service3", []string{"1.1.1.3"}, map[string]string{"app": "service3"})
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
	s.AddDaemonSetWithService("service1", []string{"1.1.2.1"}, map[string]string{"app": "service1"})
	s.AddDeploymentWithService("service2", []string{"1.1.2.2"}, map[string]string{"app": "service2"})
	s.AddDeploymentWithService("service3", []string{"1.1.2.3"}, map[string]string{"app": "service3"})
	s.AddPod("pod4", "1.1.2.4", nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	packetTime := time.Now().Add(time.Minute)

	_, err := test_gql_client.ReportSocketScanResults(context.Background(), s.client, test_gql_client.SocketScanResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "1.1.2.1",
				Destinations: []test_gql_client.Destination{
					{
						Destination: "1.1.2.2",
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.2.3",
				Destinations: []test_gql_client.Destination{
					{
						Destination: "1.1.2.1",
						LastSeen:    packetTime,
					},
					{
						Destination: "1.1.2.2",
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.2.4",
				Destinations: []test_gql_client.Destination{
					{
						Destination: "1.1.2.2",
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
				Name:      "service1",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:      "service2",
					Namespace: s.TestNamespace,
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "service3",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:      "service1",
					Namespace: s.TestNamespace,
				},
				{
					Name:      "service2",
					Namespace: s.TestNamespace,
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
					Name:      "service2",
					Namespace: s.TestNamespace,
				},
			},
		},
	})
}

func (s *ResolverTestSuite) TestIntents() {
	s.AddDeploymentWithService("service1", []string{"1.1.21.1"}, map[string]string{"app": "service1"})
	s.AddDeploymentWithService("service2", []string{"1.1.21.2"}, map[string]string{"app": "service2"})
	s.AddDaemonSetWithService("service3", []string{"1.1.21.3"}, map[string]string{"app": "service3"})
	s.AddPod("pod4", "1.1.21.4", nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	packetTime := time.Now().Add(time.Minute)
	_, err := test_gql_client.ReportCaptureResults(context.Background(), s.client, test_gql_client.CaptureResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "1.1.21.1",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.21.3",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("service1.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
					{
						Destination: fmt.Sprintf("service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: "1.1.21.4",
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

	logrus.Info("Testing Intents query")
	res, err := test_gql_client.Intents(context.Background(), s.client, nil, nil, nil, true, nil)
	s.Require().NoError(err)
	logrus.Info("Testing Intents query done")
	logrus.Infof("Intents: %v", res.Intents)

	expectedIntents := []test_gql_client.IntentsIntentsIntent{
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      "service1",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      "service2",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
		},
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      "service3",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      "service1",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
		}, {
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      "service3",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      "service2",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
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
				Name:      "service2",
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
		},
	}
	s.Require().ElementsMatch(res.Intents, expectedIntents)
}

func (s *ResolverTestSuite) TestIntentsFilterByServer() {
	service1Name := "service1"
	service1IP := "1.1.18.1"
	s.AddDeploymentWithService(service1Name, []string{service1IP}, map[string]string{"app": service1Name})
	service2Name := "service2"
	service2IP := "1.1.18.2"
	s.AddDeploymentWithService(service2Name, []string{service2IP}, map[string]string{"app": service2Name})
	service3Name := "service3"
	service3IP := "1.1.18.3"
	s.AddDaemonSetWithService(service3Name, []string{service3IP}, map[string]string{"app": service3Name})
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
						Destination: fmt.Sprintf("service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: service3IP,
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("service1.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
					{
						Destination: fmt.Sprintf("service2.%s.svc.cluster.local", s.TestNamespace),
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: podIP,
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

	logrus.Info("Waiting for report to be processed")
	serverFilter := &test_gql_client.ServerFilter{
		Name:      service1Name,
		Namespace: s.TestNamespace,
	}
	res, err := test_gql_client.Intents(context.Background(), s.client, []string{s.TestNamespace}, nil, nil, true, serverFilter)
	s.Require().NoError(err)
	logrus.Info("Report processed")
	logrus.Infof("Intents: %v", res.Intents)

	expectedIntents := []test_gql_client.IntentsIntentsIntent{
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      service3Name,
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      service1Name,
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
		}, {
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      service3Name,
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "DaemonSet",
					Version: "v1",
				},
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      service2Name,
				Namespace: s.TestNamespace,
				PodOwnerKind: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
		},
	}
	s.Require().ElementsMatch(res.Intents, expectedIntents)
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(ResolverTestSuite))
}
