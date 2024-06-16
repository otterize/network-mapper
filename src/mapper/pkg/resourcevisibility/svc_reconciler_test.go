package resourcevisibility

import (
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"time"
)

var (
	testTimestamp = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
)

type ResourceVisibilityTestSuite struct {
	suite.Suite
	testNamespace string
	cloudClient   *cloudclientmocks.MockCloudClient
	k8sClient     *mocks.K8sClient
}

func (s *ResourceVisibilityTestSuite) BeforeTest(_, testName string) {
	controller := gomock.NewController(s.T())
	s.cloudClient = cloudclientmocks.NewMockCloudClient(controller)
	s.k8sClient = mocks.NewK8sClient(controller)
}
