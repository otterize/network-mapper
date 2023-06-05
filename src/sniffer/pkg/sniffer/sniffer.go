package sniffer

import (
	"context"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/otterize/network-mapper/src/sniffer/pkg/socketscanner"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

// capturesMap is a map of source IP to a map of destination DNS to the last time it was seen
type capturesMap map[ipresolver.Identity]map[ipresolver.Identity]time.Time

type Sniffer struct {
	capturedRequests capturesMap
	socketScanner    *socketscanner.SocketScanner
	lastReportTime   time.Time
	mapperClient     mapperclient.MapperClient
	resolver         ipresolver.IpResolver
}

func NewSniffer(mapperClient mapperclient.MapperClient, ipResolver ipresolver.IpResolver) *Sniffer {
	return &Sniffer{
		capturedRequests: make(capturesMap),
		socketScanner:    socketscanner.NewSocketScanner(mapperClient, ipResolver),
		lastReportTime:   time.Now(),
		mapperClient:     mapperClient,
		resolver:         ipResolver,
	}
}

func (s *Sniffer) HandlePacket(packet gopacket.Packet) {
	packetStartTime := time.Now()
	captureTime := packet.Metadata().CaptureInfo.Timestamp
	if captureTime.IsZero() {
		logrus.Warningf("Missing capture time, using current time. raw: %s", packet.String())
		captureTime = time.Now()
	}
	handled := false

	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer != nil && ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		dns, _ := dnsLayer.(*layers.DNS)
		if dns.OpCode == layers.DNSOpCodeQuery && dns.ResponseCode == layers.DNSResponseCodeNoErr {
			logrus.Debugf("Handling packet ip: %s dns count: %d", ip.DstIP, len(dns.Answers))
			for _, answer := range dns.Answers {
				// This is the DNS Answer, so the Dst IP is the pod IP
				if answer.Type != layers.DNSTypeA && answer.Type != layers.DNSTypeAAAA {
					continue
				}
				handled = true

				startTime := time.Now()
				podIp := ip.DstIP.String()
				srcService, err := s.resolver.ResolveIp(podIp, captureTime)
				if err != nil {
					logrus.WithError(err).
						WithField("podIp", podIp).
						WithField("captureTime", captureTime).
						Warning("Failed to resolve pod name")
					continue
				}

				destDns := string(answer.Name)
				destService, err := s.resolver.ResolveDNS(destDns, captureTime)
				if err != nil {
					if err != ipresolver.NotK8sService || err != ipresolver.NotPodAddress {
						logrus.WithError(err).
							WithField("destDNS", destDns).
							WithField("captureTime", captureTime).
							Warning("Failed to resolve pod name")
					}
					continue
				}

				if _, ok := s.capturedRequests[srcService]; !ok {
					s.capturedRequests[srcService] = make(map[ipresolver.Identity]time.Time)
				}
				s.capturedRequests[srcService][destService] = captureTime
				resolvingTime := time.Since(startTime)
				logrus.Debugf("Resolved packet ip: %s dns: %s src: %s dest: %s in %s", podIp, destDns, srcService, destService, resolvingTime)
			}
		}
	}
	packetHandlingTime := time.Since(packetStartTime)
	logrus.Debugf("Handled packet in %s handled: %v", packetHandlingTime, handled)
}

func (s *Sniffer) ReportCaptureResults(ctx context.Context) error {
	s.lastReportTime = time.Now()
	if len(s.capturedRequests) == 0 {
		logrus.Debugf("No captured requests to report")
		return nil
	}
	s.PrintCapturedRequests()
	results := getCaptureResults(s.capturedRequests)
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

func getCaptureResults(capturedRequests capturesMap) []mapperclient.CaptureResultForSrcIp {
	results := make([]mapperclient.CaptureResultForSrcIp, 0, len(capturedRequests))
	for src, destToTime := range capturedRequests {
		destinations := make([]mapperclient.Destination, 0)
		for dest, lastSeen := range destToTime {
			destinations = append(destinations, mapperclient.Destination{
				Destination: mapperclient.OtterizeServiceIdentityInput{
					Name:      dest.Name,
					Namespace: dest.Namespace,
				},
				LastSeen: lastSeen,
			})
		}
		result := mapperclient.CaptureResultForSrcIp{
			Src: mapperclient.OtterizeServiceIdentityInput{
				Name:      src.Name,
				Namespace: src.Namespace,
			},
			Destinations: destinations,
		}
		results = append(results, result)
	}
	return results
}

func (s *Sniffer) PrintCapturedRequests() {
	for src, destinations := range s.capturedRequests {
		logrus.Debugf("%s/%s:\n", src.Namespace, src.Name)
		for dest, lastSeen := range destinations {
			logrus.Debugf("\t%s/%s, %s\n", dest.Namespace, dest.Name, lastSeen)
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
		case <-time.After(viper.GetDuration(config.SnifferReportIntervalKey)):
		}
		if s.lastReportTime.Add(viper.GetDuration(config.SnifferReportIntervalKey)).Before(time.Now()) {
			err := s.socketScanner.ScanProcDir()
			if err != nil {
				logrus.WithError(err).Error("Failed to scan proc dir for sockets")
			}
			// TODO: Convert IP to pod name before reporting
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
