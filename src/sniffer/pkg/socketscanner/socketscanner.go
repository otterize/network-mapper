package socketscanner

import (
	"context"
	"fmt"
	"github.com/amit7itz/goset"
	"github.com/otterize/go-procnet/procnet"
	"github.com/otterize/network-mapper/src/sniffer/pkg/client"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/spf13/viper"
	"os"
	"strconv"
)

type SocketScanner struct {
	scanResults  map[string]*goset.Set[string]
	mapperClient client.MapperClient
}

func NewSocketScanner(mapperClient client.MapperClient) *SocketScanner {
	return &SocketScanner{
		scanResults:  make(map[string]*goset.Set[string]),
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
				s.scanResults[sock.RemoteAddr.IP.String()] = goset.NewSet(sock.LocalAddr.IP.String())
			} else {
				s.scanResults[sock.RemoteAddr.IP.String()].Add(sock.LocalAddr.IP.String())
			}
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
	results := client.SocketScanResults{}
	for srcIp, destIps := range s.scanResults {
		results.Results = append(results.Results, client.SocketScanResultForSrcIp{SrcIp: srcIp, DestIps: destIps.Items()})
	}
	err := s.mapperClient.ReportSocketScanResults(ctx, results)
	if err != nil {
		return err
	}
	s.scanResults = make(map[string]*goset.Set[string])
	return nil
}
