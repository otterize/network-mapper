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
	resolver       ipresolver.IPResolver
	pending        []pendingTCPCapture
	lastRefresh    time.Time
	isRunningOnAWS bool
}

type pendingTCPCapture struct {
	srcIp       string
	srcHostname string
	destIp      string
	destPort    int
	time        time.Time
	ttl         nilable.Nilable[int]
}

func NewTCPSniffer(resolver ipresolver.IPResolver, isRunningOnAWS bool) *TCPSniffer {
	s := TCPSniffer{
		NetworkCollector: NetworkCollector{},
		resolver:         resolver,
		pending:          make([]pendingTCPCapture, 0),
		lastRefresh:      time.Now().Add(-viper.GetDuration(config.HostsMappingRefreshIntervalKey)), // Should refresh immediately
		isRunningOnAWS:   isRunningOnAWS,
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
	if ipLayer == nil {
		return
	}

	ip, ok := ipLayer.(*layers.IPv4)
	if !ok {
		logrus.Debugf("Failed to parse IP layer")
		return
	}

	dstPort, portFound, err := s.getDestPort(packet, ip)
	if err != nil {
		logrus.Debugf("Failed to parse TCP/UDP port: %s", err)
		return
	}
	if !portFound {
		logrus.Debugf("Port not found, skipping packet")
		return
	}

	logrus.Debugf("TCP SYN: %s to %s:%d", ip.SrcIP, ip.DstIP, dstPort)
	srcIP := ip.SrcIP.String()
	dstIP := ip.DstIP.String()
	if !s.isRunningOnAWS {
		s.addCapturedRequest(srcIP, "", dstIP, dstIP, captureTime, nilable.FromPtr[int](nil), &dstPort)
		return
	}

	localHostname, ok := s.resolver.ResolveIP(srcIP)
	if !ok {
		// This is still reported because might be ingress traffic, mapper would drop non-ingress captures with no src hostname
		destNameOrIP := dstIP
		destHostname, ok := s.resolver.ResolveIP(dstIP)
		if ok {
			destNameOrIP = destHostname
		}
		s.addCapturedRequest(srcIP, "", destNameOrIP, dstIP, captureTime, nilable.FromPtr[int](nil), &dstPort)
		return
	}

	logrus.Debugf("Captured TCP SYN from %s to %s", srcIP, dstIP)

	// Resolver cache could be outdated, verify same resolving result after next poll
	s.pending = append(s.pending, pendingTCPCapture{
		srcIp:       srcIP,
		srcHostname: localHostname,
		destIp:      dstIP,
		destPort:    dstPort,
		time:        captureTime,
	})
}

func (s *TCPSniffer) getDestPort(packet gopacket.Packet, ip *layers.IPv4) (int, bool, error) {
	layerType := ip.NextLayerType()
	var portName string
	var port int
	switch layerType {
	case layers.LayerTypeTCP:
		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if tcpLayer == nil {
			return 0, false, errors.New("Failed to parse TCP layer")
		}

		tcp, ok := tcpLayer.(*layers.TCP)
		if !ok {
			return 0, false, errors.New("Failed to parse TCP layer")
		}

		port = int(tcp.DstPort)
		portName = tcp.DstPort.String()
	default:
		return 0, false, errors.New("Unknown transport layer")
	}

	logrus.Debugf("Detected ip port %s: %s", ip.DstIP.String(), portName)
	return port, true, nil
}

func (s *TCPSniffer) RefreshHostsMapping() error {
	if !s.isRunningOnAWS {
		return nil
	}
	err := s.resolver.Refresh()
	if err != nil {
		return errors.Wrap(err)
	}

	for _, p := range s.pending {
		hostname, ok := s.resolver.ResolveIP(p.srcIp)
		if !ok {
			logrus.WithError(err).Debugf("Could not to resolve %s, skipping packet", p.srcIp)
			continue
		}
		if p.srcHostname != hostname {
			logrus.Debugf("IP %s was resolved to %s, but now resolves to %s. skipping packet", p.srcIp, p.srcHostname, hostname)
			continue
		}
		s.addCapturedRequest(p.srcIp, hostname, p.destIp, p.destIp, p.time, p.ttl, &p.destPort)
	}
	s.pending = make([]pendingTCPCapture, 0)
	return nil
}

func (s *TCPSniffer) GetTimeTilNextRefresh() time.Duration {
	nextRefreshTime := s.lastRefresh.Add(viper.GetDuration(config.HostsMappingRefreshIntervalKey))
	s.lastRefresh = time.Now()
	return time.Until(nextRefreshTime)
}
