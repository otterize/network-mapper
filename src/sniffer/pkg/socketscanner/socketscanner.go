package socketscanner

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/amit7itz/goset"
	"github.com/otterize/otternose/sniffer/pkg/client"
	"github.com/otterize/otternose/sniffer/pkg/config"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const localhost = "127.0.0.1"

func ipFromHex(hexIp string) string {
	z, _ := hex.DecodeString(hexIp)
	return fmt.Sprintf("%d.%d.%d.%d", z[3], z[2], z[1], z[0])
}

func portFromHex(hexPort string) int {
	z, _ := hex.DecodeString(hexPort)
	return int(binary.BigEndian.Uint16(z))
}

func parsePair(hexStr string) pair {
	l := strings.Split(hexStr, ":")
	return pair{
		ip:   ipFromHex(l[0]),
		port: portFromHex(l[1]),
	}
}

type pair struct {
	ip   string
	port int
}

func (p pair) String() string {
	return fmt.Sprintf("%s:%d", p.ip, p.port)
}

type connection struct {
	local   pair
	foreign pair
}

type SocketScanner struct {
	scanResults map[string]*goset.Set[string]
}

func NewSocketScanner() *SocketScanner {
	return &SocketScanner{scanResults: make(map[string]*goset.Set[string])}
}

func (s *SocketScanner) scanTcpFile(path string) {
	rawContent, err := os.ReadFile(path)
	if err != nil {
		// it's likely that some files will be deleted during our iteration, so we ignore errors reading the file.
		return
	}
	content := string(rawContent)
	listenPorts := make(map[int]bool)
	connections := make([]connection, 0)
	for i, line := range strings.Split(content, "\n") {
		if i == 0 {
			continue
		}
		line = strings.TrimSpace(line)
		parts := strings.Split(line, " ")
		if len(parts) < 3 {
			continue
		}
		local := parsePair(parts[1])
		foreign := parsePair(parts[2])
		if local.ip == localhost || foreign.ip == localhost {
			// ignore localhost connections as they are irrelevant to the mapping
			continue
		}
		if parts[3] == "0A" {
			// LISTEN port
			listenPorts[local.port] = true
		} else {
			connections = append(connections, connection{local: local, foreign: foreign})
		}
	}
	for _, connection := range connections {
		if _, ok := listenPorts[connection.local.port]; ok {
			if _, ok := s.scanResults[connection.foreign.ip]; !ok {
				s.scanResults[connection.foreign.ip] = goset.NewSet(connection.local.ip)
			} else {
				s.scanResults[connection.foreign.ip].Add(connection.local.ip)
			}
		}
	}
}

func (s *SocketScanner) ScanProcDir() error {
	hostProcDir := viper.GetString(config.HostProcDirKey)
	files, err := ioutil.ReadDir(hostProcDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if _, err := strconv.ParseInt(f.Name(), 10, 64); err != nil {
			// name is not a number, meaning it's not a process dir, skip
			continue
		}
		s.scanTcpFile(fmt.Sprintf("%s/%s/net/tcp", hostProcDir, f.Name()))
	}
	return nil
}

func (s *SocketScanner) ReportSocketScanResults(ctx context.Context) error {
	mapperClient := client.NewMapperClient(viper.GetString(config.MapperApiUrlKey))
	results := client.SocketScanResults{}
	for srcIp, destIps := range s.scanResults {
		results.Results = append(results.Results, client.SocketScanResultForSrcIp{SrcIp: srcIp, DestIps: destIps.Items()})
	}
	err := mapperClient.ReportSocketScanResults(ctx, results)
	if err != nil {
		return err
	}
	s.scanResults = make(map[string]*goset.Set[string])
	return nil
}
