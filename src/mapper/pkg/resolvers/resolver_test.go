package resolvers

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/awsintentsholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/azureintentsholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnscache"
	"github.com/otterize/network-mapper/src/mapper/pkg/externaltrafficholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/incomingtrafficholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/mapper/pkg/resolvers/test_gql_client"
	"github.com/otterize/network-mapper/src/shared/testbase"
	"github.com/otterize/nilable"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http/httptest"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	awsIntentsHolder             *awsintentsholder.AWSIntentsHolder
	azureIntentsHolder           *azureintentsholder.AzureIntentsHolder
	externalTrafficIntentsHolder *externaltrafficholder.ExternalTrafficIntentsHolder
	incomingTrafficIntentsHolder *incomingtrafficholder.IncomingTrafficIntentsHolder
	resolverCtx                  context.Context
	resolverCtxCancel            context.CancelFunc
	resolver                     *Resolver
}

func (s *ResolverTestSuite) SetupTest() {
	s.ControllerManagerTestSuiteBase.SetupTest()
	s.resolverCtx, s.resolverCtxCancel = context.WithCancel(context.Background())
	e := echo.New()
	var err error
	s.kubeFinder, err = kubefinder.NewKubeFinder(context.Background(), s.Mgr)
	s.Require().NoError(err)
	s.intentsHolder = intentsstore.NewIntentsHolder()
	s.externalTrafficIntentsHolder = externaltrafficholder.NewExternalTrafficIntentsHolder()
	s.incomingTrafficIntentsHolder = incomingtrafficholder.NewIncomingTrafficIntentsHolder()
	s.awsIntentsHolder = awsintentsholder.New()
	s.azureIntentsHolder = azureintentsholder.New()
	dnsCache := dnscache.NewDNSCache()

	resolver := NewResolver(
		s.kubeFinder,
		serviceidresolver.NewResolver(s.Mgr.GetClient()),
		s.intentsHolder,
		s.externalTrafficIntentsHolder,
		s.awsIntentsHolder,
		s.azureIntentsHolder,
		dnsCache,
		s.incomingTrafficIntentsHolder,
	)

	resolver.Register(e)
	s.resolver = resolver
	go func() {
		err := resolver.RunForever(s.resolverCtx)
		if err != nil {
			logrus.WithError(err).Panic("failed to run resolver")
		}
	}()
	s.server = httptest.NewServer(e)
	s.client = graphql.NewClient(s.server.URL+"/query", s.server.Client())
}

func (s *ResolverTestSuite) TearDownTest() {
	s.ControllerManagerTestSuiteBase.TearDownTest()
	s.resolverCtxCancel()
}

func (s *ResolverTestSuite) waitForCaptureResultsProcessed(timeout time.Duration) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case <-ctxTimeout.Done():
		s.Require().Fail("Timed out waiting for capture results to be processed")
	case <-s.resolver.gotResultsCtx.Done():
		return
	}
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

	s.waitForCaptureResultsProcessed(10 * time.Second)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", "service1"),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", "service3"),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "DaemonSet",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service1"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service1"),
				},
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "pod4",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.Nilable[string]{},
					Kind:    "Pod",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
	})
}

func (s *ResolverTestSuite) TestReportIncomingTraffic() {
	serviceIp := "10.0.0.16"
	service2Ip := "10.0.0.17"
	serviceExternalIP := "34.10.0.12"
	s.AddDeploymentWithIngressService("service1", []string{"1.1.1.1"}, map[string]string{"app": "service1"}, serviceIp, serviceExternalIP, 9090)

	// Add service2 to make sure our resolution handles the case where the load-balancer services share the same external IP
	s.AddDeploymentWithIngressService("service2", []string{"1.1.1.2"}, map[string]string{"app": "service1"}, service2Ip, serviceExternalIP, 1337)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))
	externalInternetServerIP := "142.198.10.38"

	packetTime := time.Now().Add(time.Minute)
	tcpResults := test_gql_client.CaptureTCPResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: externalInternetServerIP,
				Destinations: []test_gql_client.Destination{
					{
						DestinationIP:   nilable.From(serviceExternalIP),
						DestinationPort: nilable.From(9090),
						LastSeen:        packetTime,
					},
				},
			},
		},
	}

	_, err := test_gql_client.ReportTCPCaptureResults(context.Background(), s.client, tcpResults)
	s.Require().NoError(err)

	s.waitForCaptureResultsProcessed(10 * time.Second)
	intents := s.resolver.incomingTrafficHolder.GetNewIntentsSinceLastGet()
	s.Require().Len(intents, 1)
	s.Require().Equal(externalInternetServerIP, intents[0].Intent.IP)
	s.Require().Equal("deployment-service1", intents[0].Intent.Server.Name)
	s.Require().Equal(lo.ToPtr("svc-service1"), intents[0].Intent.Server.KubernetesService)
	s.Require().Equal(s.TestNamespace, intents[0].Intent.Server.Namespace)
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

	s.waitForCaptureResultsProcessed(10 * time.Second)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", "service1"),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", "service3"),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "DaemonSet",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service1"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service1"),
				},
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
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
	err = s.Mgr.GetClient().Patch(context.Background(), &podToUpdate, client.MergeFrom(pod))
	s.Require().NoError(err)

	interval := 1 * time.Second
	timeout := 10 * time.Second
	s.Require().NoError(wait.PollUntilContextTimeout(
		context.Background(),
		interval,
		timeout,
		true,
		func(ctx context.Context) (done bool, err error) {
			var readPod v1.Pod
			err = s.Mgr.GetClient().Get(ctx, types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}, &readPod)
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, errors.Wrap(err)
			}

			if !slices.Contains(readPod.Finalizers, "intents.otterize.com/finalizer-so-that-object-cant-be-deleted-for-this-test") {
				return false, nil
			}
			return true, nil
		}),
	)

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

	s.waitForCaptureResultsProcessed(10 * time.Second)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", "service1"),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", "service3"),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "DaemonSet",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service1"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service1"),
				},
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
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

	s.waitForCaptureResultsProcessed(10 * time.Second)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", "service1"),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", "service3"),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "DaemonSet",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service1"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service1"),
				},
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "pod4",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.Nilable[string]{},
					Kind:    "Pod",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              fmt.Sprintf("deployment-%s", "service2"),
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
	})
}

func (s *ResolverTestSuite) TestReportTCPCaptureResults() {
	const (
		service1IP    = "10.0.0.16"
		service2IP    = "10.0.0.17"
		service3IP    = "10.0.0.18"
		service1PodIP = "1.1.1.1"
		service2PodIP = "1.1.1.2"
		service3PodIP = "1.1.1.3"
		service4PodIP = "1.1.1.4"
	)

	s.AddDaemonSetWithService("service1", []string{service1PodIP}, map[string]string{"app": "service1"}, service1IP)
	s.AddDeploymentWithService("service2", []string{service2PodIP}, map[string]string{"app": "service2"}, service2IP)
	s.AddDeploymentWithService("service3", []string{service3PodIP}, map[string]string{"app": "service3"}, service3IP)
	s.AddPod("pod4", service4PodIP, nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	packetTime := time.Now().Add(time.Minute)
	_, err := test_gql_client.ReportTCPCaptureResults(context.Background(), s.client, test_gql_client.CaptureTCPResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: service1PodIP,
				Destinations: []test_gql_client.Destination{
					{
						Destination: service2IP,
						LastSeen:    packetTime,
					},
				},
			},
			{
				SrcIp: service3PodIP,
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
				SrcIp: service4PodIP,
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

	s.waitForCaptureResultsProcessed(10 * time.Second)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	expectedIntents := []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "daemonset-service1",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "DaemonSet",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              "deployment-service2",
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "deployment-service3",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              "daemonset-service1",
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service1"),
				},
				{
					Name:              "deployment-service2",
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "pod4",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.Nilable[string]{},
					Kind:    "Pod",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              "deployment-service2",
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
	}
	s.Require().ElementsMatch(res.ServiceIntents, expectedIntents)
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

	s.waitForCaptureResultsProcessed(10 * time.Second)

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

	s.waitForCaptureResultsProcessed(10 * time.Second)

	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "daemonset-service1",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "DaemonSet",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              "deployment-service2",
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "deployment-service3",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              "daemonset-service1",
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service1"),
				},
				{
					Name:              "deployment-service2",
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
				},
			},
		},
		{
			Client: test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentity{
				Name:      "pod4",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.ServiceIntentsServiceIntentsClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.Nilable[string]{},
					Kind:    "Pod",
					Version: "v1",
				}),
			},
			Intents: []test_gql_client.ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity{
				{
					Name:              "deployment-service2",
					Namespace:         s.TestNamespace,
					KubernetesService: nilable.From("svc-service2"),
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

	s.waitForCaptureResultsProcessed(10 * time.Second)

	logrus.Info("Testing Intents query")
	res, err := test_gql_client.Intents(context.Background(), s.client, nil, nil, nil, nilable.From(true), nil)
	s.Require().NoError(err)
	logrus.Info("Testing Intents query done")
	logrus.Infof("Intents: %v", res.Intents)

	expectedIntents := []test_gql_client.IntentsIntentsIntent{
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      "deployment-service1",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      "deployment-service2",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
				KubernetesService: nilable.From("svc-service2"),
			},
		},
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      "daemonset-service3",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "DaemonSet",
					Version: "v1",
				}),
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      "deployment-service1",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
				KubernetesService: nilable.From("svc-service1"),
			},
		}, {
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      "daemonset-service3",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "DaemonSet",
					Version: "v1",
				}),
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      "deployment-service2",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
				KubernetesService: nilable.From("svc-service2"),
			},
		},
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      "pod4",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.Nilable[string]{},
					Kind:    "Pod",
					Version: "v1",
				}),
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      "deployment-service2",
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
				KubernetesService: nilable.From("svc-service2"),
			},
		},
	}
	s.Require().ElementsMatch(res.Intents, expectedIntents)
}

func (s *ResolverTestSuite) TestIntentsToApiServerDNS() {
	service := s.GetAPIServerService()
	s.Require().NotNil(service)

	podServiceName := "client-pod"
	podIP := "1.1.19.1"
	s.AddPod(podServiceName, podIP, nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	_, err := test_gql_client.ReportCaptureResults(context.Background(), s.client, test_gql_client.CaptureResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: podIP,
				Destinations: []test_gql_client.Destination{
					{
						Destination: fmt.Sprintf("%s.%s.svc.cluster.local", service.GetName(), service.GetNamespace()),
						LastSeen:    time.Now().Add(time.Minute),
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	s.waitForCaptureResultsProcessed(10 * time.Second)

	res, err := test_gql_client.Intents(context.Background(), s.client, []string{}, nil, nil, nilable.From(true), nil)
	s.Require().NoError(err)
	logrus.Info("Report processed")
	logrus.Infof("Intents: %v", res.Intents)

	expectedIntents := []test_gql_client.IntentsIntentsIntent{
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      podServiceName,
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.Nilable[string]{},
					Kind:    "Pod",
					Version: "v1",
				}),
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:              service.GetName(),
				Namespace:         service.GetNamespace(),
				KubernetesService: nilable.From(service.GetName()),
			},
		},
	}
	s.Require().ElementsMatch(res.Intents, expectedIntents)
}

func (s *ResolverTestSuite) TestIntentsToApiServerSocketScan() {
	service := s.GetAPIServerService()
	s.Require().NotNil(service)

	podServiceName := "client-pod"
	podIP := "1.1.19.1"
	s.AddPod(podServiceName, podIP, nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	_, err := test_gql_client.ReportSocketScanResults(context.Background(), s.client, test_gql_client.SocketScanResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: podIP,
				Destinations: []test_gql_client.Destination{
					{
						Destination: service.Spec.ClusterIP,
						LastSeen:    time.Now().Add(time.Minute),
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	s.waitForCaptureResultsProcessed(10 * time.Second)

	res, err := test_gql_client.Intents(context.Background(), s.client, []string{}, nil, nil, nilable.From(true), nil)
	s.Require().NoError(err)
	logrus.Info("Report processed")
	logrus.Infof("Intents: %v", res.Intents)

	expectedIntents := []test_gql_client.IntentsIntentsIntent{
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      podServiceName,
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.Nilable[string]{},
					Kind:    "Pod",
					Version: "v1",
				}),
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:              service.GetName(),
				Namespace:         service.GetNamespace(),
				KubernetesService: nilable.From(service.GetName()),
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

	s.waitForCaptureResultsProcessed(10 * time.Second)

	logrus.Info("Waiting for report to be processed")
	serverFilter := &test_gql_client.ServerFilter{
		Name:      fmt.Sprintf("deployment-%s", service1Name),
		Namespace: s.TestNamespace,
	}
	res, err := test_gql_client.Intents(context.Background(), s.client, []string{s.TestNamespace}, nil, nil, nilable.From(true), serverFilter)
	s.Require().NoError(err)
	logrus.Info("Report processed")
	logrus.Infof("Intents: %v", res.Intents)

	expectedIntents := []test_gql_client.IntentsIntentsIntent{
		{
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", service3Name),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "DaemonSet",
					Version: "v1",
				}),
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", service1Name),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
				KubernetesService: nilable.From("svc-service1"),
			},
		}, {
			Client: test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentity{
				Name:      fmt.Sprintf("daemonset-%s", service3Name),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentClientOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "DaemonSet",
					Version: "v1",
				}),
			},
			Server: test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentity{
				Name:      fmt.Sprintf("deployment-%s", service2Name),
				Namespace: s.TestNamespace,
				PodOwnerKind: nilable.From(test_gql_client.IntentsIntentsIntentServerOtterizeServiceIdentityPodOwnerKindGroupVersionKind{
					Group:   nilable.From("apps"),
					Kind:    "Deployment",
					Version: "v1",
				}),
				KubernetesService: nilable.From("svc-service2"),
			},
		},
	}
	s.Require().ElementsMatch(res.Intents, expectedIntents)
}

func (s *ResolverTestSuite) TestResolveOtterizeIdentityIgnoreHostNetworkPods() {
	// Setup
	serviceName := "test-service"
	serviceIP := "10.0.0.10"
	podIP := "1.1.1.3"

	pod3 := s.AddPodWithHostNetwork("pod3", podIP, map[string]string{"app": "test"}, nil, true)
	s.AddClusterIPService(serviceName, map[string]string{"app": "test"}, serviceIP, []*v1.Pod{pod3})
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	service := &v1.Service{}
	err := s.Mgr.GetClient().Get(context.Background(), types.NamespacedName{Name: "svc-" + serviceName, Namespace: s.TestNamespace}, service)
	s.Require().NoError(err)

	lastSeen := time.Now().Add(time.Minute)
	_, ok, err := s.resolver.resolveOtterizeIdentityForDestinationAddress(context.Background(), model.Destination{LastSeen: lastSeen, Destination: fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace)})
	s.Require().False(ok)
	s.Require().NoError(err)

}

func (s *ResolverTestSuite) TestTCPResultsFromHostNetworkPodsIgnored() {
	// Add external service
	internalServiceIp := "10.0.0.16"
	externalServiceIP := "34.10.0.12"
	servicePort := 9090
	s.AddDeploymentWithIngressService("service1", []string{"1.1.1.1"}, map[string]string{"app": "service1"}, internalServiceIp, externalServiceIP, servicePort)

	// Add host network pod
	hostNetworkServiceName := "test-service"
	hostNetworkServiceIP := "10.0.0.10"
	hostNetworkPodIP := "1.1.1.3"
	pod := s.AddPodWithHostNetwork("pod", hostNetworkPodIP, map[string]string{"app": "test"}, nil, true)
	s.AddClusterIPService(hostNetworkServiceName, map[string]string{"app": "test"}, hostNetworkServiceIP, []*v1.Pod{pod})

	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	// Report TCP results of traffic from host network pod to external service
	packetTime := time.Now().Add(time.Minute)
	_, err := test_gql_client.ReportTCPCaptureResults(context.Background(), s.client, test_gql_client.CaptureTCPResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: hostNetworkPodIP,
				Destinations: []test_gql_client.Destination{
					{
						Destination:     externalServiceIP,
						DestinationIP:   nilable.From(externalServiceIP),
						DestinationPort: nilable.From(servicePort),
						LastSeen:        packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	s.waitForCaptureResultsProcessed(10 * time.Second)

	// Verify that the traffic from host network pod to external service is ignored
	res, err := test_gql_client.ServiceIntents(context.Background(), s.client, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(res.ServiceIntents, []test_gql_client.ServiceIntentsServiceIntents{})

	intents := s.resolver.incomingTrafficHolder.GetNewIntentsSinceLastGet()
	s.Require().Empty(intents)
}

func (s *ResolverTestSuite) TestTCPResultsFromExternalToPodSavedAsIncoming() {
	// Create deployment
	deploymentName := "coolz"
	podIP := "1.1.1.3"
	dep, _ := s.AddDeployment(deploymentName, []string{podIP}, map[string]string{"app": "coolz"})
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	// Report TCP results of traffic from external ip to pod
	packetTime := time.Now().Add(time.Minute)
	_, err := test_gql_client.ReportTCPCaptureResults(context.Background(), s.client, test_gql_client.CaptureTCPResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "8.8.8.8",
				Destinations: []test_gql_client.Destination{
					{
						Destination:     podIP,
						DestinationIP:   nilable.From(podIP),
						DestinationPort: nilable.From(80),
						LastSeen:        packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	s.waitForCaptureResultsProcessed(10 * time.Second)

	// Verify that the traffic from external ip to pod is saved as incoming
	intents := s.resolver.incomingTrafficHolder.GetNewIntentsSinceLastGet()
	s.Require().Len(intents, 1)
	s.Require().Equal(dep.Name, intents[0].Intent.Server.Name)
	s.Require().Equal(dep.Namespace, intents[0].Intent.Server.Namespace)
	s.Require().Equal("8.8.8.8", intents[0].Intent.IP)
}

func (s *ResolverTestSuite) TestTCPResultsFromExternalToPodSaveIfDestinationNameRight() {
	// Create deployment
	deploymentName := "coolz"
	podIP := "1.1.1.3"
	dep, pods := s.AddDeployment(deploymentName, []string{podIP}, map[string]string{"app": "coolz"})
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	// Report TCP results of traffic from external ip to pod
	packetTime := time.Now().Add(time.Minute)
	_, err := test_gql_client.ReportTCPCaptureResults(context.Background(), s.client, test_gql_client.CaptureTCPResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "8.8.8.8",
				Destinations: []test_gql_client.Destination{
					{
						Destination:     pods[0].Name,
						DestinationIP:   nilable.From(podIP),
						DestinationPort: nilable.From(80),
						LastSeen:        packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	s.waitForCaptureResultsProcessed(10 * time.Second)

	// Verify that the traffic from external ip to pod is saved as incoming
	intents := s.resolver.incomingTrafficHolder.GetNewIntentsSinceLastGet()
	s.Require().Len(intents, 1)
	s.Require().Equal(dep.Name, intents[0].Intent.Server.Name)
	s.Require().Equal(dep.Namespace, intents[0].Intent.Server.Namespace)
	s.Require().Equal("8.8.8.8", intents[0].Intent.IP)
}

func (s *ResolverTestSuite) TestTCPResultsFromExternalToPodSkipIfDestinationNameWrong() {
	// Create deployment
	deploymentName := "coolz"
	podIP := "1.1.1.3"
	_, pods := s.AddDeployment(deploymentName, []string{podIP}, map[string]string{"app": "coolz"})
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))

	// Report TCP results of traffic from external ip to pod
	packetTime := time.Now().Add(time.Minute)
	_, err := test_gql_client.ReportTCPCaptureResults(context.Background(), s.client, test_gql_client.CaptureTCPResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "8.8.8.8",
				Destinations: []test_gql_client.Destination{
					{
						Destination:     pods[0].Name + "-wrong",
						DestinationIP:   nilable.From(podIP),
						DestinationPort: nilable.From(80),
						LastSeen:        packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)

	s.waitForCaptureResultsProcessed(10 * time.Second)

	// Verify that the traffic from external ip to pod is saved as incoming
	intents := s.resolver.incomingTrafficHolder.GetNewIntentsSinceLastGet()
	s.Require().Len(intents, 0)
}

func (s *ResolverTestSuite) TestTCPResultsFromExternalToLoadBalancerServiceUsingNodeIpAndPortSavedAsIncoming() {
	// Create deployment
	deploymentName := "coolz"
	podIP := "1.1.1.3"
	serviceIP := "10.0.0.10"
	dep, pods := s.AddDeployment(deploymentName, []string{podIP}, map[string]string{"app": "coolz"})
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))
	svc := s.AddLoadBalancerService("coolz", map[string]string{"app": "coolz"}, serviceIP, pods)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))
	nodePort := svc.Spec.Ports[0].NodePort

	nodes := v1.NodeList{}
	err := s.Mgr.GetClient().List(context.Background(), &nodes)
	s.Require().NoError(err)
	s.Require().NotEmpty(nodes.Items)
	nodeIP := nodes.Items[0].Status.Addresses[0].Address

	// Report TCP results of traffic from external ip to nodeIp:nodePort
	packetTime := time.Now().Add(time.Minute)
	_, err = test_gql_client.ReportTCPCaptureResults(context.Background(), s.client, test_gql_client.CaptureTCPResults{
		Results: []test_gql_client.RecordedDestinationsForSrc{
			{
				SrcIp: "8.8.8.8",
				Destinations: []test_gql_client.Destination{
					{
						Destination:     nodeIP,
						DestinationIP:   nilable.From(nodeIP),
						DestinationPort: nilable.From(int(nodePort)),
						LastSeen:        packetTime,
					},
				},
			},
		},
	})
	s.Require().NoError(err)
	s.waitForCaptureResultsProcessed(10 * time.Second)

	// Verify that the traffic from external ip to nodeIp:nodePort is saved as incoming
	intents := s.resolver.incomingTrafficHolder.GetNewIntentsSinceLastGet()
	s.Require().Len(intents, 1)
	s.Require().Equal(dep.Name, intents[0].Intent.Server.Name)
	s.Require().Equal(dep.Namespace, intents[0].Intent.Server.Namespace)
	s.Require().Equal("8.8.8.8", intents[0].Intent.IP)
}

func (s *ResolverTestSuite) TestResolveOtterizeIdentityFilterSrcDestinationsByCreationTimestamp() {
	podIP := "1.1.1.3"
	pod := s.AddPod("pod3", podIP, nil, nil)
	s.Require().True(s.Mgr.GetCache().WaitForCacheSync(context.Background()))
	recorededDestinationForSrc := &model.RecordedDestinationsForSrc{
		SrcIP: podIP,
		Destinations: []model.Destination{
			{
				Destination: "target-on-time",
				LastSeen:    pod.CreationTimestamp.Add(time.Minute),
			},
			{
				Destination: "target-before-time",
				LastSeen:    pod.CreationTimestamp.Add(-time.Minute),
			},
		},
	}
	srcIdentity, err := s.resolver.discoverInternalSrcIdentity(context.Background(), recorededDestinationForSrc)
	s.Require().NoError(err)
	s.Require().Equal("pod3", srcIdentity.Name)

	s.Require().Len(recorededDestinationForSrc.Destinations, 1)
	s.Require().Equal("target-on-time", recorededDestinationForSrc.Destinations[0].Destination)

}

func (s *ResolverTestSuite) TestDiscoverInternalSrcIdentityIgnoreControlPlaneIfBackedByHostNetworkPod() {

	// get control plane service
	controlPlaneService := &v1.Service{}
	err := s.Mgr.GetClient().Get(context.Background(), types.NamespacedName{Name: "kubernetes", Namespace: "default"}, controlPlaneService)
	s.Require().NoError(err)

	// get endpoints for control plane service
	endpointsTest := &v1.Endpoints{}
	err = s.Mgr.GetClient().Get(context.Background(), types.NamespacedName{Name: "kubernetes", Namespace: "default"}, endpointsTest)
	s.Require().NoError(err)

	// Add host network pod with the IP of the first endpoint
	pod := s.AddPodWithHostNetwork("pod", endpointsTest.Subsets[0].Addresses[0].IP, map[string]string{"app": "test"}, nil, true)

	// Test source ip is pod ip
	identity, err := s.resolver.discoverInternalSrcIdentity(context.Background(),
		&model.RecordedDestinationsForSrc{
			SrcIP: pod.Status.PodIP,
			Destinations: []model.Destination{
				{
					Destination: "8.8.8.8",
				},
			},
		})
	s.Require().Equal(SourceIsHostNetworkPodError, err)
	s.Require().Empty(identity)
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(ResolverTestSuite))
}
