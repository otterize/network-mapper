package collectors

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/sirupsen/logrus"
	"time"
)

type pendingHostnameCapture struct {
	srcIp       string
	srcHostname string
	dest        string
	time        time.Time
}

type DNSSniffer struct {
	NetworkCollector
	resolver ipresolver.IPResolver
	pending  []pendingHostnameCapture
}

func NewDNSSniffer(resolver ipresolver.IPResolver) *DNSSniffer {
	s := DNSSniffer{
		NetworkCollector: NetworkCollector{},
		resolver:         resolver,
		pending:          make([]pendingHostnameCapture, 0),
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
					logrus.Debugf("Can't resolve IP addr %s, skipping", ip.DstIP.String())
				} else {
					// Resolver cache could be outdated, verify same resolving result after next poll
					s.pending = append(s.pending, pendingHostnameCapture{
						srcIp: ip.DstIP.String(), srcHostname: hostname, dest: string(answer.Name), time: captureTime,
					})
				}
			}
		}
	}
}

func (s *DNSSniffer) RefreshHostsMapping() error {
	for _, p := range s.pending {
		hostname, err := s.resolver.ResolveIP(p.srcIp)
		if err != nil {
			logrus.WithError(err).Debugf("Could not to resolve %s, skipping packet", p.srcIp)
			continue
		}
		if p.srcHostname != hostname {
			logrus.Debugf("IP %s was resolved to %s, but now resolves to %s. skipping packet", p.srcIp, p.srcHostname, hostname)
			continue
		}
		s.addCapturedRequest(p.srcIp, hostname, p.dest, p.time)
	}
	s.pending = make([]pendingHostnameCapture, 0)
	return nil
}

func detectCaptureTime(packet gopacket.Packet) time.Time {
	captureTime := packet.Metadata().CaptureInfo.Timestamp
	if captureTime.IsZero() {
		return time.Now()
	}
	return captureTime
}
