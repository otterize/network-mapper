package ipresolver

import (
	"github.com/otterize/network-mapper/src/sniffer/pkg/utils"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"
	"testing"
)

type ProcessMonitorTestSuite struct {
	suite.Suite
	processMonitor *ProcessMonitor
	pidCalledNew   []int64
	pidCalledExit  []int64
	currenPids     []int64
}

func (s *ProcessMonitorTestSuite) SetupTest() {
	s.processMonitor = NewProcessMonitor(s.onNew, s.onExit, s.scanPids)
}

func (s *ProcessMonitorTestSuite) onNew(pid int64, _ string) error {
	s.pidCalledNew = append(s.pidCalledNew, pid)
	return nil
}

func (s *ProcessMonitorTestSuite) onExit(pid int64, _ string) error {
	s.pidCalledExit = append(s.pidCalledExit, pid)
	return nil
}

func (s *ProcessMonitorTestSuite) scanPids(callback utils.ProcessScanCallback) error {
	for _, pid := range s.currenPids {
		callback(pid, "testDir")
	}
	return nil
}

func (s *ProcessMonitorTestSuite) TestNewProcess() {
	s.resetMockPid()

	s.currenPids = []int64{10, 20, 30}
	err := s.processMonitor.Poll()
	s.NoError(err)
	slices.Sort(s.pidCalledNew)
	slices.Sort(s.pidCalledExit)

	s.Require().Equal(s.currenPids, s.pidCalledNew)
	s.Require().Empty(s.pidCalledExit)
	s.resetMockPid()

	s.currenPids = []int64{10, 20, 30, 40}
	err = s.processMonitor.Poll()
	s.NoError(err)
	s.Require().Equal([]int64{40}, s.pidCalledNew)
	s.Require().Empty(s.pidCalledExit)
	s.resetMockPid()

	s.currenPids = []int64{40}
	err = s.processMonitor.Poll()
	s.NoError(err)
	s.Require().Empty(s.pidCalledNew)
	slices.Sort(s.pidCalledExit)
	s.Require().Equal([]int64{10, 20, 30}, s.pidCalledExit)
	s.resetMockPid()

	s.currenPids = []int64{40}
	err = s.processMonitor.Poll()
	s.NoError(err)
	s.Require().Empty(s.pidCalledNew)
	s.Require().Empty(s.pidCalledExit)
}

func (s *ProcessMonitorTestSuite) resetMockPid() {
	s.pidCalledNew = []int64{}
	s.pidCalledExit = []int64{}
}

func TestProcessMonitorTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessMonitorTestSuite))
}
