package collectors

import (
	"fmt"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/otterize/network-mapper/src/mapperclient"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/nilable"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"
	"gotest.tools/v3/assert"
	"os"
	"testing"
	"time"
)

type SocketScannerTestSuite struct {
	suite.Suite
}

func (s *SocketScannerTestSuite) SetupSuite() {
}

type SocketScanResultForSrcIpMatcher []mapperclient.RecordedDestinationsForSrc

func (m SocketScanResultForSrcIpMatcher) Matches(x interface{}) bool {
	actualValues, ok := x.([]mapperclient.RecordedDestinationsForSrc)
	if !ok {
		return false
	}

	if len(actualValues) != len(m) {
		return false
	}

	// Match in any order
	for _, expected := range m {
		found := false
		for _, actual := range actualValues {
			if matchSocketScanResult(expected, actual) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func matchSocketScanResult(expected, actual mapperclient.RecordedDestinationsForSrc) bool {
	if expected.SrcIp != actual.SrcIp {
		return false
	}

	if expected.SrcHostname != actual.SrcHostname {
		return false
	}

	if len(expected.Destinations) != len(actual.Destinations) {
		return false
	}

	for i := range expected.Destinations {
		if expected.Destinations[i].Destination != actual.Destinations[i].Destination {
			return false
		}
	}

	return true
}

func (m SocketScanResultForSrcIpMatcher) String() string {
	var result string
	for _, value := range m {
		result += fmt.Sprintf("{Src: %v, Dest: %v}", value.SrcIp, value.Destinations)
	}
	return result
}

func GetMatcher(expected []mapperclient.RecordedDestinationsForSrc) SocketScanResultForSrcIpMatcher {
	return expected
}

func (s *SocketScannerTestSuite) TestScanProcDir() {
	mockProcDir, err := os.MkdirTemp("", "testscamprocdir")
	s.Require().NoError(err)
	defer func() { _ = os.RemoveAll(mockProcDir) }()

	s.Require().NoError(os.MkdirAll(mockProcDir+"/100/net", 0o700))
	s.Require().NoError(os.WriteFile(mockProcDir+"/100/net/tcp", []byte(mockTcpFileContent), 0o444))
	s.Require().NoError(os.WriteFile(mockProcDir+"/100/net/tcp6", []byte(mockTcp6FileContent), 0o444))
	s.Require().NoError(os.WriteFile(mockProcDir+"/100/environ", []byte(mockEnvironFileContent), 0o444))

	viper.Set(config.HostProcDirKey, mockProcDir)

	scanner := NewSocketScanner()
	s.Require().NoError(scanner.ScanProcDir())

	// We should only see sockets that this pod serves to other clients.
	// all other sockets should be ignored (because parsing the server sides on all pods is enough)
	results := scanner.CollectResults()
	expectedResults := []mapperclient.RecordedDestinationsForSrc{
		{
			SrcIp:       "10.244.120.89",
			SrcHostname: "thisverypod",
			Destinations: []mapperclient.Destination{
				{
					Destination:   "10.98.14.179",
					DestinationIP: nilable.From("10.98.14.179"),
				},
			},
		},
		{
			SrcIp:       "193.168.38.211",
			SrcHostname: "thisverypod",
			Destinations: []mapperclient.Destination{
				{
					Destination:   "207.168.35.14",
					DestinationIP: nilable.From("207.168.35.14"),
				},
			},
		},
	}
	slices.SortFunc(results, func(a, b mapperclient.RecordedDestinationsForSrc) bool {
		return a.SrcIp < b.SrcIp
	})
	assert.DeepEqual(s.T(), expectedResults, results, cmpopts.IgnoreTypes(time.Time{}))
}

func TestSocketScannerSuite(t *testing.T) {
	suite.Run(t, new(SocketScannerTestSuite))
}

// Many connections from 10.244.120.89 -> 10.98.14.179:80 (0x50). Only the first one is ESTABLISHED.
// No LISTEN socket, which makes the ESTABLISHED socket client-side.
const mockTcpFileContent = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode                                                     
   0: 5978F40A:89A4 B30E620A:0050 01 00000000:00000000 03:00000ACB 00000000     0        0 0 3 0000000000000000                                      
   1: 5978F40A:BB5A B30E620A:0050 06 00000000:00000000 03:00000547 00000000     0        0 0 3 0000000000000000                                      
   2: 5978F40A:C1D8 B30E620A:0050 06 00000000:00000000 03:00000F86 00000000     0        0 0 3 0000000000000000                                      
   3: 5978F40A:BB58 B30E620A:0050 06 00000000:00000000 03:0000047D 00000000     0        0 0 3 0000000000000000                                      
   4: 5978F40A:C8B8 B30E620A:0050 06 00000000:00000000 03:000011E2 00000000     0        0 0 3 0000000000000000                                      
   5: 5978F40A:C1E4 B30E620A:0050 06 00000000:00000000 03:00001050 00000000     0        0 0 3 0000000000000000                                      
   6: 5978F40A:D0BE B30E620A:0050 06 00000000:00000000 03:000006D9 00000000     0        0 0 3 0000000000000000                                      
   7: 5978F40A:C8D4 B30E620A:0050 06 00000000:00000000 03:0000143F 00000000     0        0 0 3 0000000000000000                                      
   8: 5978F40A:D0B6 B30E620A:0050 06 00000000:00000000 03:00000610 00000000     0        0 0 3 0000000000000000                                      
   9: 5978F40A:C1C6 B30E620A:0050 06 00000000:00000000 03:00000DF3 00000000     0        0 0 3 0000000000000000                                      
  10: 5978F40A:D0D6 B30E620A:0050 06 00000000:00000000 03:0000086D 00000000     0        0 0 3 0000000000000000                                      
  11: 5978F40A:C1EC B30E620A:0050 06 00000000:00000000 03:0000111A 00000000     0        0 0 3 0000000000000000                                      
  12: 5978F40A:C1D2 B30E620A:0050 06 00000000:00000000 03:00000EBC 00000000     0        0 0 3 0000000000000000                                      
  13: 5978F40A:C8DE B30E620A:0050 06 00000000:00000000 03:00001509 00000000     0        0 0 3 0000000000000000                                      
  14: 5978F40A:958A B30E620A:0050 06 00000000:00000000 03:00000156 00000000     0        0 0 3 0000000000000000                                      
  15: 5978F40A:9646 B30E620A:0050 06 00000000:00000000 03:000015D3 00000000     0        0 0 3 0000000000000000                                      
  16: 5978F40A:BB42 B30E620A:0050 06 00000000:00000000 03:000002EA 00000000     0        0 0 3 0000000000000000                                      
  17: 5978F40A:89A0 B30E620A:0050 06 00000000:00000000 03:00000A01 00000000     0        0 0 3 0000000000000000                                      
  18: 5978F40A:89BE B30E620A:0050 06 00000000:00000000 03:00000D29 00000000     0        0 0 3 0000000000000000                                      
  19: 5978F40A:C8C4 B30E620A:0050 06 00000000:00000000 03:00001375 00000000     0        0 0 3 0000000000000000                                      
  20: 5978F40A:D0E2 B30E620A:0050 06 00000000:00000000 03:00000937 00000000     0        0 0 3 0000000000000000                                      
  21: 5978F40A:BB40 B30E620A:0050 06 00000000:00000000 03:00000220 00000000     0        0 0 3 0000000000000000                                      
  22: 5978F40A:C8BC B30E620A:0050 06 00000000:00000000 03:000012AB 00000000     0        0 0 3 0000000000000000                                      
  23: 5978F40A:BB4A B30E620A:0050 06 00000000:00000000 03:000003B4 00000000     0        0 0 3 0000000000000000                                      
  24: 5978F40A:D0CC B30E620A:0050 06 00000000:00000000 03:000007A3 00000000     0        0 0 3 0000000000000000                                      
  25: 5978F40A:957A B30E620A:0050 06 00000000:00000000 03:0000008C 00000000     0        0 0 3 0000000000000000                                      
  26: 5978F40A:9666 B30E620A:0050 06 00000000:00000000 03:00001766 00000000     0        0 0 3 0000000000000000                                      
  27: 5978F40A:9656 B30E620A:0050 06 00000000:00000000 03:0000169C 00000000     0        0 0 3 0000000000000000                                      
  28: 5978F40A:89AE B30E620A:0050 06 00000000:00000000 03:00000C5F 00000000     0        0 0 3 0000000000000000                                      
  29: 5978F40A:956E B30E620A:0050 06 00000000:00000000 03:00000000 00000000     0        0 0 3 0000000000000000                                      
  30: 5978F40A:89AA B30E620A:0050 06 00000000:00000000 03:00000B95 00000000     0        0 0 3 0000000000000000 `

// LISTEN on port 0x1F90 (8080)
// 192.168.35.14 -> 192.168.38.211 ESTABLISHED - should be dropped due to LISTEN, server-side socket
// 176.168.35.14 -> 192.168.38.211 ESTABLISHED - should be dropped due to LISTEN, server-side socket
// 192.168.35.14 -> 193.168.38.211 TIME_WAIT - should be dropped due to TIME_WAIT
// 176.168.35.14 -> 193.168.38.211 TIME_WAIT - should be dropped due to TIME_WAIT
// 193.168.38.211 -> 207.168.35.14 ESTABLISHED - should be returned successfully because client-side socket (no LISTEN)
const mockTcp6FileContent = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000000000000:1F90 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 2448849674 1 0000000000000000 100 0 0 10 0
   1: 0000000000000000FFFF0000D326A8C1:1F90 0000000000000000FFFF00000E23A8C0:CA08 06 00000000:00000000 03:00000A41 00000000     0        0 0 3 0000000000000000
   2: 0000000000000000FFFF0000D326A8C1:1F90 0000000000000000FFFF00000E23A8B0:CA08 06 00000000:00000000 03:00000A41 00000000     0        0 0 3 0000000000000000
   3: 0000000000000000FFFF0000D326A8C0:1F90 0000000000000000FFFF00000E23A8C0:CA08 01 00000000:00000000 03:00000A41 00000000     0        0 0 3 0000000000000000
   4: 0000000000000000FFFF0000D326A8C0:1F90 0000000000000000FFFF00000E23A8B0:CA08 01 00000000:00000000 03:00000A41 00000000     0        0 0 3 0000000000000000
   5: 0000000000000000FFFF0000D326A8C1:D0BE 0000000000000000FFFF00000E23A8CF:0050 01 00000000:00000000 03:00000A41 00000000     0        0 0 3 0000000000000000`
const mockEnvironFileContent = "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\x00HOSTNAME=thisverypod\x00TERM=xterm\x00HOME=/root\x00"
