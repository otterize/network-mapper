package collectors

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

type pendingCapture struct {
	srcIp            string
	srcHostname      string
	destHostnameOrIP string
	destIPFromDNS    string // The destination IP, if it is known at the time of capture.
	time             time.Time
	ttl              int
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
		return nil, errors.Wrap(err)
	}
	err = handle.SetBPFFilter("udp port 53")
	if err != nil {
		return nil, errors.Wrap(err)
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
					s.pending = append(s.pending, pendingCapture{
						srcIp:            ip.DstIP.String(),
						srcHostname:      hostname,
						destHostnameOrIP: string(answer.Name),
						destIPFromDNS:    answer.IP.String(),
						time:             captureTime,
						ttl:              int(answer.TTL),
					})
				}
			}
		}
	}
}

func (s *DNSSniffer) RefreshHostsMapping() error {
	err := s.resolver.Refresh()
	if err != nil {
		return errors.Wrap(err)
	}

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
		s.addCapturedRequest(p.srcIp, hostname, p.destHostnameOrIP, p.destIPFromDNS, p.time, p.ttl)
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
