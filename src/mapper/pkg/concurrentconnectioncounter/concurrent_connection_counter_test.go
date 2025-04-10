package concurrentconnectioncounter

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ConnectionCounterTestSuite struct {
	suite.Suite
}

type TestCase struct {
	Description              string
	SetupInput               []CounterInput[*CountableIntentIntent]
	TestIntent               CounterInput[*CountableIntentIntent]
	ExpectedConnectionsCount int
}

type DiffTestCase struct {
	Description              string
	CurrentSetupInput        []CounterInput[*CountableIntentIntent]
	PrevSetupInput           []CounterInput[*CountableIntentIntent]
	ExpectedConnectionsCount cloudclient.ConnectionsCount
}

func (s *ConnectionCounterTestSuite) TestCounter() {
	testCases := []TestCase{
		{
			Description: "Test only DNS intents",
			SetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
					SourcePorts: make([]int64, 0),
				},
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
					SourcePorts: make([]int64, 0),
				},
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
					SourcePorts: make([]int64, 0),
				},
			},
			TestIntent: CounterInput[*CountableIntentIntent]{
				Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
				SourcePorts: make([]int64, 0),
			},
			ExpectedConnectionsCount: 4,
		},
		{
			Description: "Test only addSocketScanServiceIntent intents",
			SetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(SocketScanServiceIntentResolution)}),
					SourcePorts: []int64{int64(1)},
				},
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(SocketScanServiceIntentResolution)}),
					SourcePorts: []int64{int64(2)},
				},
			},
			TestIntent: CounterInput[*CountableIntentIntent]{
				Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(SocketScanServiceIntentResolution)}),
				SourcePorts: []int64{int64(3)},
			},
			ExpectedConnectionsCount: 3,
		},
		{
			Description: "Test only addSocketScanPodIntent intents",
			SetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(SocketScanPodIntentResolution)}),
					SourcePorts: []int64{int64(1)},
				},
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(SocketScanPodIntentResolution)}),
					SourcePorts: []int64{int64(2)},
				},
			},
			TestIntent: CounterInput[*CountableIntentIntent]{
				Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(SocketScanPodIntentResolution)}),
				SourcePorts: []int64{int64(3)},
			},
			ExpectedConnectionsCount: 3,
		},
		{
			Description: "Test only handleInternalTrafficTCPResult intents",
			SetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(1)},
				},
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(2)},
				},
			},
			TestIntent: CounterInput[*CountableIntentIntent]{
				Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
				SourcePorts: []int64{int64(3)},
			},
			ExpectedConnectionsCount: 3,
		},
		{
			Description: "Test mix socket-scan and tcp-traffic intents",
			SetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(1)},
				},
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(SocketScanPodIntentResolution)}),
					SourcePorts: []int64{int64(2)},
				},
			},
			TestIntent: CounterInput[*CountableIntentIntent]{
				Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(SocketScanServiceIntentResolution)}),
				SourcePorts: []int64{int64(3)},
			},
			ExpectedConnectionsCount: 3,
		},
		{
			Description: "Test mix tcp-traffic intents wins over DNS",
			SetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
					SourcePorts: make([]int64, 0),
				},
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
					SourcePorts: make([]int64, 0),
				},
			},
			TestIntent: CounterInput[*CountableIntentIntent]{
				Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(SocketScanServiceIntentResolution)}),
				SourcePorts: []int64{int64(3)},
			},
			ExpectedConnectionsCount: 1,
		},
		{
			Description: "Test each port is counted only once",
			SetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(1)},
				},
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(1)},
				},
			},
			TestIntent: CounterInput[*CountableIntentIntent]{
				Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
				SourcePorts: []int64{int64(1)},
			},
			ExpectedConnectionsCount: 1,
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.Description, func() {
			counter := NewConnectionCounter[*CountableIntentIntent]()
			for _, input := range testCase.SetupInput {
				counter.AddConnection(input)
			}

			counter.AddConnection(testCase.TestIntent)
			connectionCount, isValid := counter.GetConnectionCount()
			s.True(isValid, "Expected connection count to be valid")
			s.Equal(testCase.ExpectedConnectionsCount, connectionCount)
		})
	}
}

func (s *ConnectionCounterTestSuite) TestCounter_InvalidForTypes() {
	testCases := []TestCase{
		{
			Description: "handleReportKafkaMapperResults not supported",
			SetupInput:  []CounterInput[*CountableIntentIntent]{},
			TestIntent: CounterInput[*CountableIntentIntent]{
				Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr("handleReportKafkaMapperResults")}),
				SourcePorts: make([]int64, 0),
			},
			ExpectedConnectionsCount: 0,
		},
		{
			Description: "handleReportIstioConnectionResults not supported",
			SetupInput:  []CounterInput[*CountableIntentIntent]{},
			TestIntent: CounterInput[*CountableIntentIntent]{
				Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr("handleReportIstioConnectionResults")}),
				SourcePorts: make([]int64, 0),
			},
			ExpectedConnectionsCount: 0,
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.Description, func() {
			counter := NewConnectionCounter[*CountableIntentIntent]()
			counter.AddConnection(testCase.TestIntent)
			_, isValid := counter.GetConnectionCount()
			s.Falsef(isValid, "%s is not a valid intent type", *testCase.TestIntent.Intent.intent.ResolutionData)
		})
	}
}

func (s *ConnectionCounterTestSuite) TestCounter_Diff() {
	testCases := []DiffTestCase{
		{
			Description: "Current count by DNS previous by DNS - all connections are new",
			PrevSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
					SourcePorts: make([]int64, 0),
				},
			},
			CurrentSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
					SourcePorts: make([]int64, 0),
				},
			},
			ExpectedConnectionsCount: cloudclient.ConnectionsCount{
				Current: lo.ToPtr(1),
				Added:   lo.ToPtr(1),
				Removed: lo.ToPtr(1),
			},
		},
		{
			Description: "Current count by TCP previous by TCP - same port, should count as existing connection",
			PrevSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(1)},
				},
			},
			CurrentSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(1)},
				},
			},
			ExpectedConnectionsCount: cloudclient.ConnectionsCount{
				Current: lo.ToPtr(1),
				Added:   lo.ToPtr(0),
				Removed: lo.ToPtr(0),
			},
		},
		{
			Description: "Current count by TCP previous by TCP - different port, should count as new connection",
			PrevSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(10)},
				},
			},
			CurrentSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(1)},
				},
			},
			ExpectedConnectionsCount: cloudclient.ConnectionsCount{
				Current: lo.ToPtr(1),
				Added:   lo.ToPtr(1),
				Removed: lo.ToPtr(1),
			},
		},
		{
			Description: "Current count by TCP previous by TCP - mixed of same and different port, should be smart diff",
			PrevSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(1), int64(10), int64(100)},
				},
			},
			CurrentSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(100), int64(200)},
				},
			},
			ExpectedConnectionsCount: cloudclient.ConnectionsCount{
				Current: lo.ToPtr(2),
				Added:   lo.ToPtr(1),
				Removed: lo.ToPtr(2),
			},
		},
		{
			Description: "Current count by TCP previous by DNS - all connections are new",
			PrevSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
					SourcePorts: make([]int64, 0),
				},
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
					SourcePorts: make([]int64, 0),
				},
			},
			CurrentSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(1)},
				},
			},
			ExpectedConnectionsCount: cloudclient.ConnectionsCount{
				Current: lo.ToPtr(1),
				Added:   lo.ToPtr(1),
				Removed: lo.ToPtr(2),
			},
		},
		{
			Description: "Current count by DNS previous by TCP - all connections are new",
			PrevSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(TCPTrafficIntentResolution)}),
					SourcePorts: []int64{int64(1)},
				},
			},
			CurrentSetupInput: []CounterInput[*CountableIntentIntent]{
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
					SourcePorts: make([]int64, 0),
				},
				{
					Intent:      NewCountableIntentIntent(model.Intent{ResolutionData: lo.ToPtr(DNSTrafficIntentResolution)}),
					SourcePorts: make([]int64, 0),
				},
			},
			ExpectedConnectionsCount: cloudclient.ConnectionsCount{
				Current: lo.ToPtr(2),
				Added:   lo.ToPtr(2),
				Removed: lo.ToPtr(1),
			},
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.Description, func() {
			prevCounter := NewConnectionCounter[*CountableIntentIntent]()
			lo.ForEach(testCase.PrevSetupInput, func(input CounterInput[*CountableIntentIntent], _ int) {
				prevCounter.AddConnection(input)
			})

			currentCounter := NewConnectionCounter[*CountableIntentIntent]()
			lo.ForEach(testCase.CurrentSetupInput, func(input CounterInput[*CountableIntentIntent], _ int) {
				currentCounter.AddConnection(input)
			})

			diff, isValid := currentCounter.GetConnectionCountDiff(prevCounter)
			s.True(isValid, "Expected connection count diff to be valid")
			s.Equal(testCase.ExpectedConnectionsCount, diff)
		})
	}
}

func (s *ConnectionCounterTestSuite) TestCounter_Diff_ExternalTraffic() {
	prevCounter := NewConnectionCounter[*CountableIntentExternalTrafficIntent]()
	prevCounter.AddConnection(CounterInput[*CountableIntentExternalTrafficIntent]{
		Intent: NewCountableIntentExternalTrafficIntent(), SourcePorts: make([]int64, 0),
	})
	prevCounter.AddConnection(CounterInput[*CountableIntentExternalTrafficIntent]{
		Intent: NewCountableIntentExternalTrafficIntent(), SourcePorts: make([]int64, 0),
	})
	prevCounter.AddConnection(CounterInput[*CountableIntentExternalTrafficIntent]{
		Intent: NewCountableIntentExternalTrafficIntent(), SourcePorts: make([]int64, 0),
	})

	currentCounter := NewConnectionCounter[*CountableIntentExternalTrafficIntent]()
	currentCounter.AddConnection(CounterInput[*CountableIntentExternalTrafficIntent]{
		Intent: NewCountableIntentExternalTrafficIntent(), SourcePorts: make([]int64, 0),
	})

	diff, isValid := currentCounter.GetConnectionCountDiff(prevCounter)
	s.True(isValid, "Expected connection count diff to be valid")
	expected := cloudclient.ConnectionsCount{
		Current: lo.ToPtr(1),
		Added:   lo.ToPtr(1),
		Removed: lo.ToPtr(3),
	}
	s.Equal(expected, diff)

}

func TestConnectionCounterSuite(t *testing.T) {
	suite.Run(t, new(ConnectionCounterTestSuite))
}
