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
	dnsSniffer     *collectors.DNSSniffer
	socketScanner  *collectors.SocketScanner
	lastReportTime time.Time
	mapperClient   mapperclient.MapperClient
}

func NewSniffer(mapperClient mapperclient.MapperClient, resolver ipresolver.IPResolver) *Sniffer {
	return &Sniffer{
		dnsSniffer:     collectors.NewDNSSniffer(resolver),
		socketScanner:  collectors.NewSocketScanner(resolver),
		lastReportTime: time.Now(),
		mapperClient:   mapperClient,
	}
}

func (s *Sniffer) reportCaptureResults(ctx context.Context) error {
	results := s.dnsSniffer.CollectResults()
	if len(results) == 0 {
		logrus.Debugf("No captured sniffed requests to report")
		return nil
	}

	timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
	defer cancelFunc()

	logrus.Infof("Reporting captured requests of %d clients to Mapper", len(results))
	err := s.mapperClient.ReportCaptureResults(timeoutCtx, mapperclient.CaptureResults{Results: results})
	if err != nil {
		return err
	}
	return nil
}

func (s *Sniffer) reportSocketScanResults(ctx context.Context) error {
	results := s.socketScanner.CollectResults()
	if len(results) == 0 {
		logrus.Debugf("No scanned tcp connections to report")
		return nil
	}

	timeoutCtx, cancelFunc := context.WithTimeout(ctx, viper.GetDuration(config.CallsTimeoutKey))
	defer cancelFunc()

	logrus.Infof("Reporting scanned requests of %d clients to Mapper", len(results))
	err := s.mapperClient.ReportSocketScanResults(timeoutCtx, mapperclient.SocketScanResults{Results: results})
	if err != nil {
		return err
	}
	return nil
}

func (s *Sniffer) report(ctx context.Context) {
	err := s.reportSocketScanResults(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to report socket scan result to mapper")
	}
	err = s.reportCaptureResults(ctx)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to report captured requests to mapper")
	}
	s.lastReportTime = time.Now()
}

func (s *Sniffer) getTimeTilNextReport() time.Duration {
	nextReportTime := s.lastReportTime.Add(viper.GetDuration(config.SnifferReportIntervalKey))
	return nextReportTime.Sub(time.Now())
}

func (s *Sniffer) RunForever(ctx context.Context) error {
	packetsChan, err := s.dnsSniffer.CreateDNSPacketStream()
	if err != nil {
		return err
	}

	for {
		select {
		case packet := <-packetsChan:
			s.dnsSniffer.HandlePacket(packet)
		case <-time.After(s.getTimeTilNextReport()):
			err := s.socketScanner.ScanProcDir()
			if err != nil {
				logrus.WithError(err).Error("Failed to scan proc dir for sockets")
			}

			s.report(ctx)
		}
	}
}
