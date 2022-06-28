package sniffer

import (
	"context"
	"github.com/amit7itz/goset"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/otternose/sniffer/pkg/client"
	"github.com/sirupsen/logrus"
	"time"
)

const mapperApiUrl = "http://localhost:8080/query"
const reportInterval = 10 * time.Second
const mapperCallsTimeout = 5 * time.Second

type Sniffer struct {
	capturedRequests map[string]*goset.Set[string]
	lastReportTime   time.Time
}

func NewSniffer() *Sniffer {
	return &Sniffer{
		capturedRequests: make(map[string]*goset.Set[string]),
		lastReportTime:   time.Now(),
	}
}

func (s *Sniffer) NewCapturedRequest(srcIp string, destDns string) {
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
	mapperClient := client.NewMapperClient(mapperApiUrl)
	results := make([]client.CaptureResultForSrcIp, 0, len(s.capturedRequests))
	for srcIp, destinations := range s.capturedRequests {
		results = append(results, client.CaptureResultForSrcIp{SrcIp: srcIp, Destinations: destinations.Items()})
	}
	timeoutCtx, cancelFunc := context.WithTimeout(ctx, mapperCallsTimeout)
	defer cancelFunc()
	err := mapperClient.ReportCaptureResults(timeoutCtx, client.CaptureResults{Results: results})
	if err != nil {
		return err
	}

	// delete the reported captured requests
	s.capturedRequests = make(map[string]*goset.Set[string])
	return nil
}

func (s *Sniffer) PrintCapturedRequests() {
	logrus.Infof("Reporting captured requests of %d clients to Mapper", len(s.capturedRequests))
	for ip, dests := range s.capturedRequests {
		logrus.Debugf("%s:\n", ip)
		for _, dest := range dests.Items() {
			logrus.Debugf("\t%s\n", dest)
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
			ipLayer := packet.Layer(layers.LayerTypeIPv4)
			dnsLayer := packet.Layer(layers.LayerTypeDNS)
			if dnsLayer != nil && ipLayer != nil {
				ip, _ := ipLayer.(*layers.IPv4)
				dns, _ := dnsLayer.(*layers.DNS)
				if dns.OpCode == layers.DNSOpCodeQuery {
					for _, question := range dns.Questions {
						s.NewCapturedRequest(ip.SrcIP.String(), string(question.Name))
					}
				}
			}
		case <-time.After(reportInterval):
		}
		if s.lastReportTime.Add(reportInterval).Before(time.Now()) {
			err := s.ReportCaptureResults(ctx)
			if err != nil {
				logrus.Errorf("Failed to report captured requests to the Mapper: %s", err)
			}
		}
	}
}
