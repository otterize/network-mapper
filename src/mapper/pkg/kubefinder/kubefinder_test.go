package kubefinder

import (
	"context"
	"github.com/otterize/otternose/shared/testutils"
	"github.com/stretchr/testify/suite"
	"testing"
)

type KubeFinderTestSuite struct {
	testutils.ManagerTestSuite
}

func (suite *KubeFinderTestSuite) TestResolveIpToPod() {
	kf, err := NewKubeFinder(suite.Mgr)
	suite.Require().NoError(err)

	pod, err := kf.ResolveIpToPod(context.Background(), "1.1.1.1")
	suite.Require().Nil(pod)
	suite.Require().Error(err)

	suite.AddPod("some-pod", "2.2.2.2")
	suite.AddPod("test-pod", "1.1.1.1")
	suite.AddPod("pod-with-no-ip", "")

	suite.Mgr.GetCache().WaitForCacheSync(context.Background())
	pod, err = kf.ResolveIpToPod(context.Background(), "1.1.1.1")
	suite.Require().NoError(err)
	suite.Require().Equal("test-pod", pod.Name)

}

func TestKubeFinderTestSuite(t *testing.T) {
	suite.Run(t, new(KubeFinderTestSuite))
}
