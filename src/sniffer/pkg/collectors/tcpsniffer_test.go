package collectors

import (
	"encoding/hex"
	"github.com/golang/mock/gomock"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
)

func TestTCPSniffer_TestHandlePacket(t *testing.T) {
	controller := gomock.NewController(t)
	mockResolver := ipresolver.NewMockIPResolver(controller)
	mockResolver.EXPECT().ResolveIP("127.0.0.1").Return("curl", nil).
		Times(2) // once for the initial check, and then another for verification

	sniffer := NewTCPSniffer(mockResolver)

	tcpSYN, err := hex.DecodeString("4500004000004000400600007f0000017f000001d93d1f40a16450e500000000b002fffffe34000002043fd8010303060101080ab6a645bc0000000004020000")
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
			SrcIp:       "127.0.0.1",
			SrcHostname: "curl",
			Destinations: []mapperclient.Destination{
				{
					Destination: "127.0.0.1",
					LastSeen:    timestamp,
				},
			},
		},
	}, sniffer.CollectResults())
}
