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

func TestDNSSniffer_HandlePacket(t *testing.T) {
	controller := gomock.NewController(t)
	mockResolver := ipresolver.NewMockIPResolver(controller)
	mockResolver.EXPECT().ResolveIP("10.101.81.13").Return("curl", nil).
		Times(2) // once for the initial check, and then another for verification

	sniffer := NewDNSSniffer(mockResolver)

	rawDnsResponse, err := hex.DecodeString("f84d8969309600090f090002080045000059eb6c40004011b325d05b70340a65510d0035fcb40045a621339681800001000100000000037374730975732d656173742d3109616d617a6f6e61777303636f6d0000010001c00c000100010000003c00044815ce60")
	if err != nil {
		require.NoError(t, err)
	}
	packet := gopacket.NewPacket(rawDnsResponse, layers.LayerTypeEthernet, gopacket.Default)
	timestamp := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	packet.Metadata().CaptureInfo.Timestamp = timestamp
	sniffer.HandlePacket(packet)
	_ = sniffer.RefreshHostsMapping()
	time.Sleep(1 * time.Second)

	require.Equal(t, sniffer.CollectResults(), []mapperclient.RecordedDestinationsForSrc{
		{
			SrcIp:       "10.101.81.13",
			SrcHostname: "curl",
			Destinations: []mapperclient.Destination{
				{
					Destination: "sts.us-east-1.amazonaws.com",
					LastSeen:    timestamp,
				},
			},
		},
	})
}
