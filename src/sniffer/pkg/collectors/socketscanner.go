package collectors

import (
	"fmt"
	"github.com/otterize/go-procnet/procnet"
	"github.com/otterize/network-mapper/src/sniffer/pkg/utils"

	"time"
)

type SocketScanner struct {
	NetworkCollector
}

func NewSocketScanner() *SocketScanner {
	s := SocketScanner{
		NetworkCollector{},
	}
	s.resetData()
	return &s
}

func (s *SocketScanner) scanTcpFile(hostname string, path string) {
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

		if sock.State != procnet.Established {
			// Skip sockets that are not in ESTABLISHED state, to avoid reporting stale connections (such as connections in TIME_WAIT).
			continue
		}

		if _, ok := listenPorts[sock.LocalAddr.Port]; ok {
			// FIXME: don't check hostname since we have the server's hostname and the client's IP here. Consider reversing direction to reporting from the client's point-of-view,
			// but this requires being able to resolve service IPs
			// For example, Remote: 10.244.120.96 (loadgenerator) -> Local: 10.244.120.95 (frontend). The hostname we have is frontend (local), but the client we are attempting to report is loadgenerator.
			s.addCapturedRequest(sock.RemoteAddr.IP.String(), "", sock.LocalAddr.IP.String(), time.Now())
		}
	}
}

func (s *SocketScanner) ScanProcDir() error {
	return utils.ScanProcDirProcesses(func(_ int64, pDir string) {
		hostname, err := utils.ExtractProcessHostname(pDir)
		if err != nil {
			return
		}
		s.scanTcpFile(hostname, fmt.Sprintf("%s/net/tcp", pDir))
		s.scanTcpFile(hostname, fmt.Sprintf("%s/net/tcp6", pDir))
	})
}
