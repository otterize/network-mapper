package socketscanner

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/otterize/otternose/sniffer/pkg/client"
	mock_client "github.com/otterize/otternose/sniffer/pkg/client/mockclient"
	"github.com/otterize/otternose/sniffer/pkg/config"
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

	s.mockMapperClient.EXPECT().ReportSocketScanResults(gomock.Any(), client.SocketScanResults{
		Results: []client.SocketScanResultForSrcIp{
			// We should only see sockets that this pod serves to other clients.
			// all other sockets should be ignored (because parsing the server sides on all pods is enough)
			{
				SrcIp:   "192.168.35.14",
				DestIps: []string{"192.168.38.211"},
			},
			{
				SrcIp:   "176.168.35.14",
				DestIps: []string{"192.168.38.211"},
			},
		},
	})
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
