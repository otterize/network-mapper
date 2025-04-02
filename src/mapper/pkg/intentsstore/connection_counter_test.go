package intentsstore

import (
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
	SetupInput               []CounterInput
	TestIntent               CounterInput
	ExpectedConnectionsCount int
}

func (s *ConnectionCounterTestSuite) TestCounter() {
	testCases := []TestCase{
		{
			Description: "Test only DNS intents",
			SetupInput: []CounterInput{
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("handleDNSCaptureResultsAsKubernetesPods")},
					SourcePorts: make([]int64, 0),
				},
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("handleDNSCaptureResultsAsKubernetesPods")},
					SourcePorts: make([]int64, 0),
				},
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("handleDNSCaptureResultsAsKubernetesPods")},
					SourcePorts: make([]int64, 0),
				},
			},
			TestIntent: CounterInput{
				Intent:      model.Intent{ResolutionData: lo.ToPtr("handleDNSCaptureResultsAsKubernetesPods")},
				SourcePorts: make([]int64, 0),
			},
			ExpectedConnectionsCount: 4,
		},
		{
			Description: "Test only addSocketScanServiceIntent intents",
			SetupInput: []CounterInput{
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("addSocketScanServiceIntent")},
					SourcePorts: []int64{int64(1)},
				},
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("addSocketScanServiceIntent")},
					SourcePorts: []int64{int64(2)},
				},
			},
			TestIntent: CounterInput{
				Intent:      model.Intent{ResolutionData: lo.ToPtr("addSocketScanServiceIntent")},
				SourcePorts: []int64{int64(3)},
			},
			ExpectedConnectionsCount: 3,
		},
		{
			Description: "Test only addSocketScanPodIntent intents",
			SetupInput: []CounterInput{
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("addSocketScanPodIntent")},
					SourcePorts: []int64{int64(1)},
				},
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("addSocketScanPodIntent")},
					SourcePorts: []int64{int64(2)},
				},
			},
			TestIntent: CounterInput{
				Intent:      model.Intent{ResolutionData: lo.ToPtr("addSocketScanPodIntent")},
				SourcePorts: []int64{int64(3)},
			},
			ExpectedConnectionsCount: 3,
		},
		{
			Description: "Test only handleInternalTrafficTCPResult intents",
			SetupInput: []CounterInput{
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("handleInternalTrafficTCPResult")},
					SourcePorts: []int64{int64(1)},
				},
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("handleInternalTrafficTCPResult")},
					SourcePorts: []int64{int64(2)},
				},
			},
			TestIntent: CounterInput{
				Intent:      model.Intent{ResolutionData: lo.ToPtr("handleInternalTrafficTCPResult")},
				SourcePorts: []int64{int64(3)},
			},
			ExpectedConnectionsCount: 3,
		},
		{
			Description: "Test mix socket-scan and tcp-traffic intents",
			SetupInput: []CounterInput{
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("handleInternalTrafficTCPResult")},
					SourcePorts: []int64{int64(1)},
				},
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("addSocketScanPodIntent")},
					SourcePorts: []int64{int64(2)},
				},
			},
			TestIntent: CounterInput{
				Intent:      model.Intent{ResolutionData: lo.ToPtr("addSocketScanServiceIntent")},
				SourcePorts: []int64{int64(3)},
			},
			ExpectedConnectionsCount: 3,
		},
		{
			Description: "Test mix tcp-traffic intents wins over DNS",
			SetupInput: []CounterInput{
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("handleDNSCaptureResultsAsKubernetesPods")},
					SourcePorts: make([]int64, 0),
				},
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("handleDNSCaptureResultsAsKubernetesPods")},
					SourcePorts: make([]int64, 0),
				},
			},
			TestIntent: CounterInput{
				Intent:      model.Intent{ResolutionData: lo.ToPtr("addSocketScanServiceIntent")},
				SourcePorts: []int64{int64(3)},
			},
			ExpectedConnectionsCount: 1,
		},
		{
			Description: "Test each port is counted only once",
			SetupInput: []CounterInput{
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("handleInternalTrafficTCPResult")},
					SourcePorts: []int64{int64(1)},
				},
				{
					Intent:      model.Intent{ResolutionData: lo.ToPtr("handleInternalTrafficTCPResult")},
					SourcePorts: []int64{int64(1)},
				},
			},
			TestIntent: CounterInput{
				Intent:      model.Intent{ResolutionData: lo.ToPtr("handleInternalTrafficTCPResult")},
				SourcePorts: []int64{int64(1)},
			},
			ExpectedConnectionsCount: 1,
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.Description, func() {
			counter := NewConnectionCounter()
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
			SetupInput:  []CounterInput{},
			TestIntent: CounterInput{
				Intent:      model.Intent{ResolutionData: lo.ToPtr("handleReportKafkaMapperResults")},
				SourcePorts: make([]int64, 0),
			},
			ExpectedConnectionsCount: 0,
		},
		{
			Description: "handleReportIstioConnectionResults not supported",
			SetupInput:  []CounterInput{},
			TestIntent: CounterInput{
				Intent:      model.Intent{ResolutionData: lo.ToPtr("handleReportIstioConnectionResults")},
				SourcePorts: make([]int64, 0),
			},
			ExpectedConnectionsCount: 0,
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.Description, func() {
			counter := NewConnectionCounter()
			counter.AddConnection(testCase.TestIntent)
			_, isValid := counter.GetConnectionCount()
			s.Falsef(isValid, "%s is not a valid intent type", *testCase.TestIntent.Intent.ResolutionData)
		})
	}
}

func TestConnectionCounterSuite(t *testing.T) {
	suite.Run(t, new(ConnectionCounterTestSuite))
}
