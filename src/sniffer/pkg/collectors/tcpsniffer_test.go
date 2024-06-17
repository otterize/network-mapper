package collectors

import (
	"encoding/hex"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/nilable"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
)

func TestTCPSniffer_TestHandlePacketAWS(t *testing.T) {
	controller := gomock.NewController(t)
	mockResolver := ipresolver.NewMockIPResolver(controller)
	mockResolver.EXPECT().ResolveIP("10.0.2.48").Return("client-1", nil).Times(2) // once for the initial check, and then another for verification
	mockResolver.EXPECT().Refresh().Return(nil).Times(1)

	sniffer := NewTCPSniffer(mockResolver, true)

	tcpSYN, err := hex.DecodeString("4500004000004000400600000a0002300af4784ed93d1f40a16450e500000000b002fffffe34000002043fd8010303060101080ab6a645bc0000000004020000")
	if err != nil {
		require.NoError(t, err)
	}
	packet := gopacket.NewPacket(tcpSYN, layers.LayerTypeIPv4, gopacket.Default)
	timestamp := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	packet.Metadata().CaptureInfo.Timestamp = timestamp
	sniffer.HandlePacket(packet)
	require.NoError(t, sniffer.RefreshHostsMapping())

	require.Equal(t, []mapperclient.RecordedDestinationsForSrc{
		{
			SrcIp:       "10.0.2.48",
			SrcHostname: "client-1",
			Destinations: []mapperclient.Destination{
				{
					Destination:     "10.244.120.78",
					DestinationIP:   nilable.From("10.244.120.78"),
					DestinationPort: nilable.From(8000),
					LastSeen:        timestamp,
				},
			},
		},
	}, sniffer.CollectResults())
}

func TestTCPSniffer_TestHandlePacketNonAWS(t *testing.T) {
	controller := gomock.NewController(t)
	mockResolver := ipresolver.NewMockIPResolver(controller)

	sniffer := NewTCPSniffer(mockResolver, false)

	tcpSYN, err := hex.DecodeString("4500004000004000400600000a0002300af4784ed93d1f40a16450e500000000b002fffffe34000002043fd8010303060101080ab6a645bc0000000004020000")
	if err != nil {
		require.NoError(t, err)
	}
	packet := gopacket.NewPacket(tcpSYN, layers.LayerTypeIPv4, gopacket.Default)
	timestamp := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	packet.Metadata().CaptureInfo.Timestamp = timestamp
	sniffer.HandlePacket(packet)
	require.NoError(t, sniffer.RefreshHostsMapping())

	require.Equal(t, []mapperclient.RecordedDestinationsForSrc{
		{
			SrcIp:       "10.0.2.48",
			SrcHostname: "",
			Destinations: []mapperclient.Destination{
				{
					Destination:     "10.244.120.78",
					DestinationIP:   nilable.From("10.244.120.78"),
					DestinationPort: nilable.From(8000),
					LastSeen:        timestamp,
				},
			},
		},
	}, sniffer.CollectResults())
}
