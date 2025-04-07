package collectors

import (
	"encoding/hex"
	"github.com/otterize/network-mapper/src/mapperclient"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/nilable"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
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
					SrcPorts:      []int{},
				},
			},
		},
	}, sniffer.CollectResults())
}

func (s *SnifferTestSuite) TestHandlePacketWithCNAME() {
	sniffer := NewDNSSniffer(&ipresolver.MockIPResolver{}, false)

	rawDnsResponse, err := hex.DecodeString("92e72b05f87b02af9e5f513c0800450000b1443940004011e0100af400020af4000900359c2b009d16a123e085800001000200000001036170690c6f74746572697a652d64657603636f6d0000010001036170690c6f74746572697a652d64657603636f6d000005000100000006001b08696e7465726e616c0c6f74746572697a652d64657603636f6d0008696e7465726e616c0c6f74746572697a652d64657603636f6d00000100010000000600040bdc020000002904d0000000000000")
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
			SrcIp: "10.244.0.9",
			Destinations: []mapperclient.Destination{
				{
					Destination:   "api.otterize-dev.com",
					DestinationIP: nilable.From("11.220.2.0"),
					LastSeen:      timestamp,
					TTL:           nilable.From(6),
					SrcPorts:      []int{},
				},
			},
		},
	}, sniffer.CollectResults())
}

func TestDNSSnifferSuite(t *testing.T) {
	suite.Run(t, new(SnifferTestSuite))
}
