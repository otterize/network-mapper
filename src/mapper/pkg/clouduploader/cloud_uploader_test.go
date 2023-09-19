package clouduploader

import (
	"context"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

var (
	testTimestamp = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
)

type CloudUploaderTestSuite struct {
	suite.Suite
	testNamespace string
	intentsHolder *intentsstore.IntentsHolder
	cloudUploader *CloudUploader
	clientMock    *cloudclientmocks.MockCloudClient
}

func (s *CloudUploaderTestSuite) SetupTest() {
	s.testNamespace = "test-namespace"
	s.intentsHolder = intentsstore.NewIntentsHolder()
}

func (s *CloudUploaderTestSuite) BeforeTest(_, testName string) {
	controller := gomock.NewController(s.T())
	s.clientMock = cloudclientmocks.NewMockCloudClient(controller)
	s.cloudUploader = NewCloudUploader(s.intentsHolder, Config{UploadBatchSize: config.UploadBatchSizeDefault}, s.clientMock)
}

func (s *CloudUploaderTestSuite) addIntent(source string, srcNamespace string, destination string, dstNamespace string) {
	s.intentsHolder.AddIntent(
		testTimestamp,
		model.Intent{
			Client: &model.OtterizeServiceIdentity{Name: source, Namespace: srcNamespace},
			Server: &model.OtterizeServiceIdentity{Name: destination, Namespace: dstNamespace},
		},
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

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents1)).Return(nil).Times(1)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())

	s.addIntent("client2", s.testNamespace, "server1", s.testNamespace)

	intents2 := []cloudclient.IntentInput{
		intentInput("client2", s.testNamespace, "server1", s.testNamespace),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents2)).Return(nil).Times(1)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())
}

func (s *CloudUploaderTestSuite) TestUploadIntentsInBatches() {
	s.cloudUploader.config.UploadBatchSize = 1
	s.addIntent("client1", s.testNamespace, "server1", s.testNamespace)
	s.addIntent("client1", s.testNamespace, "server2", "external-namespace")

	intents1 := []cloudclient.IntentInput{
		intentInput("client1", s.testNamespace, "server1", s.testNamespace),
		intentInput("client1", s.testNamespace, "server2", "external-namespace"),
	}

	firstReport := s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher([]cloudclient.IntentInput{intents1[0]})).Return(nil).Times(1)
	secondReport := s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher([]cloudclient.IntentInput{intents1[1]})).Return(nil).Times(1)
	gomock.InOrder(
		firstReport,
		secondReport,
	)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())

}

func (s *CloudUploaderTestSuite) TestDontUploadWithoutIntents() {
	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), gomock.Any()).Times(0)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())
}

func (s *CloudUploaderTestSuite) TestUploadSameIntentOnce() {
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)

	intents := []cloudclient.IntentInput{
		intentInput("client", s.testNamespace, "server", s.testNamespace),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents)).Return(nil).Times(1)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents)).Return(nil).Times(1)
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)
	s.cloudUploader.uploadDiscoveredIntents(context.Background())
}

func (s *CloudUploaderTestSuite) TestRetryOnFailed() {
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)

	intents := []cloudclient.IntentInput{
		intentInput("client", s.testNamespace, "server", s.testNamespace),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents)).Return(errors.New("fail")).Times(1)

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents)).Return(nil).Times(1)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())
}

func (s *CloudUploaderTestSuite) TestDontUploadWhenNothingNew() {
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)

	intents := []cloudclient.IntentInput{
		intentInput("client", s.testNamespace, "server", s.testNamespace),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents)).Return(nil).Times(1)

	s.cloudUploader.uploadDiscoveredIntents(context.Background())
	s.cloudUploader.uploadDiscoveredIntents(context.Background())
}

func (s *CloudUploaderTestSuite) TestReportMapperComponent() {
	s.clientMock.EXPECT().ReportComponentStatus(gomock.Any(), cloudclient.ComponentTypeNetworkMapper).Times(1)

	s.cloudUploader.reportStatus(context.Background())
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(CloudUploaderTestSuite))
}
