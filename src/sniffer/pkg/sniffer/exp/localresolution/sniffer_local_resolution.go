package localresolution

import (
	"context"
	"errors"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"time"
)

type capturedResponse struct {
	srcIp   ipresolver.PodIP
	destDns ipresolver.DestDNS
	seenAt  time.Time
}

type Sniffer struct {
	capturedResponses chan capturedResponse
	lastReportTime    time.Time
	mapperClient      mapperclient.MapperClient
	resolver          ipresolver.PodResolver
}

func NewSniffer(mapperClient mapperclient.MapperClient, ipResolver ipresolver.PodResolver) *Sniffer {
	return &Sniffer{
		capturedResponses: make(chan capturedResponse, 10000),
		lastReportTime:    time.Now(),
		mapperClient:      mapperClient,
		resolver:          ipResolver,
	}
}

func (s *Sniffer) addCapturedRequest(srcIp ipresolver.PodIP, destDns ipresolver.DestDNS, seenAt time.Time) {
	select {
	case s.capturedResponses <- capturedResponse{
		srcIp:   srcIp,
		destDns: destDns,
		seenAt:  seenAt,
	}:
	default:
		logrus.Debug("dropped captured request due to full channel")
	}
}

func (s *Sniffer) HandlePacket(packet gopacket.Packet) {
	captureTime := packet.Metadata().CaptureInfo.Timestamp
	if captureTime.IsZero() {
		logrus.Warning("Missing capture time, using current time")
		captureTime = time.Now()
	}

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

				s.addCapturedRequest(ipresolver.PodIP(ip.DstIP.String()), ipresolver.DestDNS(answer.Name), captureTime)
			}
		}
	}
}

func (s *Sniffer) ReportCaptureResults(ctx context.Context, captureResults map[ipresolver.Identity]map[ipresolver.Identity]time.Time) error {
	s.lastReportTime = time.Now()
	if len(captureResults) == 0 {
		logrus.Debugf("No captured requests to report")
		return nil
	}
	s.PrintCapturedRequests(captureResults)
	results := getCaptureResults(captureResults)
	timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
	defer cancelFunc()

	logrus.Infof("Reporting captured requests of %d clients to Mapper", len(captureResults))
	err := s.mapperClient.ReportResolvedCaptureResults(timeoutCtx, results)
	if err != nil {
		return err
	}
	return nil
}

func getCaptureResults(capturedRequests map[ipresolver.Identity]map[ipresolver.Identity]time.Time) []mapperclient.ResolvedCaptureResult {
	results := make([]mapperclient.ResolvedCaptureResult, 0, len(capturedRequests))
	for src, destToTime := range capturedRequests {
		destinations := make([]mapperclient.ResolvedDestination, 0)
		for dest, lastSeen := range destToTime {
			destinations = append(destinations, mapperclient.ResolvedDestination{
				Destination: mapperclient.ResolvedOtterizeServiceIdentityInput{
					Name:      dest.Name,
					Namespace: dest.Namespace,
				},
				LastSeen: lastSeen,
			})
		}
		result := mapperclient.ResolvedCaptureResult{
			Src: mapperclient.ResolvedOtterizeServiceIdentityInput{
				Name:      src.Name,
				Namespace: src.Namespace,
			},
			Destinations: destinations,
		}
		results = append(results, result)
	}
	return results
}

func (s *Sniffer) PrintCapturedRequests(captureResults map[ipresolver.Identity]map[ipresolver.Identity]time.Time) {
	for src, destinations := range captureResults {
		logrus.Debugf("%s/%s:\n", src.Namespace, src.Name)
		for dest, lastSeen := range destinations {
			logrus.Debugf("\t%s/%s, %s\n", dest.Namespace, dest.Name, lastSeen)
		}
	}
}

func (s *Sniffer) handlePacketsForever(ctx context.Context) error {
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
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Sniffer) resolveAndReportCapturedResponsesForever(ctx context.Context) error {
	timer := time.NewTimer(viper.GetDuration(config.SnifferResolveIntervalKey))
	defer timer.Stop()
	capturedResponses := make(map[ipresolver.PodIP]map[ipresolver.DestDNS]time.Time)
	capturedResponseMaxTime := time.Time{}
	// Every 1 sec, wait for latest seen time to appear in update, or wait up to 5 sec.
	for {
		select {
		case <-timer.C:
			previousCapturedResponses := capturedResponses
			resolvedResponses, err := s.resolveResponses(ctx, capturedResponseMaxTime, previousCapturedResponses)
			if err != nil {
				return err
			}
			capturedResponses = make(map[ipresolver.PodIP]map[ipresolver.DestDNS]time.Time)
			capturedResponseMaxTime = time.Time{}
			err = s.ReportCaptureResults(ctx, resolvedResponses)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to report captured requests to mapper")
			}

		case <-ctx.Done():
			return ctx.Err()
		case resp := <-s.capturedResponses:
			if _, ok := capturedResponses[resp.srcIp]; !ok {
				capturedResponses[resp.srcIp] = make(map[ipresolver.DestDNS]time.Time)
			}
			capturedResponses[resp.srcIp][resp.destDns] = resp.seenAt
			if resp.seenAt.After(capturedResponseMaxTime) {
				capturedResponseMaxTime = resp.seenAt
			}
		}
	}
}

func (s *Sniffer) resolveResponses(
	ctx context.Context,
	capturedResponseMaxTimeInput time.Time,
	capturedResponseInput map[ipresolver.PodIP]map[ipresolver.DestDNS]time.Time) (resolvedResponses map[ipresolver.Identity]map[ipresolver.Identity]time.Time, err error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	waitErr := s.resolver.WaitForUpdateTime(ctxTimeout, capturedResponseMaxTimeInput)
	if waitErr != nil {
		if errors.Is(waitErr, context.DeadlineExceeded) {
			logrus.Warn("waited to receive pod status update before resolving pod identities, but did not receive updates in this time")
		} else {
			return nil, waitErr
		}
	}
	resolvedResponses = make(map[ipresolver.Identity]map[ipresolver.Identity]time.Time)

	for podIp, destDnsToTime := range capturedResponseInput {
		for destDns, seenAt := range destDnsToTime {
			srcService, err := s.resolver.ResolveIP(podIp, seenAt)
			if err != nil {
				logrus.WithError(err).
					WithField("podIp", podIp).
					WithField("captureTime", seenAt).
					Warning("Failed to resolve pod name")
				continue
			}

			destService, err := s.resolver.ResolveDNS(destDns, seenAt)
			if err != nil {
				if !errors.Is(err, ipresolver.NotK8sService) || !errors.Is(err, ipresolver.NotPodAddress) {
					logrus.WithError(err).
						WithField("destDNS", destDns).
						WithField("captureTime", seenAt).
						Warning("Failed to resolve pod name")
				}
				continue
			}

			if _, ok := resolvedResponses[srcService]; !ok {
				resolvedResponses[srcService] = make(map[ipresolver.Identity]time.Time)
			}
			resolvedResponses[srcService][destService] = seenAt
		}
	}
	return resolvedResponses, nil
}

func (s *Sniffer) RunForever(ctx context.Context) error {
	errGrp, errGroupCtx := errgroup.WithContext(ctx)
	errGrp.Go(func() error {
		return s.handlePacketsForever(errGroupCtx)
	})
	errGrp.Go(func() error {
		return s.resolveAndReportCapturedResponsesForever(errGroupCtx)
	})

	return errGrp.Wait()
}
