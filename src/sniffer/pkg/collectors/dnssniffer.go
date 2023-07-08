package collectors

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

type pendingCapture struct {
	srcIp string
	dest  string
	time  time.Time
}

type DNSSniffer struct {
	NetworkCollector
	resolver    ipresolver.IPResolver
	pending     []pendingCapture
	lastRefresh time.Time
}

func NewDNSSniffer(resolver ipresolver.IPResolver) *DNSSniffer {
	s := DNSSniffer{
		NetworkCollector: NetworkCollector{},
		resolver:         resolver,
		pending:          make([]pendingCapture, 0),
		lastRefresh:      time.Now().Add(-viper.GetDuration(config.HostsMappingRefreshIntervalKey)), // Should refresh immediately
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
					// Try to resolve the IP again after next refresh interval
					s.pending = append(s.pending, pendingCapture{
						srcIp: ip.DstIP.String(), dest: string(answer.Name), time: captureTime,
					})
				}
				s.addCapturedRequest(ip.DstIP.String(), hostname, string(answer.Name), captureTime)
			}
		}
	}
}

func (s *DNSSniffer) RefreshHostsMapping() error {
	err := s.resolver.Refresh()
	if err != nil {
		return err
	}

	for _, p := range s.pending {
		hostname, err := s.resolver.ResolveIP(p.srcIp)
		if err != nil {
			logrus.WithError(err).Errorf("Could not to resolve %s, skipping packet", p.srcIp)
			continue
		}
		s.addCapturedRequest(p.srcIp, hostname, p.dest, p.time)
	}
	s.pending = make([]pendingCapture, 0)
	s.lastRefresh = time.Now()
	return nil
}

func (s *DNSSniffer) GetTimeTilNextRefresh() time.Duration {
	nextRefreshTime := s.lastRefresh.Add(viper.GetDuration(config.HostsMappingRefreshIntervalKey))
	return time.Until(nextRefreshTime)
}

func detectCaptureTime(packet gopacket.Packet) time.Time {
	captureTime := packet.Metadata().CaptureInfo.Timestamp
	if captureTime.IsZero() {
		return time.Now()
	}
	return captureTime
}
