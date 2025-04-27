package clouduploader

import (
	"context"
	"errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/awsintentsholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/incomingtrafficholder"
	"testing"
	"time"

	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	testTimestamp = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
)

type CloudUploaderTestSuite struct {
	suite.Suite
	testNamespace    string
	intentsHolder    *intentsstore.IntentsHolder
	awsIntentsHolder *awsintentsholder.AWSIntentsHolder
	incomingHolder   *incomingtrafficholder.IncomingTrafficIntentsHolder
	cloudUploader    *CloudUploader
	clientMock       *cloudclientmocks.MockCloudClient
}

func (s *CloudUploaderTestSuite) SetupTest() {
	s.testNamespace = "test-namespace"
	s.intentsHolder = intentsstore.NewIntentsHolder()
	s.incomingHolder = incomingtrafficholder.NewIncomingTrafficIntentsHolder()
	s.awsIntentsHolder = awsintentsholder.New()
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
			Client:         &model.OtterizeServiceIdentity{Name: source, Namespace: srcNamespace},
			Server:         &model.OtterizeServiceIdentity{Name: destination, Namespace: dstNamespace, ResolutionData: lo.ToPtr(model.IdentityResolutionData{})},
			ResolutionData: lo.ToPtr("handleInternalTrafficTCPResult"),
		},
		[]int64{int64(20205)},
	)
}

func (s *CloudUploaderTestSuite) addAwsIntent(source string, srcNamespace string, actions []string, arn string, role string) {
	s.awsIntentsHolder.AddIntent(
		awsintentsholder.AWSIntent{
			Client:  model.OtterizeServiceIdentity{Name: source, Namespace: srcNamespace},
			Actions: actions,
			ARN:     arn,
			IamRole: role,
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
		ClientName:       nilIfEmpty(clientName),
		ServerName:       nilIfEmpty(serverName),
		Namespace:        nilIfEmpty(namespace),
		ServerNamespace:  nilIfEmpty(serverNamespace),
		Topics:           []*cloudclient.KafkaConfigInput{},
		Resources:        []*cloudclient.HTTPConfigInput{},
		ResolutionData:   lo.ToPtr("handleInternalTrafficTCPResult"),
		ConnectionsCount: lo.ToPtr(cloudclient.ConnectionsCount{Current: lo.ToPtr(1), Added: lo.ToPtr(1), Removed: lo.ToPtr(0)}),
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

	s.cloudUploader.NotifyIntents(context.Background(), s.intentsHolder.GetNewIntentsSinceLastGet())

	s.addIntent("client2", s.testNamespace, "server1", s.testNamespace)

	intents2 := []cloudclient.IntentInput{
		intentInput("client2", s.testNamespace, "server1", s.testNamespace),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents2)).Return(nil).Times(1)

	s.cloudUploader.NotifyIntents(context.Background(), s.intentsHolder.GetNewIntentsSinceLastGet())
}

func (s *CloudUploaderTestSuite) TestUploadIncomingTrafficIntents() {
	sourceIP := "34.13.0.22"
	incomingIntent := incomingtrafficholder.IncomingTrafficIntent{
		Server:   model.OtterizeServiceIdentity{Name: "server1", Namespace: s.testNamespace},
		LastSeen: testTimestamp,
		IP:       sourceIP,
	}
	s.incomingHolder.AddIntent(incomingIntent)

	intents1 := []cloudclient.IncomingTrafficDiscoveredIntentInput{
		{
			DiscoveredAt: testTimestamp,
			Intent: cloudclient.IncomingTrafficIntentInput{
				ServerName: "server1",
				Namespace:  s.testNamespace,
				Source: cloudclient.IncomingInternetSourceInput{
					Ip: sourceIP,
				},
			},
		},
	}

	s.clientMock.EXPECT().ReportIncomingTrafficDiscoveredIntents(gomock.Any(), intents1).Return(nil).Times(1)

	s.cloudUploader.NotifyIncomingTrafficIntents(context.Background(), s.incomingHolder.GetNewIntentsSinceLastGet())

	// Expect no new intents to be uploaded
	s.cloudUploader.NotifyIncomingTrafficIntents(context.Background(), s.incomingHolder.GetNewIntentsSinceLastGet())
}

func (s *CloudUploaderTestSuite) TestUploadIntentsWithOperations() {
	discoveredProduce := model.Intent{
		Client: &model.OtterizeServiceIdentity{Name: "client1", Namespace: s.testNamespace},
		Server: &model.OtterizeServiceIdentity{Name: "server1", Namespace: s.testNamespace, ResolutionData: lo.ToPtr(model.IdentityResolutionData{})},
		Type:   lo.ToPtr(model.IntentTypeKafka),
		KafkaTopics: []model.KafkaConfig{
			{
				Name:       "my-topic",
				Operations: []model.KafkaOperation{model.KafkaOperationProduce},
			},
		},
	}

	s.intentsHolder.AddIntent(testTimestamp, discoveredProduce, make([]int64, 0))

	discoveredConsume := model.Intent{
		Client: &model.OtterizeServiceIdentity{Name: "client1", Namespace: s.testNamespace},
		Server: &model.OtterizeServiceIdentity{Name: "server1", Namespace: s.testNamespace, ResolutionData: lo.ToPtr(model.IdentityResolutionData{})},
		Type:   lo.ToPtr(model.IntentTypeKafka),
		KafkaTopics: []model.KafkaConfig{
			{
				Name:       "my-topic",
				Operations: []model.KafkaOperation{model.KafkaOperationConsume},
			},
		},
		HTTPResources: []model.HTTPResource{},
	}

	s.intentsHolder.AddIntent(testTimestamp, discoveredConsume, make([]int64, 0))
	cloudIntent := []cloudclient.IntentInput{
		{
			ClientName:      lo.ToPtr("client1"),
			ServerName:      lo.ToPtr("server1"),
			Namespace:       lo.ToPtr(s.testNamespace),
			ServerNamespace: lo.ToPtr(s.testNamespace),
			Type:            lo.ToPtr(cloudclient.IntentTypeKafka),
			Topics: []*cloudclient.KafkaConfigInput{
				{
					Name: lo.ToPtr("my-topic"),
					Operations: []*cloudclient.KafkaOperation{
						lo.ToPtr(cloudclient.KafkaOperationConsume),
						lo.ToPtr(cloudclient.KafkaOperationProduce),
					},
				},
			},
			Resources: []*cloudclient.HTTPConfigInput{},
		},
	}
	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(cloudIntent)).Return(nil).Times(1)

	s.cloudUploader.NotifyIntents(context.Background(), s.intentsHolder.GetNewIntentsSinceLastGet())

	newTimestamp := testTimestamp.Add(time.Hour)
	s.intentsHolder.AddIntent(newTimestamp, discoveredProduce, make([]int64, 0))

	produceOnly := []cloudclient.IntentInput{
		{
			ClientName:      lo.ToPtr("client1"),
			ServerName:      lo.ToPtr("server1"),
			Namespace:       lo.ToPtr(s.testNamespace),
			ServerNamespace: lo.ToPtr(s.testNamespace),
			Type:            lo.ToPtr(cloudclient.IntentTypeKafka),
			Topics: []*cloudclient.KafkaConfigInput{
				{
					Name: lo.ToPtr("my-topic"),
					Operations: []*cloudclient.KafkaOperation{
						lo.ToPtr(cloudclient.KafkaOperationProduce),
					},
				},
			},
			Resources: []*cloudclient.HTTPConfigInput{},
		},
	}
	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(produceOnly)).Return(nil).Times(1)

	s.cloudUploader.NotifyIntents(context.Background(), s.intentsHolder.GetNewIntentsSinceLastGet())
}

func (s *CloudUploaderTestSuite) TestUploadIntentsInBatches() {
	s.cloudUploader.config.UploadBatchSize = 1
	s.addIntent("client1", s.testNamespace, "server1", s.testNamespace)
	s.addIntent("client1", s.testNamespace, "server2", "external-namespace")

	intents1 := []cloudclient.IntentInput{
		intentInput("client1", s.testNamespace, "server1", s.testNamespace),
		intentInput("client1", s.testNamespace, "server2", "external-namespace"),
	}

	// This can happen in any order, but either way only one intent should be uploaded at a batch
	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher([]cloudclient.IntentInput{intents1[0]})).Return(nil).Times(1)
	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher([]cloudclient.IntentInput{intents1[1]})).Return(nil).Times(1)

	s.cloudUploader.NotifyIntents(context.Background(), s.intentsHolder.GetNewIntentsSinceLastGet())
}

func (s *CloudUploaderTestSuite) TestDontUploadWithoutIntents() {
	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), gomock.Any()).Times(0)

	s.cloudUploader.NotifyIntents(context.Background(), s.intentsHolder.GetNewIntentsSinceLastGet())
}

func (s *CloudUploaderTestSuite) TestUploadSameIntentOnce() {
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)

	intents := []cloudclient.IntentInput{
		intentInput("client", s.testNamespace, "server", s.testNamespace),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents)).Return(nil).Times(1)

	s.cloudUploader.NotifyIntents(context.Background(), s.intentsHolder.GetNewIntentsSinceLastGet())

	// We will upload the same intent twice, so we expect the add count to be 0
	intents[0].ConnectionsCount.Added = lo.ToPtr(0)
	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents)).Return(nil).Times(1)
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)
	s.cloudUploader.NotifyIntents(context.Background(), s.intentsHolder.GetNewIntentsSinceLastGet())
}

func (s *CloudUploaderTestSuite) TestRetryOnFailed() {
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)

	intents := []cloudclient.IntentInput{
		intentInput("client", s.testNamespace, "server", s.testNamespace),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents)).Return(errors.New("fail")).Times(1)

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents)).Return(nil).Times(1)

	s.cloudUploader.NotifyIntents(context.Background(), s.intentsHolder.GetNewIntentsSinceLastGet())
}

func (s *CloudUploaderTestSuite) TestDontUploadWhenNothingNew() {
	s.addIntent("client", s.testNamespace, "server", s.testNamespace)

	intents := []cloudclient.IntentInput{
		intentInput("client", s.testNamespace, "server", s.testNamespace),
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents)).Return(nil).Times(1)

	s.cloudUploader.NotifyIntents(context.Background(), s.intentsHolder.GetNewIntentsSinceLastGet())
	s.cloudUploader.NotifyIntents(context.Background(), s.intentsHolder.GetNewIntentsSinceLastGet())
}

func (s *CloudUploaderTestSuite) TestReportMapperComponent() {
	s.clientMock.EXPECT().ReportComponentStatus(gomock.Any(), cloudclient.ComponentTypeNetworkMapper).Times(1)

	s.cloudUploader.reportStatus(context.Background())
}

func (s *CloudUploaderTestSuite) TestNotifyAWSIntents() {
	clientName := "client1"
	resoureArn := "arn:aws:s3:::bucket"
	awsRole := "role"
	awsActions := []string{"s3:GetObject"}
	intentType := cloudclient.IntentTypeAws
	s.addAwsIntent(clientName, s.testNamespace, awsActions, resoureArn, awsRole)

	intents1 := []cloudclient.IntentInput{
		{
			ClientName: lo.ToPtr(clientName),
			ServerName: lo.ToPtr(resoureArn),
			Namespace:  lo.ToPtr(s.testNamespace),
			Type:       lo.ToPtr(intentType),
			AwsRole:    lo.ToPtr(awsRole),
			AwsActions: []*string{lo.ToPtr("s3:GetObject")},
		},
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents1)).Return(nil).Times(1)

	s.cloudUploader.NotifyAWSIntents(context.Background(), s.awsIntentsHolder.GetNewIntentsSinceLastGet())

	// Test with empty role - should send nil

	s.addAwsIntent(clientName, s.testNamespace, awsActions, resoureArn, "")

	intents2 := []cloudclient.IntentInput{
		{
			ClientName: lo.ToPtr(clientName),
			ServerName: lo.ToPtr(resoureArn),
			Namespace:  lo.ToPtr(s.testNamespace),
			Type:       lo.ToPtr(intentType),
			AwsRole:    nil,
			AwsActions: []*string{lo.ToPtr("s3:GetObject")},
		},
	}

	s.clientMock.EXPECT().ReportDiscoveredIntents(gomock.Any(), GetMatcher(intents2)).Return(nil).Times(1)

	s.cloudUploader.NotifyAWSIntents(context.Background(), s.awsIntentsHolder.GetNewIntentsSinceLastGet())
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(CloudUploaderTestSuite))
}
