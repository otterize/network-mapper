package sniffer

import (
	"context"
	"encoding/hex"
	"github.com/golang/mock/gomock"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/otterize/otternose/sniffer/pkg/client"
	mock_client "github.com/otterize/otternose/sniffer/pkg/client/mockclient"
	"github.com/stretchr/testify/suite"
	"testing"
)

type SnifferTestSuite struct {
	suite.Suite
	mockController   *gomock.Controller
	mockMapperClient *mock_client.MockMapperClient
}

func (s *SnifferTestSuite) SetupSuite() {
	s.mockController = gomock.NewController(s.T())
	s.mockMapperClient = mock_client.NewMockMapperClient(s.mockController)
}

func (s *SnifferTestSuite) TestHandlePacket() {
	sniffer := NewSniffer(s.mockMapperClient)
	rawDnsResponse, err := hex.DecodeString("f84d8969309600090f090002080045000059eb6c40004011b325d05b70340a65510d0035fcb40045a621339681800001000100000000037374730975732d656173742d3109616d617a6f6e61777303636f6d0000010001c00c000100010000003c00044815ce60")
	if err != nil {
		s.Require().NoError(err)
	}
	packet := gopacket.NewPacket(rawDnsResponse, layers.LayerTypeEthernet, gopacket.Default)
	sniffer.HandlePacket(packet)

	s.mockMapperClient.EXPECT().ReportCaptureResults(gomock.Any(), client.CaptureResults{
		Results: []client.CaptureResultForSrcIp{
			{
				SrcIp:        "10.101.81.13",
				Destinations: []string{"sts.us-east-1.amazonaws.com"},
			},
		},
	})
	err = sniffer.ReportCaptureResults(context.Background())
	s.Require().NoError(err)
}

func TestRunIntegrationsResolversSuite(t *testing.T) {
	suite.Run(t, new(SnifferTestSuite))
}
