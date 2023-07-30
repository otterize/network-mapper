package collectors

import (
	"fmt"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
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
	//expectedResult := GetMatcher([]mapperclient.RecordedDestinationsForSrc{
	//	{
	//		SrcIp:       "192.168.35.14",
	//		SrcHostname: "thisverypod",
	//		Destinations: []mapperclient.Destination{
	//			{
	//				Destination: "192.168.38.211",
	//			},
	//		},
	//	},
	//	{
	//		SrcIp:       "176.168.35.14",
	//		SrcHostname: "thisverypod",
	//		Destinations: []mapperclient.Destination{
	//			{
	//				Destination: "192.168.38.211",
	//			},
	//		},
	//	},
	//})
	results := scanner.CollectResults()
	s.Require().ElementsMatch([]mapperclient.RecordedDestinationsForSrc{
		{
			SrcIp:       "192.168.35.14",
			SrcHostname: "",
			Destinations: []mapperclient.Destination{
				{
					Destination: "192.168.38.211",
				},
			},
		},
		{
			SrcIp:       "176.168.35.14",
			SrcHostname: "",
			Destinations: []mapperclient.Destination{
				{
					Destination: "192.168.38.211",
				},
			},
		},
	}, results)
}

func TestSocketScannerSuite(t *testing.T) {
	suite.Run(t, new(SocketScannerTestSuite))
}

const mockTcpFileContent = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: D326A8C0:EC8E 7EB3640A:1B9E 01 00000000:00000000 02:0000058E 00000000     0        0 2688924237 2 0000000000000000 20 4 24 10 -1
   1: D326A8C0:89F6 0EB8640A:C383 01 00000000:00000000 02:000002DB 00000000     0        0 2448859443 2 0000000000000000 20 4 1 4 -1
   2: D326A8C0:ADEE 20CB640A:2553 01 00000000:00000000 02:000003A7 00000000     0        0 2448899655 2 0000000000000000 20 4 1 10 -1`
const mockTcp6FileContent = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000000000000:1F90 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 2448849674 1 0000000000000000 100 0 0 10 0
   1: 0000000000000000FFFF0000D326A8C0:1F90 0000000000000000FFFF00000E23A8C0:CA08 06 00000000:00000000 03:00000A41 00000000     0        0 0 3 0000000000000000
   2: 0000000000000000FFFF0000D326A8C0:1F90 0000000000000000FFFF00000E23A8B0:CA08 06 00000000:00000000 03:00000A41 00000000     0        0 0 3 0000000000000000`
const mockEnvironFileContent = "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\x00HOSTNAME=thisverypod\x00TERM=xterm\x00HOME=/root\x00"
