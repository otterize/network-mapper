package socketscanner

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	mock_client "github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient/mockclient"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type SocketScannerTestSuite struct {
	suite.Suite
	mockController   *gomock.Controller
	mockMapperClient *mock_client.MockMapperClient
}

func (s *SocketScannerTestSuite) SetupSuite() {
	s.mockController = gomock.NewController(s.T())
	s.mockMapperClient = mock_client.NewMockMapperClient(s.mockController)
}

type SocketScanResultForSrcIpMatcher []mapperclient.SocketScanResultForSrcIp

func (m SocketScanResultForSrcIpMatcher) Matches(x interface{}) bool {
	results, ok := x.(mapperclient.SocketScanResults)
	if !ok {
		return false
	}

	actualValues := results.Results
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

func matchSocketScanResult(expected, actual mapperclient.SocketScanResultForSrcIp) bool {
	if expected.SrcIp != actual.SrcIp {
		return false
	}

	if len(expected.DestIps) != len(actual.DestIps) {
		return false
	}

	for i := range expected.DestIps {
		if expected.DestIps[i].Destination != actual.DestIps[i].Destination {
			return false
		}
	}

	return true
}

func (m SocketScanResultForSrcIpMatcher) String() string {
	var result string
	for _, value := range m {
		result += fmt.Sprintf("{Src: %v, Dest: %v}", value.SrcIp, value.DestIps)
	}
	return result
}

func GetMatcher(expected []mapperclient.SocketScanResultForSrcIp) SocketScanResultForSrcIpMatcher {
	return expected
}

func (s *SocketScannerTestSuite) TestScanProcDir() {
	mockProcDir, err := os.MkdirTemp("", "testscamprocdir")
	s.Require().NoError(err)
	defer os.RemoveAll(mockProcDir)

	s.Require().NoError(os.MkdirAll(mockProcDir+"/100/net", 0o700))
	s.Require().NoError(os.WriteFile(mockProcDir+"/100/net/tcp", []byte(mockTcpFileContent), 0o444))
	s.Require().NoError(os.WriteFile(mockProcDir+"/100/net/tcp6", []byte(mockTcp6FileContent), 0o444))
	viper.Set(config.HostProcDirKey, mockProcDir)

	sniffer := NewSocketScanner(s.mockMapperClient)
	s.Require().NoError(sniffer.ScanProcDir())

	// We should only see sockets that this pod serves to other clients.
	// all other sockets should be ignored (because parsing the server sides on all pods is enough)
	expectedResult := []mapperclient.SocketScanResultForSrcIp{
		{
			SrcIp: "192.168.35.14",
			DestIps: []mapperclient.Destination{
				{
					Destination: "192.168.38.211",
				},
			},
		},
		{
			SrcIp: "176.168.35.14",
			DestIps: []mapperclient.Destination{
				{
					Destination: "192.168.38.211",
				},
			},
		},
	}

	s.mockMapperClient.EXPECT().ReportSocketScanResults(gomock.Any(), GetMatcher(expectedResult))
	err = sniffer.ReportSocketScanResults(context.Background())
	s.Require().NoError(err)
}

func TestRunIntegrationsResolversSuite(t *testing.T) {
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
