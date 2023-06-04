package sniffer

import (
	"context"
	"encoding/hex"
	"github.com/golang/mock/gomock"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	ipresolvermocks "github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver/mocks"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	mock_client "github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient/mockclient"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type SnifferTestSuite struct {
	suite.Suite
	mockController   *gomock.Controller
	mockMapperClient *mock_client.MockMapperClient
	mockKubeFinder   *ipresolvermocks.MockIpResolver
}

func (s *SnifferTestSuite) SetupSuite() {
	s.mockController = gomock.NewController(s.T())
	s.mockMapperClient = mock_client.NewMockMapperClient(s.mockController)
	s.mockKubeFinder = ipresolvermocks.NewMockIpResolver(s.mockController)
}

func (s *SnifferTestSuite) TestHandlePacket() {
	sniffer := NewSniffer(s.mockMapperClient, s.mockKubeFinder)
	rawDnsResponse, err := hex.DecodeString("f84d8969309600090f090002080045000059eb6c40004011b325d05b70340a65510d0035fcb40045a621339681800001000100000000037374730975732d656173742d3109616d617a6f6e61777303636f6d0000010001c00c000100010000003c00044815ce60")
	packet := gopacket.NewPacket(rawDnsResponse, layers.LayerTypeEthernet, gopacket.Default)
	timestamp := time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC)

	packet.Metadata().CaptureInfo.Timestamp = timestamp

	clientIdentity := ipresolver.Identity{
		Name:      "items-service",
		Namespace: "backend",
	}
	serverIdentity := ipresolver.Identity{
		Name:      "checkout-service",
		Namespace: "backend",
	}

	s.mockKubeFinder.EXPECT().ResolveIp("10.101.81.13", timestamp).Return(clientIdentity, nil)
	s.mockKubeFinder.EXPECT().ResolveDNS("sts.us-east-1.amazonaws.com", timestamp).Return(serverIdentity, nil)

	sniffer.HandlePacket(packet)

	s.mockMapperClient.EXPECT().ReportCaptureResults(gomock.Any(), mapperclient.CaptureResults{
		Results: []mapperclient.CaptureResultForSrcIp{
			{
				Src: mapperclient.OtterizeServiceIdentityInput{
					Name:      clientIdentity.Name,
					Namespace: clientIdentity.Namespace,
				},
				Destinations: []mapperclient.Destination{
					{
						Destination: mapperclient.OtterizeServiceIdentityInput{
							Name:      serverIdentity.Name,
							Namespace: serverIdentity.Namespace,
						},
						LastSeen: timestamp,
					},
				},
			},
		},
	})
	err = sniffer.ReportCaptureResults(context.Background())
	s.Require().NoError(err)
}

func TestRunIntegrationsResolversSuite(t *testing.T) {
	suite.Run(t, new(SnifferTestSuite))
}
