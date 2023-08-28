package collectors

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/sirupsen/logrus"
	"time"
)

type TCPSniffer struct {
	NetworkCollector
	resolver ipresolver.IPResolver
	pending  []pendingTCPCapture
}

type pendingTCPCapture struct {
	srcIp       string
	srcHostname string
	destIp      string
	time        time.Time
}

func NewTCPSniffer(resolver ipresolver.IPResolver) *TCPSniffer {
	s := TCPSniffer{
		NetworkCollector: NetworkCollector{},
		resolver:         resolver,
		pending:          make([]pendingTCPCapture, 0),
	}
	s.resetData()
	return &s
}

func setCaptureIncomingTCPSYN(handle *pcap.Handle) error {
	err := handle.SetDirection(pcap.DirectionIn)
	if err != nil {
		return err
	}
	err = handle.SetBPFFilter("tcp and tcp[tcpflags] == tcp-syn")
	if err != nil {
		return err
	}
	return nil
}

func (s *TCPSniffer) CreateTCPPacketStream() (chan gopacket.Packet, error) {
	handle, err := pcap.OpenLive("any", 0, true, pcap.BlockForever)
	if err != nil {
		return nil, err
	}

	err = setCaptureIncomingTCPSYN(handle)
	if err != nil {
		return nil, err
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	return packetSource.Packets(), nil
}

func (s *TCPSniffer) HandlePacket(packet gopacket.Packet) {
	captureTime := detectCaptureTime(packet)
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		// This is the TCP SYN on the incoming side, so the Dst IP is the pod IP
		localHostname, err := s.resolver.ResolveIP(ip.DstIP.String())
		if err != nil {
			logrus.Debugf("Can't resolve IP addr %s, skipping", ip.DstIP.String())
		} else {
			// Resolver cache could be outdated, verify same resolving result after next poll
			s.pending = append(s.pending, pendingTCPCapture{
				srcIp: ip.DstIP.String(), srcHostname: localHostname, destIp: ip.SrcIP.String(), time: captureTime,
			})
		}
	}
}

func (s *TCPSniffer) RefreshHostsMapping() error {
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
		s.addCapturedRequest(p.srcIp, hostname, p.destIp, p.time)
	}
	s.pending = make([]pendingTCPCapture, 0)
	return nil
}
