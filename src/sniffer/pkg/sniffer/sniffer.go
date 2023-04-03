package sniffer

import (
	"context"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/otterize/network-mapper/src/sniffer/pkg/socketscanner"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

// capturesMap is a map of source IP to a map of destination DNS to the last time it was seen
type capturesMap map[string]map[string]time.Time

type Sniffer struct {
	capturedRequests capturesMap
	socketScanner    *socketscanner.SocketScanner
	lastReportTime   time.Time
	mapperClient     mapperclient.MapperClient
}

func NewSniffer(mapperClient mapperclient.MapperClient) *Sniffer {
	return &Sniffer{
		capturedRequests: make(capturesMap),
		socketScanner:    socketscanner.NewSocketScanner(mapperClient),
		lastReportTime:   time.Now(),
		mapperClient:     mapperClient,
	}
}

func (s *Sniffer) HandlePacket(packet gopacket.Packet) {
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
				s.addCapturedRequest(ip.DstIP.String(), string(answer.Name), captureTime)
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

func (s *Sniffer) addCapturedRequest(srcIp string, destDns string, seenAt time.Time) {
	if _, ok := s.capturedRequests[srcIp]; !ok {
		s.capturedRequests[srcIp] = make(map[string]time.Time)
	}
	s.capturedRequests[srcIp][destDns] = seenAt
}

func (s *Sniffer) ReportCaptureResults(ctx context.Context) error {
	s.lastReportTime = time.Now()
	if len(s.capturedRequests) == 0 {
		logrus.Debugf("No captured requests to report")
		return nil
	}
	s.PrintCapturedRequests()
	results := getCaptureResults(s.capturedRequests)
	timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(sharedconfig.CallsTimeoutKey))
	defer cancelFunc()

	logrus.Infof("Reporting captured requests of %d clients to Mapper", len(s.capturedRequests))
	err := s.mapperClient.ReportCaptureResults(timeoutCtx, mapperclient.CaptureResults{Results: results})
	if err != nil {
		return err
	}

	// delete the reported captured requests
	s.capturedRequests = make(capturesMap)
	return nil
}

func getCaptureResults(capturedRequests capturesMap) []mapperclient.CaptureResultForSrcIp {
	results := make([]mapperclient.CaptureResultForSrcIp, 0, len(capturedRequests))
	for srcIp, destDNSToTime := range capturedRequests {
		destinations := make([]mapperclient.Destination, 0)
		for destDNS, lastSeen := range destDNSToTime {
			destinations = append(destinations, mapperclient.Destination{Destination: destDNS, LastSeen: lastSeen})
		}
		results = append(results, mapperclient.CaptureResultForSrcIp{SrcIp: srcIp, Destinations: destinations})
	}
	return results
}

func (s *Sniffer) PrintCapturedRequests() {
	for ip, destinations := range s.capturedRequests {
		logrus.Debugf("%s:\n", ip)
		for destDNS, lastSeen := range destinations {
			logrus.Debugf("\t%s, %s\n", destDNS, lastSeen)
		}
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
		case <-time.After(viper.GetDuration(sharedconfig.ReportIntervalKey)):
		}
		if s.lastReportTime.Add(viper.GetDuration(sharedconfig.ReportIntervalKey)).Before(time.Now()) {
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
