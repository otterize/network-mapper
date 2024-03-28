package collectors

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/intents-operator/src/shared/errors"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/nilable"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

type TCPSniffer struct {
	NetworkCollector
	resolver    ipresolver.IPResolver
	pending     []pendingTCPCapture
	lastRefresh time.Time
}

type pendingTCPCapture struct {
	srcIp       string
	srcHostname string
	destIp      string
	time        time.Time
	ttl         nilable.Nilable[int]
}

func NewTCPSniffer(resolver ipresolver.IPResolver) *TCPSniffer {
	s := TCPSniffer{
		NetworkCollector: NetworkCollector{},
		resolver:         resolver,
		pending:          make([]pendingTCPCapture, 0),
		lastRefresh:      time.Now().Add(-viper.GetDuration(config.HostsMappingRefreshIntervalKey)), // Should refresh immediately
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
	if !viper.GetBool(sharedconfig.EnableTCPKey) {
		return
	}
	captureTime := detectCaptureTime(packet)
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ip, ok := ipLayer.(*layers.IPv4)
		if !ok {
			logrus.Debugf("Failed to parse IP layer")
			return
		}

		srcIP := ip.SrcIP.String()
		dstIP := ip.DstIP.String()
		localHostname, err := s.resolver.ResolveIP(srcIP)
		if err != nil {
			logrus.Debugf("Can't resolve IP addr %s, skipping", srcIP)
		} else {
			logrus.Debugf("Captured TCP SYN from %s to %s", srcIP, dstIP)

			// Resolver cache could be outdated, verify same resolving result after next poll
			s.pending = append(s.pending, pendingTCPCapture{
				srcIp: srcIP, srcHostname: localHostname, destIp: dstIP, time: captureTime,
			})
		}
	}
}

func (s *TCPSniffer) RefreshHostsMapping() error {
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
		s.addCapturedRequest(p.srcIp, hostname, p.destIp, p.destIp, p.time, p.ttl)
	}
	s.pending = make([]pendingTCPCapture, 0)
	return nil
}

func (s *TCPSniffer) GetTimeTilNextRefresh() time.Duration {
	nextRefreshTime := s.lastRefresh.Add(viper.GetDuration(config.HostsMappingRefreshIntervalKey))
	s.lastRefresh = time.Now()
	return time.Until(nextRefreshTime)
}
