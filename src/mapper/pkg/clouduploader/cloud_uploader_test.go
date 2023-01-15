package clouduploader

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/resolvers"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
	"testing"
	"time"
)

var (
	testTimestamp = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
)

type CloudUploaderTestSuite struct {
	suite.Suite
	testNamespace string
	intentsHolder *resolvers.IntentsHolder
	cloudUploader *CloudUploader
	cloudConfig   Config
	clientMock    *cloudclientmocks.MockCloudClient
}

func (s *CloudUploaderTestSuite) SetupTest() {
	s.testNamespace = "test-namespace"
	s.intentsHolder = resolvers.NewIntentsHolder(nil, resolvers.IntentsHolderConfig{StoreConfigMap: config.StoreConfigMapDefault, Namespace: s.testNamespace})
	s.cloudConfig = Config{
		ClientId: "test-client-id",
	}
}

func (s *CloudUploaderTestSuite) BeforeTest(_, testName string) {
	controller := gomock.NewController(s.T())
	factory := s.GetCloudClientFactoryMock(controller)
	s.cloudUploader = NewCloudUploader(s.intentsHolder, s.cloudConfig, factory)
}

func (s *CloudUploaderTestSuite) GetCloudClientFactoryMock(controller *gomock.Controller) cloudclient.FactoryFunction {
	s.clientMock = cloudclientmocks.NewMockCloudClient(controller)

	factory := func(ctx context.Context, apiAddress string, tokenSource oauth2.TokenSource) cloudclient.CloudClient {
		return s.clientMock
	}
	return factory
}

func (s *CloudUploaderTestSuite) addIntent(source string, srcNamespace string, destination string, dstNamespace string) {
	s.intentsHolder.AddIntent(
		model.OtterizeServiceIdentity{Name: source, Namespace: srcNamespace},
		model.OtterizeServiceIdentity{Name: destination, Namespace: dstNamespace},
		testTimestamp,
	)
}

func intentInput(clientName string, namespace string, serverName string, serverNamespace string) cloudclient.IntentInput {
	nilIfEmpty := func(s string) *string {
		if s == "" {
			return nil
		}
		return lo.ToPtr(s)
	}

	return cloudclient.IntentInput{
		ClientName:      nilIfEmpty(clientName),
		ServerName:      nilIfEmpty(serverName),
		Namespace:       nilIfEmpty(namespace),
		ServerNamespace: nilIfEmpty(serverNamespace),
	}
}

func (s *CloudUploaderTestSuite) TestUploadIntents() {
	s.addIntent("client1", s.testNamespace, "server1", s.testNamespace)
	s.addIntent("client1", s.testNamespace, "server2", "external-namespace")

	intents1 := []cloudclient.IntentInput{
		intentInput("client1", s.testNamespace, "server1", s.testNamespace),
		intentInput("client1", s.testNamespace, "server2", "external-namespace"),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(GetMatcher(intents1)).Return(true).Times(1)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())

	s.addIntent("client2", s.testNamespace, "server1", s.testNamespace)

	intents2 := []cloudclient.IntentInput{
		intentInput("client2", s.testNamespace, "server1", s.testNamespace),
		intentInput("client1", s.testNamespace, "server1", s.testNamespace),
		intentInput("client1", s.testNamespace, "server2", "external-namespace"),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(GetMatcher(intents2)).Return(true).Times(1)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())
}

func (s *CloudUploaderTestSuite) TestDontUploadWithoutIntents() {
	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any()).Times(0)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())
}

func (s *CloudUploaderTestSuite) TestUploadSameIntentOnce() {
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)

	intents := []cloudclient.IntentInput{
		intentInput("client", s.testNamespace, "server", s.testNamespace),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(GetMatcher(intents)).Return(true).Times(1)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)
	s.cloudUploader.uploadDiscoveredIntents(context.Background())
}

func (s *CloudUploaderTestSuite) TestRetryOnFailed() {
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)

	intents := []cloudclient.IntentInput{
		intentInput("client", s.testNamespace, "server", s.testNamespace),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(GetMatcher(intents)).Return(false).Times(1)

	s.clientMock.EXPECT().ReportDiscoveredIntents(GetMatcher(intents)).Return(true).Times(1)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())
	s.cloudUploader.uploadDiscoveredIntents(context.Background())
}

func (s *CloudUploaderTestSuite) TestDontUploadWhenNothingNew() {
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)

	intents := []cloudclient.IntentInput{
		intentInput("client", s.testNamespace, "server", s.testNamespace),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(GetMatcher(intents)).Return(true).Times(1)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())
	s.cloudUploader.uploadDiscoveredIntents(context.Background())
}

func (s *CloudUploaderTestSuite) TestReportMapperComonent() {
	s.clientMock.EXPECT().ReportComponentStatus(cloudclient.ComponentTypeNetworkMapper).Times(1)

	s.cloudUploader.reportStatus(context.Background())
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(CloudUploaderTestSuite))
}
