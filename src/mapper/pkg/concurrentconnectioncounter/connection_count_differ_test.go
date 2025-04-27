package concurrentconnectioncounter

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type ConnectionCountDifferSuite struct {
	suite.Suite
}

type CountableIntentTCPDummy struct{}
type CountableIntentDNSDummy struct{}

func NewCountableIntentDummy() *CountableIntentTCPDummy {
	return &CountableIntentTCPDummy{}
}

func (c *CountableIntentTCPDummy) ShouldCountUsingSrcPortMethod() bool {
	return true
}

func (c *CountableIntentTCPDummy) ShouldCountUsingDNSMethod() bool {
	return false
}

func NewCountableIntentDNSDummy() *CountableIntentDNSDummy {
	return &CountableIntentDNSDummy{}
}

func (c *CountableIntentDNSDummy) ShouldCountUsingSrcPortMethod() bool {
	return false
}

func (c *CountableIntentDNSDummy) ShouldCountUsingDNSMethod() bool {
	return true
}

func (s *ConnectionCountDifferSuite) TestTCPDiff_TestNoPrevValue() {
	differ := NewConnectionCountDiffer[string, *CountableIntentTCPDummy]()

	// Add a connections
	differ.Increment("key1", CounterInput[*CountableIntentTCPDummy]{
		Intent:      NewCountableIntentDummy(),
		SourcePorts: []int64{1, 2},
	})
	differ.Increment("key1", CounterInput[*CountableIntentTCPDummy]{
		Intent:      NewCountableIntentDummy(),
		SourcePorts: []int64{2, 3},
	})

	// Get the diff
	diff, ok := differ.GetDiff("key1")

	s.Require().True(ok)
	s.Require().Equal(3, *diff.Current)
	s.Require().Equal(3, *diff.Added)
	s.Require().Equal(0, *diff.Removed)
}

func (s *ConnectionCountDifferSuite) TestTCPDiff_TestPrevConnectionsAreTheSameAsCurrent() {
	differ := NewConnectionCountDiffer[string, *CountableIntentTCPDummy]()

	// Add a connections
	differ.Increment("key1", CounterInput[*CountableIntentTCPDummy]{
		Intent:      NewCountableIntentDummy(),
		SourcePorts: []int64{1, 2, 3},
	})

	differ.Reset()

	// Add same connections
	differ.Increment("key1", CounterInput[*CountableIntentTCPDummy]{
		Intent:      NewCountableIntentDummy(),
		SourcePorts: []int64{1, 2, 3},
	})

	// Get the diff
	diff, ok := differ.GetDiff("key1")

	s.Require().True(ok)
	s.Require().Equal(3, *diff.Current)
	s.Require().Equal(0, *diff.Added)
	s.Require().Equal(0, *diff.Removed)
}

func (s *ConnectionCountDifferSuite) TestDNSDiff_TestNoPrevValue() {
	differ := NewConnectionCountDiffer[string, *CountableIntentDNSDummy]()

	// Add a connections
	differ.Increment("key1", CounterInput[*CountableIntentDNSDummy]{
		Intent:      NewCountableIntentDNSDummy(),
		SourcePorts: []int64{1, 2},
	})
	differ.Increment("key1", CounterInput[*CountableIntentDNSDummy]{
		Intent:      NewCountableIntentDNSDummy(),
		SourcePorts: []int64{2, 3},
	})

	// Get the diff
	diff, ok := differ.GetDiff("key1")

	s.Require().True(ok)
	s.Require().Equal(2, *diff.Current)
	s.Require().Equal(2, *diff.Added)
	s.Require().Equal(0, *diff.Removed)
}

func (s *ConnectionCountDifferSuite) TestDNSDiff_TestWithPrevValue() {
	differ := NewConnectionCountDiffer[string, *CountableIntentDNSDummy]()

	// Add a connections
	differ.Increment("key1", CounterInput[*CountableIntentDNSDummy]{
		Intent:      NewCountableIntentDNSDummy(),
		SourcePorts: []int64{1, 2},
	})
	differ.Increment("key1", CounterInput[*CountableIntentDNSDummy]{
		Intent:      NewCountableIntentDNSDummy(),
		SourcePorts: []int64{2, 3},
	})

	differ.Reset()

	differ.Increment("key1", CounterInput[*CountableIntentDNSDummy]{
		Intent:      NewCountableIntentDNSDummy(),
		SourcePorts: make([]int64, 0),
	})

	// Get the diff
	diff, ok := differ.GetDiff("key1")

	s.Require().True(ok)
	s.Require().Equal(1, *diff.Current)
	s.Require().Equal(1, *diff.Added)
	s.Require().Equal(2, *diff.Removed)
}

func TestConnectionCountDifferSuite(t *testing.T) {
	suite.Run(t, new(ConnectionCountDifferSuite))
}
