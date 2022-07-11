package sniffer

import (
	"context"
	"github.com/amit7itz/goset"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/otternose/sniffer/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

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
	mapperClient := client.NewMapperClient(viper.GetString(mapperApiUrlKey))
	results := make([]client.CaptureResultForSrcIp, 0, len(s.capturedRequests))
	for srcIp, destinations := range s.capturedRequests {
		results = append(results, client.CaptureResultForSrcIp{SrcIp: srcIp, Destinations: destinations.Items()})
	}
	timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(callsTimeoutKey))
	defer cancelFunc()

	logrus.Infof("Reporting captured requests of %d clients to Mapper", len(s.capturedRequests))
	err := mapperClient.ReportCaptureResults(timeoutCtx, client.CaptureResults{Results: results})
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
				if dns.OpCode == layers.DNSOpCodeQuery && dns.ResponseCode == layers.DNSResponseCodeNoErr {
					for _, answer := range dns.Answers {
						// This is the DNS Answer, so the Dst IP is the pod IP
						s.NewCapturedRequest(ip.DstIP.String(), string(answer.Name))
					}
				}
			}
		case <-time.After(viper.GetDuration(reportIntervalKey)):
		}
		if s.lastReportTime.Add(viper.GetDuration(reportIntervalKey)).Before(time.Now()) {
			err := s.ReportCaptureResults(ctx)
			if err != nil {
				logrus.Errorf("Failed to report captured requests to the Mapper: %s", err)
			}
		}
	}
}