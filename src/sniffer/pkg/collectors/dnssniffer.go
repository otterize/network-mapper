package collectors

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/sirupsen/logrus"
	"time"
)

type DNSSniffer struct {
	NetworkCollector
}

func NewDNSSniffer(resolver ipresolver.IPResolver) *DNSSniffer {
	s := DNSSniffer{
		NetworkCollector{
			resolver: resolver,
		},
	}
	s.resetData()
	return &s
}

func (s *DNSSniffer) CreateDNSPacketStream() (chan gopacket.Packet, error) {
	handle, err := pcap.OpenLive("any", 0, true, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	err = handle.SetBPFFilter("udp port 53")
	if err != nil {
		return nil, err
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	return packetSource.Packets(), nil
}

func (s *DNSSniffer) HandlePacket(packet gopacket.Packet) {
	captureTime := detectCaptureTime(packet)
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer != nil && ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		dns, _ := dnsLayer.(*layers.DNS)
		if dns.OpCode == layers.DNSOpCodeQuery && dns.ResponseCode == layers.DNSResponseCodeNoErr {
			for _, answer := range dns.Answers {
				// This is the DNS Answer, so the Dst IP is the pod IP
				if answer.Type != layers.DNSTypeA && answer.Type != layers.DNSTypeAAAA {
					continue
				}
				hostname, err := s.resolver.ResolveIP(ip.DstIP.String())
				if err != nil {
					// Try to resolve the IP again after waiting for next refresh interval
					s.resolver.WaitForNextRefresh()
					hostname, err = s.resolver.ResolveIP(ip.DstIP.String())
					if err != nil {
						logrus.WithError(err).Errorf("Could not to resolve %s, skipping packet", ip.DstIP.String())
						continue
					}
				}
				s.addCapturedRequest(ip.DstIP.String(), hostname, string(answer.Name), captureTime)
			}
		}
	}
}

func detectCaptureTime(packet gopacket.Packet) time.Time {
	captureTime := packet.Metadata().CaptureInfo.Timestamp
	if captureTime.IsZero() {
		return time.Now()
	}
	return captureTime
}
