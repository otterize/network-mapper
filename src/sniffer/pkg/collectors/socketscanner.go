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

		// Only report sockets from the client-side by checking if the local port for this socket is the same port as a listen socket.
		if _, isServersideSocket := listenPorts[sock.LocalAddr.Port]; !isServersideSocket {
			// The hostname we have here is the hostname for the client.
			s.addCapturedRequest(sock.LocalAddr.IP.String(), hostname, sock.RemoteAddr.IP.String(), sock.RemoteAddr.IP.String(), time.Now())
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
