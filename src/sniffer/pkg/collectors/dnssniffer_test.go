package collectors

import (
	"encoding/hex"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/nilable"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/stretchr/testify/suite"
)

type SnifferTestSuite struct {
	suite.Suite
}

func (s *SnifferTestSuite) SetupSuite() {
}

func (s *SnifferTestSuite) TestHandlePacket() {
	sniffer := NewDNSSniffer(&ipresolver.MockIPResolver{}, false)

	rawDnsResponse, err := hex.DecodeString("f84d8969309600090f090002080045000059eb6c40004011b325d05b70340a65510d0035fcb40045a621339681800001000100000000037374730975732d656173742d3109616d617a6f6e61777303636f6d0000010001c00c000100010000003c00044815ce60")
	if err != nil {
		s.Require().NoError(err)
	}
	packet := gopacket.NewPacket(rawDnsResponse, layers.LayerTypeEthernet, gopacket.Default)
	timestamp := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	packet.Metadata().CaptureInfo.Timestamp = timestamp
	sniffer.HandlePacket(packet)
	_ = sniffer.RefreshHostsMapping()

	s.Require().Equal([]mapperclient.RecordedDestinationsForSrc{
		{
			SrcIp: "10.101.81.13",
			Destinations: []mapperclient.Destination{
				{
					Destination:   "sts.us-east-1.amazonaws.com",
					DestinationIP: nilable.From("72.21.206.96"),
					LastSeen:      timestamp,
					TTL:           nilable.From(60),
				},
			},
		},
	}, sniffer.CollectResults())
}

func TestDNSSnifferSuite(t *testing.T) {
	suite.Run(t, new(SnifferTestSuite))
}
