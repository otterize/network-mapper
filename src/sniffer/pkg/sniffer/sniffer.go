package sniffer

import (
	"context"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/otterize/network-mapper/src/sniffer/pkg/socketscanner"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type UniqueRequest struct {
	srcIP       string
	srcHostname string
	destDNS     string
}

// For each unique request info, we store the time of the last request (no need to report duplicates)
type capturesMap map[UniqueRequest]time.Time

type Sniffer struct {
	capturedRequests capturesMap
	socketScanner    *socketscanner.SocketScanner
	lastReportTime   time.Time
	mapperClient     mapperclient.MapperClient
	resolver         ipresolver.IPResolver
}

func NewSniffer(mapperClient mapperclient.MapperClient, resolver ipresolver.IPResolver) *Sniffer {
	return &Sniffer{
		capturedRequests: make(capturesMap),
		socketScanner:    socketscanner.NewSocketScanner(mapperClient),
		lastReportTime:   time.Now(),
		mapperClient:     mapperClient,
		resolver:         resolver,
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
				hostname, err := s.resolver.ResolveIP(ip.DstIP.String())
				if err != nil {
					// Try to resolve the IP again after waiting for next refresh interval
					// TODO: Should we do this asynchronously?
					s.resolver.WaitForNextRefresh()
					hostname, err = s.resolver.ResolveIP(ip.DstIP.String())
					if err != nil {
						logrus.WithError(err).Errorf("Could not to resolve %s, skipping packet", ip.DstIP.String())
						continue
					}
				}
				s.addCapturedRequest(ip.DstIP.String(), hostname, string(answer.Name), captureTime)
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

func (s *Sniffer) addCapturedRequest(srcIp string, srcHost string, destDns string, seenAt time.Time) {
	req := UniqueRequest{srcIp, srcHost, destDns}
	s.capturedRequests[req] = seenAt
}

func (s *Sniffer) ReportCaptureResults(ctx context.Context) error {
	s.lastReportTime = time.Now()
	if len(s.capturedRequests) == 0 {
		logrus.Debugf("No captured requests to report")
		return nil
	}
	results := normalizeCaptureResults(s.capturedRequests)
	timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
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

func normalizeCaptureResults(capturedRequests capturesMap) []mapperclient.CaptureResultForSrcIp {
	type srcInfo struct {
		Ip       string
		Hostname string
	}
	srcToDests := make(map[srcInfo][]mapperclient.Destination)

	for reqInfo, reqLastSeen := range capturedRequests {
		src := srcInfo{Ip: reqInfo.srcIP, Hostname: reqInfo.srcHostname}

		if _, ok := srcToDests[src]; !ok {
			srcToDests[src] = make([]mapperclient.Destination, 0)
		}
		srcToDests[src] = append(srcToDests[src], mapperclient.Destination{Destination: reqInfo.destDNS, LastSeen: reqLastSeen})
	}

	results := make([]mapperclient.CaptureResultForSrcIp, 0)
	for src, destinations := range srcToDests {
		// Debug print the results
		logrus.Debugf("%s (%s):\n", src.Ip, src.Hostname)
		for _, dest := range destinations {
			logrus.Debugf("\t%s, %s\n", dest.Destination, dest.LastSeen)
		}

		results = append(results, mapperclient.CaptureResultForSrcIp{SrcIp: src.Ip, SrcHostname: src.Hostname, Destinations: destinations})
	}

	return results
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
		case <-time.After(viper.GetDuration(config.SnifferReportIntervalKey)):
		}
		if s.lastReportTime.Add(viper.GetDuration(config.SnifferReportIntervalKey)).Before(time.Now()) {
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
