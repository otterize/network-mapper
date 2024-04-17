package collectors

import (
	"context"
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
	"net"
	"sync"
	"time"
)

type pendingCapture struct {
	srcIp            string
	srcHostname      string
	destHostnameOrIP string
	destIPFromDNS    string // The destination IP, if it is known at the time of capture.
	time             time.Time
	ttl              nilable.Nilable[int]
}

type DNSSniffer struct {
	NetworkCollector
	resolver       ipresolver.IPResolver
	pending        []pendingCapture
	lastRefresh    time.Time
	isRunningOnAWS bool
}

func NewDNSSniffer(resolver ipresolver.IPResolver, isRunningOnAWS bool) *DNSSniffer {
	s := DNSSniffer{
		NetworkCollector: NetworkCollector{},
		resolver:         resolver,
		pending:          make([]pendingCapture, 0),
		lastRefresh:      time.Now().Add(-viper.GetDuration(config.HostsMappingRefreshIntervalKey)), // Should refresh immediately
		isRunningOnAWS:   isRunningOnAWS,
	}
	s.resetData()
	return &s
}

type PacketChannelCombiner struct {
	Channels     []chan gopacket.Packet
	combined     chan gopacket.Packet
	combinedOnce sync.Once
}

func NewPacketChannelCombiner(channels []chan gopacket.Packet) *PacketChannelCombiner {
	return &PacketChannelCombiner{
		Channels: channels,
	}
}

func (p *PacketChannelCombiner) Packets() chan gopacket.Packet {
	p.combinedOnce.Do(func() {
		p.combined = make(chan gopacket.Packet)
		for _, c := range p.Channels {
			go func(channel chan gopacket.Packet) {
				for packet := range channel {
					p.combined <- packet
				}
			}(c)
		}
	})
	return p.combined
}

func (s *DNSSniffer) CreatePacketChannelForInterface(iface net.Interface) (result chan gopacket.Packet, err error) {
	doneCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() {
		defer cancel()
		handle, openLiveErr := pcap.OpenLive(iface.Name, 0, true, pcap.BlockForever)
		if openLiveErr != nil {
			err = errors.Wrap(openLiveErr)
			return
		}
		bpfErr := handle.SetBPFFilter("udp port 53")
		if bpfErr != nil {
			err = errors.Wrap(bpfErr)
			return
		}

		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		result = packetSource.Packets()
		return
	}()
	<-doneCtx.Done()
	if errors.Is(doneCtx.Err(), context.DeadlineExceeded) {
		return nil, errors.Errorf("timed out starting capture on interface '%s': %w", iface.Name, doneCtx.Err())
	}
	if err != nil {
		return nil, errors.Errorf("failed to start capture on interface '%s': %w", iface.Name, err)
	}
	return result, nil
}

func (s *DNSSniffer) CreateDNSPacketStream() (chan gopacket.Packet, error) {
	interfaceList, err := net.Interfaces()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	packetChans := make([]chan gopacket.Packet, 0)
	for _, iface := range interfaceList {
		logrus.Debugf("Starting capture on interface '%s'", iface.Name)
		packetChannel, err := s.CreatePacketChannelForInterface(iface)
		if err != nil {
			logrus.WithError(err).Errorf("failed to open packet channel for interface '%s', skipping", iface.Name)
			continue
		}
		packetChans = append(packetChans, packetChannel)
	}
	if len(packetChans) == 0 {
		return nil, errors.New("no captures opened successfully")
	}
	return NewPacketChannelCombiner(packetChans).Packets(), nil
}

func (s *DNSSniffer) HandlePacket(packet gopacket.Packet) {
	if !viper.GetBool(sharedconfig.EnableDNSKey) {
		return
	}

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
				if !s.isRunningOnAWS {
					s.addCapturedRequest(ip.DstIP.String(), "", string(answer.Name), answer.IP.String(), captureTime, nilable.From(int(answer.TTL)), nil)
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
						ttl:              nilable.From(int(answer.TTL)),
					})
				}
			}
		}
	}
}

func (s *DNSSniffer) RefreshHostsMapping() error {
	if !s.isRunningOnAWS {
		return nil
	}
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
		s.addCapturedRequest(p.srcIp, hostname, p.destHostnameOrIP, p.destIPFromDNS, p.time, p.ttl, nil)
	}
	s.pending = make([]pendingCapture, 0)
	return nil
}

func (s *DNSSniffer) GetTimeTilNextRefresh() time.Duration {
	nextRefreshTime := s.lastRefresh.Add(viper.GetDuration(config.HostsMappingRefreshIntervalKey))
	s.lastRefresh = time.Now()
	return time.Until(nextRefreshTime)
}

func detectCaptureTime(packet gopacket.Packet) time.Time {
	captureTime := packet.Metadata().CaptureInfo.Timestamp
	if captureTime.IsZero() {
		return time.Now()
	}
	return captureTime
}
