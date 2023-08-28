package sniffer

import (
	"context"
	"time"

	"github.com/otterize/network-mapper/src/sniffer/pkg/collectors"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Sniffer struct {
	dnsSniffer       *collectors.DNSSniffer
	tcpSniffer       *collectors.TCPSniffer
	procFSIPResolver *ipresolver.ProcFSIPResolver
	socketScanner    *collectors.SocketScanner
	lastReportTime   time.Time
	mapperClient     mapperclient.MapperClient
	lastRefresh      time.Time
}

func NewSniffer(mapperClient mapperclient.MapperClient) *Sniffer {
	procFSIPResolver := ipresolver.NewProcFSIPResolver()
	return &Sniffer{
		dnsSniffer:       collectors.NewDNSSniffer(procFSIPResolver),
		tcpSniffer:       collectors.NewTCPSniffer(procFSIPResolver),
		procFSIPResolver: procFSIPResolver,
		socketScanner:    collectors.NewSocketScanner(),
		mapperClient:     mapperClient,
		lastRefresh:      time.Now().Add(-viper.GetDuration(config.HostsMappingRefreshIntervalKey)), // Should refresh immediately

	}
}

func (s *Sniffer) getTimeTilNextRefresh() time.Duration {
	nextRefreshTime := s.lastRefresh.Add(viper.GetDuration(config.HostsMappingRefreshIntervalKey))
	return time.Until(nextRefreshTime)
}

func (s *Sniffer) reportCaptureResults(ctx context.Context) {
	results := s.dnsSniffer.CollectResults()
	if len(results) == 0 {
		logrus.Debugf("No captured sniffed requests to report")
		return
	}
	logrus.Infof("Reporting captured requests of %d clients to Mapper", len(results))

	go func() {
		timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
		defer cancelFunc()

		err := s.mapperClient.ReportCaptureResults(timeoutCtx, mapperclient.CaptureResults{Results: results})
		if err != nil {
			logrus.WithError(err).Error("Failed to report capture results")
		}
	}()
}

func (s *Sniffer) reportTCPCaptureResults(ctx context.Context) {
	results := s.tcpSniffer.CollectResults()
	if len(results) == 0 {
		logrus.Debugf("No captured sniffed requests to report")
		return
	}
	logrus.Infof("Reporting captured requests of %d clients to Mapper", len(results))

	go func() {
		timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
		defer cancelFunc()

		err := s.mapperClient.ReportTCPCaptureResults(timeoutCtx, mapperclient.CaptureResults{Results: results})
		if err != nil {
			logrus.WithError(err).Error("Failed to report capture results")
		}
	}()
}

func (s *Sniffer) reportSocketScanResults(ctx context.Context) {
	results := s.socketScanner.CollectResults()
	if len(results) == 0 {
		logrus.Debugf("No scanned tcp connections to report")
		return
	}
	logrus.Infof("Reporting scanned requests of %d clients to Mapper", len(results))

	go func() {
		timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
		defer cancelFunc()

		err := s.mapperClient.ReportSocketScanResults(timeoutCtx, mapperclient.SocketScanResults{Results: results})
		if err != nil {
			logrus.WithError(err).Error("Failed to report socket scan results")
		}
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
		return err
	}

	tcpPacketsChan, err := s.tcpSniffer.CreateTCPPacketStream()
	if err != nil {
		return err
	}

	for {
		select {
		case packet := <-dnsPacketsChan:
			s.dnsSniffer.HandlePacket(packet)
		case packet := <-tcpPacketsChan:
			s.tcpSniffer.HandlePacket(packet)
		case <-time.After(s.getTimeTilNextRefresh()):
			err := s.procFSIPResolver.Refresh()
			if err != nil {
				logrus.WithError(err).Error("Failed to refresh ip->host resolving map")
			}
			if err := s.dnsSniffer.RefreshHostsMapping(); err != nil {
				logrus.WithError(err).Error("Failed to refresh DNS resolving map")
			}

			if err := s.tcpSniffer.RefreshHostsMapping(); err != nil {
				logrus.WithError(err).Error("Failed to refresh TCP resolving map")
			}
			s.lastRefresh = time.Now()
		case <-time.After(s.getTimeTilNextReport()):
			if err := s.socketScanner.ScanProcDir(); err != nil {
				logrus.WithError(err).Error("Failed to scan proc dir for sockets")
			}
			// Flush pending packets before reporting
			err := s.procFSIPResolver.Refresh()
			if err != nil {
				logrus.WithError(err).Error("Failed to refresh ip->host resolving map")
			}
			if err := s.dnsSniffer.RefreshHostsMapping(); err != nil {
				logrus.WithError(err).Error("Failed to refresh DNS resolving map")
			}

			if err := s.tcpSniffer.RefreshHostsMapping(); err != nil {
				logrus.WithError(err).Error("Failed to refresh TCP resolving map")
			}
			s.lastRefresh = time.Now()
			// Actual server request is async, won't block packet handling
			s.report(ctx)
		}
	}
}
