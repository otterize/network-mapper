package socketscanner

import (
	"context"
	"fmt"
	"github.com/otterize/go-procnet/procnet"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/spf13/viper"
	"os"
	"strconv"
	"time"
)

type scanResultMap map[string]map[string]time.Time

type SocketScanner struct {
	scanResults  scanResultMap
	mapperClient mapperclient.MapperClient
}

func NewSocketScanner(mapperClient mapperclient.MapperClient) *SocketScanner {
	return &SocketScanner{
		scanResults:  make(scanResultMap),
		mapperClient: mapperClient,
	}
}

func (s *SocketScanner) scanTcpFile(path string) {
	socks, err := procnet.SocksFromPath(path)
	if err != nil {
		// it's likely that some files will be deleted during our iteration, so we ignore errors reading the file.
		return
	}
	listenPorts := make(map[uint16]bool)
	for _, sock := range socks {
		if sock.State == procnet.Listen {
			// LISTEN ports always appear first
			listenPorts[sock.LocalAddr.Port] = true
			continue
		}
		if sock.LocalAddr.IP.IsLoopback() || sock.RemoteAddr.IP.IsLoopback() {
			// ignore localhost connections as they are irrelevant to the mapping
			continue
		}
		if _, ok := listenPorts[sock.LocalAddr.Port]; ok {
			if _, ok := s.scanResults[sock.RemoteAddr.IP.String()]; !ok {
				s.scanResults[sock.RemoteAddr.IP.String()] = make(map[string]time.Time)
			}
			s.scanResults[sock.RemoteAddr.IP.String()][sock.LocalAddr.IP.String()] = time.Now()
		}
	}
}

func (s *SocketScanner) ScanProcDir() error {
	hostProcDir := viper.GetString(config.HostProcDirKey)
	files, err := os.ReadDir(hostProcDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if _, err := strconv.ParseInt(f.Name(), 10, 64); err != nil {
			// name is not a number, meaning it's not a process dir, skip
			continue
		}
		s.scanTcpFile(fmt.Sprintf("%s/%s/net/tcp", hostProcDir, f.Name()))
		s.scanTcpFile(fmt.Sprintf("%s/%s/net/tcp6", hostProcDir, f.Name()))
	}
	return nil
}

func (s *SocketScanner) ReportSocketScanResults(ctx context.Context) error {
	results := getModelResults(s.scanResults)
	err := s.mapperClient.ReportSocketScanResults(ctx, results)
	if err != nil {
		return err
	}
	s.scanResults = make(scanResultMap)
	return nil
}

func getModelResults(scanResults scanResultMap) mapperclient.SocketScanResults {
	results := mapperclient.SocketScanResults{}
	for srcIp, destinationsMap := range scanResults {
		destinations := make([]mapperclient.Destination, 0)
		for destIP, lastSeen := range destinationsMap {
			destinations = append(destinations, mapperclient.Destination{Destination: destIP, LastSeen: lastSeen})
		}
		results.Results = append(results.Results, mapperclient.SocketScanResultForSrcIp{
			SrcIp:   srcIp,
			DestIps: destinations,
		})
	}
	return results
}

func (s *SocketScanner) RunForever(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
