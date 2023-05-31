package resolvers

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/otterize/network-mapper/src/mapper/pkg/resolvers/test_gql_client"
	"github.com/otterize/network-mapper/src/shared/kubefinder"
	"github.com/otterize/network-mapper/src/shared/testbase"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"testing"
)

type ResolverTestSuite struct {
	testbase.ControllerManagerTestSuiteBase
	server        *httptest.Server
	client        graphql.Client
	kubeFinder    kubefinder.KubeFinder
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

	_, err := test_gql_client.ReportCaptureResults(context.Background(), s.client, test_gql_client.CaptureResults{
		Results: []test_gql_client.CaptureResultForSrcIp{
			{
				SrcIp: "1.1.1.1",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("service2.%s.svc.cluster.local", s.TestNamespace),
					},
				},
			},
			{
				SrcIp: "1.1.1.3",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("service1.%s.svc.cluster.local", s.TestNamespace),
					},
					{
						Destination: fmt.Sprintf("service2.%s.svc.cluster.local", s.TestNamespace),
					},
				},
			},
			{
				SrcIp: "1.1.1.4",
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("service2.%s.svc.cluster.local", s.TestNamespace),
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

func (s *ResolverTestSuite) TestSocketScanResults() {
	s.AddDaemonSetWithService("service1", []string{"1.1.2.1"}, map[string]string{"app": "service1"})
	s.AddDeploymentWithService("service2", []string{"1.1.2.2"}, map[string]string{"app": "service2"})
	s.AddDeploymentWithService("service3", []string{"1.1.2.3"}, map[string]string{"app": "service3"})
	s.AddPod("pod4", "1.1.2.4", nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	_, err := test_gql_client.ReportSocketScanResults(context.Background(), s.client, test_gql_client.SocketScanResults{
		Results: []test_gql_client.SocketScanResultForSrcIp{
			{
				SrcIp: "1.1.2.1",
				DestIps: []test_gql_client.Destination{
					{
						Destination: "1.1.2.2",
					},
				},
			},
			{
				SrcIp: "1.1.2.3",
				DestIps: []test_gql_client.Destination{
					{
						Destination: "1.1.2.1",
					},
					{
						Destination: "1.1.2.2",
					},
				},
			},
			{
				SrcIp: "1.1.2.4",
				DestIps: []test_gql_client.Destination{
					{
						Destination: "1.1.2.2",
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

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(ResolverTestSuite))
}
