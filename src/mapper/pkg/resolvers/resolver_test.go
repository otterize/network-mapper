package resolvers

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/mapper/pkg/resolvers/test_gql_client"
	"github.com/otterize/network-mapper/src/shared/testbase"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"testing"
)

type ResolverTestSuite struct {
	testbase.ControllerManagerTestSuiteBase
	server        *httptest.Server
	client        graphql.Client
	kubeFinder    *kubefinder.KubeFinder
	intentsHolder *IntentsHolder
}

func (s *ResolverTestSuite) SetupTest() {
	s.ControllerManagerTestSuiteBase.SetupTest()
	e := echo.New()
	var err error
	s.kubeFinder, err = kubefinder.NewKubeFinder(s.Mgr)
	s.Require().NoError(err)
	s.intentsHolder = NewIntentsHolder(s.Mgr.GetClient(), IntentsHolderConfig{StoreConfigMap: config.StoreConfigMapDefault, Namespace: s.TestNamespace})
	resolver := NewResolver(s.kubeFinder, serviceidresolver.NewResolver(s.Mgr.GetClient()), s.intentsHolder)
	resolver.Register(e)
	s.server = httptest.NewServer(e)
	s.client = graphql.NewClient(s.server.URL+"/query", s.server.Client())
}

func (s *ResolverTestSuite) TestReportCaptureResults() {
	s.AddDeploymentWithService("service1", []string{"1.1.1.1"}, map[string]string{"app": "service1"})
	s.AddDeploymentWithService("service2", []string{"1.1.1.2"}, map[string]string{"app": "service2"})
	s.AddDeploymentWithService("service3", []string{"1.1.1.3"}, map[string]string{"app": "service3"})
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	_, err := test_gql_client.ReportCaptureResults(context.Background(), s.client, test_gql_client.CaptureResults{
		Results: []test_gql_client.CaptureResultForSrcIp{
			{
				SrcIp:        "1.1.1.1",
				Destinations: []string{fmt.Sprintf("service2.%s.svc.cluster.local", s.TestNamespace)},
			},
			{
				SrcIp: "1.1.1.3",
				Destinations: []string{
					fmt.Sprintf("service1.%s.svc.cluster.local", s.TestNamespace),
					fmt.Sprintf("service2.%s.svc.cluster.local", s.TestNamespace),
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
	})
}

func (s *ResolverTestSuite) TestSocketScanResults() {
	s.AddDeploymentWithService("service1", []string{"1.1.2.1"}, map[string]string{"app": "service1"})
	s.AddDeploymentWithService("service2", []string{"1.1.2.2"}, map[string]string{"app": "service2"})
	s.AddDeploymentWithService("service3", []string{"1.1.2.3"}, map[string]string{"app": "service3"})
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	_, err := test_gql_client.ReportSocketScanResults(context.Background(), s.client, test_gql_client.SocketScanResults{
		Results: []test_gql_client.SocketScanResultForSrcIp{
			{
				SrcIp:   "1.1.2.1",
				DestIps: []string{"1.1.2.2"},
			},
			{
				SrcIp:   "1.1.2.3",
				DestIps: []string{"1.1.2.1", "1.1.2.2"},
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
	})
}

func (s *ResolverTestSuite) TestLoadStore() {
	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().Len(res.ServiceIntents, 0)

	s.AddDeploymentWithService("service1", []string{"1.1.3.1"}, map[string]string{"app": "service1"})
	s.AddDeploymentWithService("service2", []string{"1.1.3.2"}, map[string]string{"app": "service2"})
	s.AddDeploymentWithService("service3", []string{"1.1.3.3"}, map[string]string{"app": "service3"})
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))
	_, err = test_gql_client.ReportSocketScanResults(context.Background(), s.client, test_gql_client.SocketScanResults{
		Results: []test_gql_client.SocketScanResultForSrcIp{
			{
				SrcIp:   "1.1.3.1",
				DestIps: []string{"1.1.3.2"},
			},
			{
				SrcIp:   "1.1.3.3",
				DestIps: []string{"1.1.3.1", "1.1.3.2"},
			},
		},
	})
	s.Require().NoError(err)
	res, err = test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().Len(res.ServiceIntents, 2)

	// create a new resolver and see LoadStore works
	resolver := NewResolver(s.kubeFinder, serviceidresolver.NewResolver(s.Mgr.GetClient()), NewIntentsHolder(s.Mgr.GetClient(), IntentsHolderConfig{StoreConfigMap: config.StoreConfigMapDefault, Namespace: s.TestNamespace}))
	intents, err := resolver.Query().ServiceIntents(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().Len(intents, 0)

	err = resolver.LoadStore(context.Background())
	s.Require().NoError(err)
	intents, err = resolver.Query().ServiceIntents(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().Len(intents, 2)
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(ResolverTestSuite))
}
