package dnsintentspublisher

import (
	"context"
	otterizev2alpha1 "github.com/otterize/intents-operator/src/operator/api/v2alpha1"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnscache"
	"github.com/otterize/network-mapper/src/mapper/pkg/mocks"
	"github.com/otterize/network-mapper/src/shared/testbase"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

const (
	IP1 = "10.0.0.1"
	IP2 = "10.0.0.2"
	IP3 = "10.0.0.3"
)

type PublisherTestSuite struct {
	suite.Suite
	k8sMockClient *mocks.K8sClient
	k8sMockStatus *mocks.K8sStatus
	dnsCache      *dnscache.DNSCache
	publisher     *Publisher
}

func (s *PublisherTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.k8sMockClient = mocks.NewK8sClient(controller)
	s.k8sMockStatus = mocks.NewK8sStatus(controller)
	s.dnsCache = dnscache.NewDNSCache()
	s.publisher = NewPublisher(s.k8sMockClient, s.dnsCache)
	s.Require().NotNil(s.publisher)
}

func (s *PublisherTestSuite) TestPublisher() {
	ttlInSeconds := ttlForTest()
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)
	s.dnsCache.AddOrUpdateDNSData("ottersgram.com", IP2, ttlInSeconds)

	intentsWithDNS1 := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "blog-reader",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Internet: &otterizev2alpha1.Internet{
						Domains: []string{"my-blog.de"},
					},
				},
				{
					Kubernetes: &otterizev2alpha1.KubernetesTarget{Name: "server"},
				},
			},
		},
	}

	intentsWithDNS2 := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "time-waster",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Internet: &otterizev2alpha1.Internet{
						Domains: []string{"ottersgram.com"},
					},
				},
			},
		},
	}

	justOtherIntents := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "just-client",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Kubernetes: &otterizev2alpha1.KubernetesTarget{Name: "server"},
				},
			},
		},
	}

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{intentsWithDNS1, intentsWithDNS2, justOtherIntents}
			return nil
		})

	s.k8sMockClient.EXPECT().Status().Return(s.k8sMockStatus).Times(2)

	intents1WithResolvedIPs := intentsWithDNS1.DeepCopy()
	intents1WithResolvedIPs.Status.ResolvedIPs = []otterizev2alpha1.ResolvedIPs{
		{
			DNS: "my-blog.de",
			IPs: []string{IP1},
		},
	}
	intents2WithResolvedIPs := intentsWithDNS2.DeepCopy()
	intents2WithResolvedIPs.Status.ResolvedIPs = []otterizev2alpha1.ResolvedIPs{
		{
			DNS: "ottersgram.com",
			IPs: []string{IP2},
		},
	}
	s.k8sMockStatus.EXPECT().Patch(gomock.Any(), intents1WithResolvedIPs, testbase.MatchPatch(client.MergeFrom(&intentsWithDNS1))).Return(nil)
	s.k8sMockStatus.EXPECT().Patch(gomock.Any(), intents2WithResolvedIPs, testbase.MatchPatch(client.MergeFrom(&intentsWithDNS2))).Return(nil)

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestNoIntents() {
	ttlInSeconds := ttlForTest()
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)
	s.dnsCache.AddOrUpdateDNSData("ottersbook.com", IP2, ttlInSeconds)

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{}
			return nil
		})

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestNoDNSIntents() {
	clientIntentsWithoutDNS := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "just-client",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Kubernetes: &otterizev2alpha1.KubernetesTarget{Name: "server"},
				},
			},
		},
	}

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{clientIntentsWithoutDNS}
			return nil
		})

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestIPNeverSeen() {
	ttlInSeconds := ttlForTest()
	s.dnsCache.AddOrUpdateDNSData("what-should-i-eat.com", IP1, ttlInSeconds)

	intentsWithoutResolvedIPs := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "just-client",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Internet: &otterizev2alpha1.Internet{
						Domains: []string{"how-to-cook-kube.com"},
					},
				},
			},
		},
	}

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{intentsWithoutResolvedIPs}
			return nil
		})

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestNothingToUpdate() {
	ttlInSeconds := ttlForTest()
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)

	intentsWithDNS1 := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "blog-reader",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Internet: &otterizev2alpha1.Internet{
						Domains: []string{"my-blog.de"},
					},
				},
			},
		},
		Status: otterizev2alpha1.IntentsStatus{
			ResolvedIPs: []otterizev2alpha1.ResolvedIPs{
				{
					DNS: "my-blog.de",
					IPs: []string{IP1},
				},
			},
		},
	}

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{intentsWithDNS1}
			return nil
		})

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestDoNothingWithIntentsWithoutDNS() {
	ttlInSeconds := ttlForTest()
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)

	intentsWithDNS1 := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "blog-reader",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Internet: &otterizev2alpha1.Internet{
						Ips: []string{IP1},
					},
				},
			},
		},
	}

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{intentsWithDNS1}
			return nil
		})

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestAppendToOldIP() {
	ttlInSeconds := ttlForTest()
	oldIP := "10.0.2.10"
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)

	intentsWithDNS1 := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "blog-reader",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Internet: &otterizev2alpha1.Internet{
						Domains: []string{"my-blog.de"},
					},
				},
			},
		},
		Status: otterizev2alpha1.IntentsStatus{
			ResolvedIPs: []otterizev2alpha1.ResolvedIPs{
				{
					DNS: "my-blog.de",
					IPs: []string{oldIP},
				},
			},
		},
	}

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{intentsWithDNS1}
			return nil
		})

	s.k8sMockClient.EXPECT().Status().Return(s.k8sMockStatus).Times(1)

	intents1WithResolvedIPs := intentsWithDNS1.DeepCopy()
	intents1WithResolvedIPs.Status.ResolvedIPs = []otterizev2alpha1.ResolvedIPs{
		{
			DNS: "my-blog.de",
			IPs: []string{
				oldIP,
				IP1,
			},
		},
	}

	s.k8sMockStatus.EXPECT().Patch(gomock.Any(), intents1WithResolvedIPs, testbase.MatchPatch(client.MergeFrom(&intentsWithDNS1))).Return(nil)

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestRemoveOldDNSFromStatus_OtherDNSExists() {
	ttlInSeconds := ttlForTest()
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)

	intentsWithDNS := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "blog-reader",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Internet: &otterizev2alpha1.Internet{
						Domains: []string{"my-blog.de"},
					},
				},
			},
		},
		Status: otterizev2alpha1.IntentsStatus{
			ResolvedIPs: []otterizev2alpha1.ResolvedIPs{
				{
					DNS: "my-blog.de",
					IPs: []string{IP1},
				},
				{
					DNS: "otternet.com",
					IPs: []string{IP2},
				},
			},
		},
	}

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{intentsWithDNS}
			return nil
		})

	s.k8sMockClient.EXPECT().Status().Return(s.k8sMockStatus).Times(1)

	intents1WithResolvedIPs := intentsWithDNS.DeepCopy()
	intents1WithResolvedIPs.Status.ResolvedIPs = []otterizev2alpha1.ResolvedIPs{
		{
			DNS: "my-blog.de",
			IPs: []string{IP1},
		},
	}

	s.k8sMockStatus.EXPECT().Patch(gomock.Any(), intents1WithResolvedIPs, testbase.MatchPatch(client.MergeFrom(&intentsWithDNS))).Return(nil)

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestRemoveOldDNSFromStatus_DeleteAll() {
	intentsWithDNS := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "blog-reader",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Internet: &otterizev2alpha1.Internet{
						Ips: []string{IP1},
					},
				},
			},
		},
		Status: otterizev2alpha1.IntentsStatus{
			ResolvedIPs: []otterizev2alpha1.ResolvedIPs{
				{
					DNS: "my-blog.de",
					IPs: []string{IP1},
				},
				{
					DNS: "otternet.com",
					IPs: []string{IP2},
				},
			},
		},
	}

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{intentsWithDNS}
			return nil
		})

	s.k8sMockClient.EXPECT().Status().Return(s.k8sMockStatus).Times(1)

	intents1WithResolvedIPs := intentsWithDNS.DeepCopy()
	intents1WithResolvedIPs.Status.ResolvedIPs = []otterizev2alpha1.ResolvedIPs{}

	s.k8sMockStatus.EXPECT().Patch(gomock.Any(), intents1WithResolvedIPs, testbase.MatchPatch(client.MergeFrom(&intentsWithDNS))).Return(nil)

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestShouldNotAffectOtherStatusFields() {
	ttlInSeconds := ttlForTest()
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)

	intentsWithDNS1 := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "blog-reader",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Internet: &otterizev2alpha1.Internet{
						Domains: []string{"my-blog.de"},
					},
				},
			},
		},
		Status: otterizev2alpha1.IntentsStatus{
			UpToDate:           true,
			ObservedGeneration: 89,
		},
	}

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{intentsWithDNS1}
			return nil
		})

	s.k8sMockClient.EXPECT().Status().Return(s.k8sMockStatus).Times(1)

	intents1WithResolvedIPs := intentsWithDNS1.DeepCopy()
	intents1WithResolvedIPs.Status = otterizev2alpha1.IntentsStatus{
		UpToDate:           true,
		ObservedGeneration: 89,
		ResolvedIPs: []otterizev2alpha1.ResolvedIPs{
			{
				DNS: "my-blog.de",
				IPs: []string{
					IP1,
				},
			},
		},
	}

	s.k8sMockStatus.EXPECT().Patch(gomock.Any(), intents1WithResolvedIPs, testbase.MatchPatch(client.MergeFrom(&intentsWithDNS1))).Return(nil)

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestOnlyLatestIP() {
	ttlInSeconds := ttlForTest()
	oldIP := "10.0.2.10"
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", oldIP, ttlInSeconds)
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)

	intentsWithDNS1 := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "blog-reader",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Internet: &otterizev2alpha1.Internet{
						Domains: []string{"my-blog.de"},
					},
				},
			},
		},
	}

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{intentsWithDNS1}
			return nil
		})

	s.k8sMockClient.EXPECT().Status().Return(s.k8sMockStatus).Times(1)

	intents1WithResolvedIPs := intentsWithDNS1.DeepCopy()
	intents1WithResolvedIPs.Status.ResolvedIPs = []otterizev2alpha1.ResolvedIPs{
		{
			DNS: "my-blog.de",
			IPs: []string{
				IP1,
			},
		},
	}

	s.k8sMockStatus.EXPECT().Patch(gomock.Any(), intents1WithResolvedIPs, testbase.MatchPatch(client.MergeFrom(&intentsWithDNS1))).Return(nil)

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestUpdate() {
	ttlInSeconds := ttlForTest()

	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP2, ttlInSeconds)

	intentsWithDNS1 := otterizev2alpha1.ClientIntents{
		Spec: &otterizev2alpha1.IntentsSpec{
			Workload: otterizev2alpha1.Workload{
				Name: "blog-reader",
			},
			Targets: []otterizev2alpha1.Target{
				{
					Internet: &otterizev2alpha1.Internet{
						Domains: []string{"my-blog.de"},
					},
				},
				{
					Kubernetes: &otterizev2alpha1.KubernetesTarget{Name: "server"},
				},
			},
		},
		Status: otterizev2alpha1.IntentsStatus{
			ResolvedIPs: []otterizev2alpha1.ResolvedIPs{
				{
					DNS: "my-blog.de",
					IPs: []string{
						IP1,
					},
				},
			},
		},
	}

	var intentsList otterizev2alpha1.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev2alpha1.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev2alpha1.ClientIntents{intentsWithDNS1}
			return nil
		})

	intents1WithResolvedIPs := intentsWithDNS1.DeepCopy()
	intents1WithResolvedIPs.Status.ResolvedIPs = []otterizev2alpha1.ResolvedIPs{
		{
			DNS: "my-blog.de",
			IPs: []string{
				IP1,
				IP2,
			},
		},
	}

	s.k8sMockClient.EXPECT().Status().Return(s.k8sMockStatus)
	s.k8sMockStatus.EXPECT().Patch(gomock.Any(), intents1WithResolvedIPs, testbase.MatchPatch(client.MergeFrom(&intentsWithDNS1))).Return(nil)

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func ttlForTest() int {
	return int(time.Hour.Seconds())
}

func TestPublisherTestSuite(t *testing.T) {
	suite.Run(t, new(PublisherTestSuite))
}
