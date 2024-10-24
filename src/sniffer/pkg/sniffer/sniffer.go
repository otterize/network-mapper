package sniffer

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/shared/isrunningonaws"
	"github.com/otterize/network-mapper/src/sniffer/pkg/collectors"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/otterize/network-mapper/src/sniffer/pkg/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

type Sniffer struct {
	dnsSniffer     *collectors.DNSSniffer
	socketScanner  *collectors.SocketScanner
	tcpSniffer     *collectors.TCPSniffer
	lastReportTime time.Time
	mapperClient   mapperclient.MapperClient
}

func NewSniffer(mapperClient mapperclient.MapperClient) *Sniffer {
	procFSIPResolver := ipresolver.NewProcFSIPResolver()
	isRunningOnAws := isrunningonaws.Check()

	return &Sniffer{
		dnsSniffer:    collectors.NewDNSSniffer(procFSIPResolver, isRunningOnAws),
		tcpSniffer:    collectors.NewTCPSniffer(procFSIPResolver, isRunningOnAws),
		socketScanner: collectors.NewSocketScanner(),
		mapperClient:  mapperClient,
	}
}

func (s *Sniffer) reportCaptureResults(ctx context.Context) {
	results := s.dnsSniffer.CollectResults()
	if len(results) == 0 {
		logrus.Debugf("No captured sniffed requests to report")
		return
	}
	logrus.Debugf("Reporting captured requests of %d clients to Mapper", len(results))

	go func() {
		timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
		defer cancelFunc()

		err := s.mapperClient.ReportCaptureResults(timeoutCtx, mapperclient.CaptureResults{Results: results})
		if err != nil {
			logrus.WithError(err).Error("Failed to report capture results")
			return
		}
		logrus.Debugf("Reported captured requests of %d clients to Mapper", len(results))
		prometheus.IncrementDNSCaptureReports(len(results))
	}()
}

func (s *Sniffer) reportTCPCaptureResults(ctx context.Context) {
	results := s.tcpSniffer.CollectResults()
	if len(results) == 0 {
		logrus.Debugf("No TCP captured sniffed requests to report")
		return
	}
	logrus.Debugf("Reporting TCP captured requests of %d clients to Mapper", len(results))

	go func() {
		timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
		defer cancelFunc()

		logrus.Debugf("Reporting TCP captured requests of %d clients to Mapper", len(results))
		err := s.mapperClient.ReportTCPCaptureResults(timeoutCtx, mapperclient.CaptureTCPResults{Results: results})
		if err != nil {
			logrus.WithError(err).Error("Failed to report capture results")
		}
	}()
}

func (s *Sniffer) reportSocketScanResults(ctx context.Context) {
	results := s.socketScanner.CollectResults()
	if len(results) == 0 {
		logrus.Debugf("No socket scanned connections to report")
		return
	}
	logrus.Debugf("Reporting scanned requests of %d clients to Mapper", len(results))

	go func() {
		timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
		defer cancelFunc()

		err := s.mapperClient.ReportSocketScanResults(timeoutCtx, mapperclient.SocketScanResults{Results: results})
		if err != nil {
			logrus.WithError(err).Error("Failed to report socket scan results")
			return
		}
		logrus.Debugf("Reported scanned requests of %d clients to Mapper", len(results))
		prometheus.IncrementSocketScanReports(len(results))
	}()
}

func (s *Sniffer) report(ctx context.Context) {
	s.reportSocketScanResults(ctx)
	s.reportCaptureResults(ctx)
	s.reportTCPCaptureResults(ctx)
	s.lastReportTime = time.Now()
}

func (s *Sniffer) getTimeTilNextReport() time.Duration {
	nextReportTime := s.lastReportTime.Add(viper.GetDuration(config.SnifferReportIntervalKey))
	return time.Until(nextReportTime)
}

func (s *Sniffer) RunForever(ctx context.Context) error {
	dnsPacketsChan, err := s.dnsSniffer.CreateDNSPacketStream()
	if err != nil {
		return errors.Wrap(err)
	}

	tcpPacketsChan, err := s.tcpSniffer.CreateTCPPacketStream()
	if err != nil {
		return errors.Wrap(err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case packet := <-dnsPacketsChan:
			s.dnsSniffer.HandlePacket(packet)
		case packet := <-tcpPacketsChan:
			s.tcpSniffer.HandlePacket(packet)
		case <-time.After(s.dnsSniffer.GetTimeTilNextRefresh()):
			if err := s.dnsSniffer.RefreshHostsMapping(); err != nil {
				logrus.WithError(err).Error("Failed to refresh ip->host resolving map for DNS")
			}
		case <-time.After(s.tcpSniffer.GetTimeTilNextRefresh()):
			if err := s.tcpSniffer.RefreshHostsMapping(); err != nil {
				logrus.WithError(err).Error("Failed to refresh ip->host resolving map for TCP")
			}
		case <-time.After(s.getTimeTilNextReport()):
			if err := s.socketScanner.ScanProcDir(); err != nil {
				logrus.WithError(err).Error("Failed to scan proc dir for sockets")
			}
			// Flush pending packets before reporting
			if err := s.dnsSniffer.RefreshHostsMapping(); err != nil {
				logrus.WithError(err).Error("Failed to refresh ip->host resolving map for DNS")
			}
			if err := s.tcpSniffer.RefreshHostsMapping(); err != nil {
				logrus.WithError(err).Error("Failed to refresh ip->host resolving map for TCP")
			}
			// Actual server request is async, won't block packet handling
			s.report(ctx)
		}
	}
}
