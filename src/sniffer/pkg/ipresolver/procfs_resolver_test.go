package ipresolver

import (
	"fmt"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type ProcFSIPResolverTestSuite struct {
	suite.Suite
	resolver        *ProcFSIPResolver
	mockRootProcDir string
}

func (s *ProcFSIPResolverTestSuite) SetupSuite() {
	var err error
	s.mockRootProcDir, err = os.MkdirTemp("", "testscamprocdir")
	s.Require().NoError(err)

	viper.Set(config.HostProcDirKey, s.mockRootProcDir)

	// Create couple of processes for "host" machine
	s.mockCreateProcess(1, "1.2.3.4", "host")
	s.mockCreateProcess(2, "1.2.3.4", "host")

	s.resolver = NewProcFSIPResolver()
}

func (s *ProcFSIPResolverTestSuite) TearDownSuite() {
	_ = os.RemoveAll(s.mockRootProcDir)
	//s.resolver.Stop()
}

const mockEnvironFileContent = "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\x00HOSTNAME=%s\x00TERM=xterm\x00HOME=/root\x00"
const mockFibTrieFileContent = `Main:
  +-- 0.0.0.0/0 3 0 5
     |-- 0.0.0.0
        /0 universe UNICAST
     +-- 127.0.0.0/8 2 0 2
        +-- 127.0.0.0/31 1 0 0
           |-- 127.0.0.0
              /8 host LOCAL
           |-- 127.0.0.1
              /32 host LOCAL
        |-- 127.255.255.255
           /32 link BROADCAST
     +-- 172.17.0.0/16 2 0 2
        +-- 172.17.0.0/30 2 0 2
           |-- 172.17.0.0
              /16 link UNICAST
           |-- %[1]s
              /32 host LOCAL
        |-- 172.17.255.255
           /32 link BROADCAST
Local:
  +-- 0.0.0.0/0 3 0 5
     |-- 0.0.0.0
        /0 universe UNICAST
     +-- 127.0.0.0/8 2 0 2
        +-- 127.0.0.0/31 1 0 0
           |-- 127.0.0.0
              /8 host LOCAL
           |-- 127.0.0.1
              /32 host LOCAL
        |-- 127.255.255.255
           /32 link BROADCAST
     +-- 172.17.0.0/16 2 0 2
        +-- 172.17.0.0/30 2 0 2
           |-- 172.17.0.0
              /16 link UNICAST
           |-- %[1]s
              /32 host LOCAL
        |-- 172.17.255.255
           /32 link BROADCAST
`

func (s *ProcFSIPResolverTestSuite) getMockProcDir(pid int64) string {
	return fmt.Sprintf("%s/%d", s.mockRootProcDir, pid)
}

func (s *ProcFSIPResolverTestSuite) mockCreateProcess(pid int64, ipaddr, hostname string) {
	mockProcDir := s.getMockProcDir(pid)
	s.Require().NoError(os.MkdirAll(mockProcDir, 0o777))
	s.Require().NoError(os.MkdirAll(mockProcDir+"/net", 0o777))

	mockEnvironFile := fmt.Sprintf(mockEnvironFileContent, hostname)
	mockFibTrieFile := fmt.Sprintf(mockFibTrieFileContent, ipaddr)

	s.Require().NoError(os.WriteFile(mockProcDir+"/environ", []byte(mockEnvironFile), 0o444))
	s.Require().NoError(os.WriteFile(mockProcDir+"/net/fib_trie", []byte(mockFibTrieFile), 0o444))
}

func (s *ProcFSIPResolverTestSuite) mockKillProcess(pid int64) {
	s.Require().NoError(os.RemoveAll(s.getMockProcDir(pid)))
}

func (s *ProcFSIPResolverTestSuite) TestResolverSimple() {
	s.mockCreateProcess(10, "172.17.0.1", "service-1")
	_ = s.resolver.Refresh()

	hostname, ok := s.resolver.ResolveIP("172.17.0.1")
	s.Require().True(ok)
	s.Require().Equal("service-1", hostname)

	s.mockKillProcess(10)
	_ = s.resolver.Refresh()

	hostname, ok = s.resolver.ResolveIP("172.17.0.1")
	s.Require().False(ok)
	s.Require().Equal(hostname, "")
}

func (s *ProcFSIPResolverTestSuite) TestResolverRefCount() {
	s.mockCreateProcess(20, "172.17.0.2", "service-2")
	_ = s.resolver.Refresh()

	hostname, ok := s.resolver.ResolveIP("172.17.0.2")
	s.Require().True(ok)
	s.Require().Equal("service-2", hostname)

	s.mockCreateProcess(21, "172.17.0.2", "service-2")
	s.mockCreateProcess(22, "172.17.0.2", "service-2")
	_ = s.resolver.Refresh()

	hostname, ok = s.resolver.ResolveIP("172.17.0.2")
	s.Require().True(ok)
	s.Require().Equal("service-2", hostname)

	s.mockKillProcess(20)
	s.mockKillProcess(22)
	_ = s.resolver.Refresh()

	hostname, ok = s.resolver.ResolveIP("172.17.0.2")
	s.Require().True(ok)
	s.Require().Equal("service-2", hostname)

	s.mockKillProcess(21)
	_ = s.resolver.Refresh()

	hostname, ok = s.resolver.ResolveIP("172.17.0.2")
	s.Require().False(ok)
	s.Require().Equal(hostname, "")
}

func (s *ProcFSIPResolverTestSuite) TestResolverCollision() {
	s.mockCreateProcess(30, "172.17.0.3", "service-3")
	_ = s.resolver.Refresh()

	hostname, ok := s.resolver.ResolveIP("172.17.0.3")
	s.Require().True(ok)
	s.Require().Equal("service-3", hostname)

	s.mockCreateProcess(31, "172.17.0.3", "service-3-new")
	_ = s.resolver.Refresh()

	// Newer hostname should override older one
	hostname, ok = s.resolver.ResolveIP("172.17.0.3")
	s.Require().True(ok)
	s.Require().Equal("service-3-new", hostname)

	s.mockKillProcess(31)
	_ = s.resolver.Refresh()

	// Older process isn't counted into ProcRefCount, the exit of the new one should be enough to remove the entry
	hostname, ok = s.resolver.ResolveIP("172.17.0.3")
	s.Require().False(ok)
	s.Require().Equal(hostname, "")

	s.mockKillProcess(30)
	_ = s.resolver.Refresh()
	hostname, ok = s.resolver.ResolveIP("172.17.0.3")
	s.Require().False(ok)
	s.Require().Equal(hostname, "")
}

func TestProcFSIPResolverTestSuite(t *testing.T) {
	suite.Run(t, new(ProcFSIPResolverTestSuite))
}
