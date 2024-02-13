package dnsintentspublisher

import (
	"context"
	otterizev1alpha3 "github.com/otterize/intents-operator/src/operator/api/v1alpha3"
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

	intentsWithDNS1 := otterizev1alpha3.ClientIntents{
		Spec: &otterizev1alpha3.IntentsSpec{
			Service: otterizev1alpha3.Service{
				Name: "blog-reader",
			},
			Calls: []otterizev1alpha3.Intent{
				{
					Type: otterizev1alpha3.IntentTypeInternet,
					Internet: &otterizev1alpha3.Internet{
						Dns: "my-blog.de",
					},
				},
				{
					Name: "server",
				},
			},
		},
	}

	intentsWithDNS2 := otterizev1alpha3.ClientIntents{
		Spec: &otterizev1alpha3.IntentsSpec{
			Service: otterizev1alpha3.Service{
				Name: "time-waster",
			},
			Calls: []otterizev1alpha3.Intent{
				{
					Type: otterizev1alpha3.IntentTypeInternet,
					Internet: &otterizev1alpha3.Internet{
						Dns: "ottersgram.com",
					},
				},
			},
		},
	}

	justOtherIntents := otterizev1alpha3.ClientIntents{
		Spec: &otterizev1alpha3.IntentsSpec{
			Service: otterizev1alpha3.Service{
				Name: "just-client",
			},
			Calls: []otterizev1alpha3.Intent{
				{
					Name: "server",
				},
			},
		},
	}

	var intentsList otterizev1alpha3.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev1alpha3.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev1alpha3.ClientIntents{intentsWithDNS1, intentsWithDNS2, justOtherIntents}
			return nil
		})

	s.k8sMockClient.EXPECT().Status().Return(s.k8sMockStatus).Times(2)

	intents1WithResolvedIPs := intentsWithDNS1.DeepCopy()
	intents1WithResolvedIPs.Status.ResolvedIPs = []otterizev1alpha3.ResolvedIPs{
		{
			DNS: "my-blog.de",
			IPs: []string{IP1},
		},
	}
	intents2WithResolvedIPs := intentsWithDNS2.DeepCopy()
	intents2WithResolvedIPs.Status.ResolvedIPs = []otterizev1alpha3.ResolvedIPs{
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

	var intentsList otterizev1alpha3.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev1alpha3.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev1alpha3.ClientIntents{}
			return nil
		})

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestNoDNSIntents() {
	clientIntentsWithoutDNS := otterizev1alpha3.ClientIntents{
		Spec: &otterizev1alpha3.IntentsSpec{
			Service: otterizev1alpha3.Service{
				Name: "just-client",
			},
			Calls: []otterizev1alpha3.Intent{
				{
					Name: "server",
				},
			},
		},
	}

	var intentsList otterizev1alpha3.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev1alpha3.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev1alpha3.ClientIntents{clientIntentsWithoutDNS}
			return nil
		})

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestIPNeverSeen() {
	ttlInSeconds := ttlForTest()
	s.dnsCache.AddOrUpdateDNSData("what-should-i-eat.com", IP1, ttlInSeconds)

	intentsWithoutResolvedIPs := otterizev1alpha3.ClientIntents{
		Spec: &otterizev1alpha3.IntentsSpec{
			Service: otterizev1alpha3.Service{
				Name: "just-client",
			},
			Calls: []otterizev1alpha3.Intent{
				{
					Type: otterizev1alpha3.IntentTypeInternet,
					Internet: &otterizev1alpha3.Internet{
						Dns: "how-to-cook-kube.com",
					},
				},
			},
		},
	}

	var intentsList otterizev1alpha3.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev1alpha3.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev1alpha3.ClientIntents{intentsWithoutResolvedIPs}
			return nil
		})

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestNothingToUpdate() {
	ttlInSeconds := ttlForTest()
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)

	intentsWithDNS1 := otterizev1alpha3.ClientIntents{
		Spec: &otterizev1alpha3.IntentsSpec{
			Service: otterizev1alpha3.Service{
				Name: "blog-reader",
			},
			Calls: []otterizev1alpha3.Intent{
				{
					Type: otterizev1alpha3.IntentTypeInternet,
					Internet: &otterizev1alpha3.Internet{
						Dns: "my-blog.de",
					},
				},
			},
		},
		Status: otterizev1alpha3.IntentsStatus{
			ResolvedIPs: []otterizev1alpha3.ResolvedIPs{
				{
					DNS: "my-blog.de",
					IPs: []string{IP1},
				},
			},
		},
	}

	var intentsList otterizev1alpha3.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev1alpha3.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev1alpha3.ClientIntents{intentsWithDNS1}
			return nil
		})

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestDoNothingWithIntentsWithoutDNS() {
	ttlInSeconds := ttlForTest()
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)

	intentsWithDNS1 := otterizev1alpha3.ClientIntents{
		Spec: &otterizev1alpha3.IntentsSpec{
			Service: otterizev1alpha3.Service{
				Name: "blog-reader",
			},
			Calls: []otterizev1alpha3.Intent{
				{
					Type: otterizev1alpha3.IntentTypeInternet,
					Internet: &otterizev1alpha3.Internet{
						Ips: []string{IP1},
					},
				},
			},
		},
	}

	var intentsList otterizev1alpha3.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev1alpha3.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev1alpha3.ClientIntents{intentsWithDNS1}
			return nil
		})

	err := s.publisher.PublishDNSIntents(context.Background())
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestAppendToOldIP() {
	ttlInSeconds := ttlForTest()
	oldIP := "10.0.2.10"
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)

	intentsWithDNS1 := otterizev1alpha3.ClientIntents{
		Spec: &otterizev1alpha3.IntentsSpec{
			Service: otterizev1alpha3.Service{
				Name: "blog-reader",
			},
			Calls: []otterizev1alpha3.Intent{
				{
					Type: otterizev1alpha3.IntentTypeInternet,
					Internet: &otterizev1alpha3.Internet{
						Dns: "my-blog.de",
					},
				},
			},
		},
		Status: otterizev1alpha3.IntentsStatus{
			ResolvedIPs: []otterizev1alpha3.ResolvedIPs{
				{
					DNS: "my-blog.de",
					IPs: []string{oldIP},
				},
			},
		},
	}

	var intentsList otterizev1alpha3.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev1alpha3.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev1alpha3.ClientIntents{intentsWithDNS1}
			return nil
		})

	s.k8sMockClient.EXPECT().Status().Return(s.k8sMockStatus).Times(1)

	intents1WithResolvedIPs := intentsWithDNS1.DeepCopy()
	intents1WithResolvedIPs.Status.ResolvedIPs = []otterizev1alpha3.ResolvedIPs{
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

func (s *PublisherTestSuite) TestShouldNotAffectOtherStatusFields() {
	ttlInSeconds := ttlForTest()
	s.dnsCache.AddOrUpdateDNSData("my-blog.de", IP1, ttlInSeconds)

	intentsWithDNS1 := otterizev1alpha3.ClientIntents{
		Spec: &otterizev1alpha3.IntentsSpec{
			Service: otterizev1alpha3.Service{
				Name: "blog-reader",
			},
			Calls: []otterizev1alpha3.Intent{
				{
					Type: otterizev1alpha3.IntentTypeInternet,
					Internet: &otterizev1alpha3.Internet{
						Dns: "my-blog.de",
					},
				},
			},
		},
		Status: otterizev1alpha3.IntentsStatus{
			UpToDate:           true,
			ObservedGeneration: 89,
		},
	}

	var intentsList otterizev1alpha3.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev1alpha3.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev1alpha3.ClientIntents{intentsWithDNS1}
			return nil
		})

	s.k8sMockClient.EXPECT().Status().Return(s.k8sMockStatus).Times(1)

	intents1WithResolvedIPs := intentsWithDNS1.DeepCopy()
	intents1WithResolvedIPs.Status = otterizev1alpha3.IntentsStatus{
		UpToDate:           true,
		ObservedGeneration: 89,
		ResolvedIPs: []otterizev1alpha3.ResolvedIPs{
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

	intentsWithDNS1 := otterizev1alpha3.ClientIntents{
		Spec: &otterizev1alpha3.IntentsSpec{
			Service: otterizev1alpha3.Service{
				Name: "blog-reader",
			},
			Calls: []otterizev1alpha3.Intent{
				{
					Type: otterizev1alpha3.IntentTypeInternet,
					Internet: &otterizev1alpha3.Internet{
						Dns: "my-blog.de",
					},
				},
			},
		},
	}

	var intentsList otterizev1alpha3.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev1alpha3.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev1alpha3.ClientIntents{intentsWithDNS1}
			return nil
		})

	s.k8sMockClient.EXPECT().Status().Return(s.k8sMockStatus).Times(1)

	intents1WithResolvedIPs := intentsWithDNS1.DeepCopy()
	intents1WithResolvedIPs.Status.ResolvedIPs = []otterizev1alpha3.ResolvedIPs{
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

	intentsWithDNS1 := otterizev1alpha3.ClientIntents{
		Spec: &otterizev1alpha3.IntentsSpec{
			Service: otterizev1alpha3.Service{
				Name: "blog-reader",
			},
			Calls: []otterizev1alpha3.Intent{
				{
					Type: otterizev1alpha3.IntentTypeInternet,
					Internet: &otterizev1alpha3.Internet{
						Dns: "my-blog.de",
					},
				},
				{
					Name: "server",
				},
			},
		},
		Status: otterizev1alpha3.IntentsStatus{
			ResolvedIPs: []otterizev1alpha3.ResolvedIPs{
				{
					DNS: "my-blog.de",
					IPs: []string{
						IP1,
					},
				},
			},
		},
	}

	var intentsList otterizev1alpha3.ClientIntentsList
	s.k8sMockClient.EXPECT().List(gomock.Any(), &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue}).DoAndReturn(
		func(ctx context.Context, list *otterizev1alpha3.ClientIntentsList, opts ...client.ListOption) error {
			list.Items = []otterizev1alpha3.ClientIntents{intentsWithDNS1}
			return nil
		})

	intents1WithResolvedIPs := intentsWithDNS1.DeepCopy()
	intents1WithResolvedIPs.Status.ResolvedIPs = []otterizev1alpha3.ResolvedIPs{
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
