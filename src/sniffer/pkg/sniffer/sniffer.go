package sniffer

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/sniffer/pkg/prometheus"
	"time"

	"github.com/otterize/network-mapper/src/sniffer/pkg/collectors"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Sniffer struct {
	dnsSniffer     *collectors.DNSSniffer
	socketScanner  *collectors.SocketScanner
	lastReportTime time.Time
	mapperClient   mapperclient.MapperClient
}

func NewSniffer(mapperClient mapperclient.MapperClient) *Sniffer {
	return &Sniffer{
		dnsSniffer:    collectors.NewDNSSniffer(ipresolver.NewProcFSIPResolver()),
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
	logrus.Infof("Reporting captured requests of %d clients to Mapper", len(results))

	go func() {
		timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
		defer cancelFunc()

		err := s.mapperClient.ReportCaptureResults(timeoutCtx, mapperclient.CaptureResults{Results: results})
		if err != nil {
			logrus.WithError(err).Error("Failed to report capture results")
			return
		}
		logrus.Infof("Reported captured requests of %d clients to Mapper", len(results))
		prometheus.IncrementDNSCaptureReports(len(results))
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
			return
		}
		logrus.Infof("Reported scanned requests of %d clients to Mapper", len(results))
		prometheus.IncrementSocketScanReports(len(results))
	}()
}

func (s *Sniffer) report(ctx context.Context) {
	s.reportSocketScanResults(ctx)
	s.reportCaptureResults(ctx)
	s.lastReportTime = time.Now()
}

func (s *Sniffer) getTimeTilNextReport() time.Duration {
	nextReportTime := s.lastReportTime.Add(viper.GetDuration(config.SnifferReportIntervalKey))
	return time.Until(nextReportTime)
}

func (s *Sniffer) RunForever(ctx context.Context) error {
	packetsChan, err := s.dnsSniffer.CreateDNSPacketStream()
	if err != nil {
		return errors.Wrap(err)
	}

	for {
		select {
		case packet := <-packetsChan:
			s.dnsSniffer.HandlePacket(packet)
		case <-time.After(s.dnsSniffer.GetTimeTilNextRefresh()):
			if err := s.dnsSniffer.RefreshHostsMapping(); err != nil {
				logrus.WithError(err).Error("Failed to refresh ip->host resolving map")
			}
		case <-time.After(s.getTimeTilNextReport()):
			if err := s.socketScanner.ScanProcDir(); err != nil {
				logrus.WithError(err).Error("Failed to scan proc dir for sockets")
			}
			// Flush pending packets before reporting
			if err := s.dnsSniffer.RefreshHostsMapping(); err != nil {
				logrus.WithError(err).Error("Failed to refresh ip->host resolving map")
			}
			// Actual server request is async, won't block packet handling
			s.report(ctx)
		}
	}
}
