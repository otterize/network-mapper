package sniffer

import (
	"context"
	"github.com/amit7itz/goset"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/network-mapper/sniffer/pkg/client"
	"github.com/otterize/network-mapper/sniffer/pkg/config"
	"github.com/otterize/network-mapper/sniffer/pkg/socketscanner"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

type Sniffer struct {
	capturedRequests map[string]*goset.Set[string]
	socketScanner    *socketscanner.SocketScanner
	lastReportTime   time.Time
	mapperClient     client.MapperClient
}

func NewSniffer(mapperClient client.MapperClient) *Sniffer {
	return &Sniffer{
		capturedRequests: make(map[string]*goset.Set[string]),
		socketScanner:    socketscanner.NewSocketScanner(mapperClient),
		lastReportTime:   time.Now(),
		mapperClient:     mapperClient,
	}
}

func (s *Sniffer) HandlePacket(packet gopacket.Packet) {
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
				s.addCapturedRequest(ip.DstIP.String(), string(answer.Name))
			}
		}
	}
}

func (s *Sniffer) addCapturedRequest(srcIp string, destDns string) {
	if _, ok := s.capturedRequests[srcIp]; !ok {
		s.capturedRequests[srcIp] = goset.NewSet[string](destDns)
	} else {
		s.capturedRequests[srcIp].Add(destDns)
	}
}

func (s *Sniffer) ReportCaptureResults(ctx context.Context) error {
	s.lastReportTime = time.Now()
	if len(s.capturedRequests) == 0 {
		logrus.Debugf("No captured requests to report")
		return nil
	}
	s.PrintCapturedRequests()
	results := make([]client.CaptureResultForSrcIp, 0, len(s.capturedRequests))
	for srcIp, destinations := range s.capturedRequests {
		results = append(results, client.CaptureResultForSrcIp{SrcIp: srcIp, Destinations: destinations.Items()})
	}
	timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
	defer cancelFunc()

	logrus.Infof("Reporting captured requests of %d clients to Mapper", len(s.capturedRequests))
	err := s.mapperClient.ReportCaptureResults(timeoutCtx, client.CaptureResults{Results: results})
	if err != nil {
		return err
	}

	// delete the reported captured requests
	s.capturedRequests = make(map[string]*goset.Set[string])
	return nil
}

func (s *Sniffer) PrintCapturedRequests() {
	for ip, dests := range s.capturedRequests {
		logrus.Debugf("%s:\n", ip)
		dests.For(func(dest string) {
			logrus.Debugf("\t%s\n", dest)
		})
	}
}

func (s *Sniffer) RunForever(ctx context.Context) error {
	handle, err := pcap.OpenLive("any", 0, true, pcap.BlockForever)
	if err != nil {
		return err
	}
	err = handle.SetBPFFilter("udp port 53")
	if err != nil {
		return err
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetsChan := packetSource.Packets()

	for {
		select {
		case packet := <-packetsChan:
			s.HandlePacket(packet)
		case <-time.After(viper.GetDuration(config.ReportIntervalKey)):
		}
		if s.lastReportTime.Add(viper.GetDuration(config.ReportIntervalKey)).Before(time.Now()) {
			err := s.socketScanner.ScanProcDir()
			if err != nil {
				logrus.WithError(err).Error("Failed to scan proc dir for sockets")
			}
			err = s.socketScanner.ReportSocketScanResults(ctx)
			if err != nil {
				logrus.WithError(err).Error("Failed to report socket scan result to mapper")
			}
			err = s.ReportCaptureResults(ctx)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to report captured requests to mapper")
			}
		}
	}
}
