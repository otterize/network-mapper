package concurrentconnectioncounter

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
)

type ConnectionCounterMap[K comparable, Countable CountableIntent] map[K]*ConnectionCounter[Countable]

type ConnectionCountDiffer[K comparable, Countable CountableIntent] struct {
	currentCounters  ConnectionCounterMap[K, Countable]
	previousCounters ConnectionCounterMap[K, Countable]
}

func NewConnectionCountDiffer[K comparable, Countable CountableIntent]() *ConnectionCountDiffer[K, Countable] {
	return &ConnectionCountDiffer[K, Countable]{
		currentCounters:  make(ConnectionCounterMap[K, Countable]),
		previousCounters: make(ConnectionCounterMap[K, Countable]),
	}
}

func (c *ConnectionCountDiffer[K, Countable]) Reset() {
	c.previousCounters = c.currentCounters
	c.currentCounters = make(ConnectionCounterMap[K, Countable])
}

func (c *ConnectionCountDiffer[K, Countable]) Increment(key K, counterInput CounterInput[Countable]) {

	_, existingCounterFound := c.currentCounters[key]
	if !existingCounterFound {
		c.currentCounters[key] = NewConnectionCounter[Countable]()
	}

	c.currentCounters[key].AddConnection(counterInput)
}

func (c *ConnectionCountDiffer[K, Countable]) GetDiff(key K) (cloudclient.ConnectionsCount, bool) {
	currentCounter, hasCurrentValue := c.currentCounters[key]
	prevCounter, hasPrevValue := c.previousCounters[key]

	if hasCurrentValue && !hasPrevValue {
		connectionsCount, isValid := currentCounter.GetConnectionCount()
		if isValid {
			return cloudclient.ConnectionsCount{
				Current: lo.ToPtr(connectionsCount),
				Added:   lo.ToPtr(connectionsCount),
				Removed: lo.ToPtr(0),
			}, true
		}
		return cloudclient.ConnectionsCount{}, false
	}

	if !hasCurrentValue && hasPrevValue {
		connectionsCount, isValid := prevCounter.GetConnectionCount()
		if isValid {
			return cloudclient.ConnectionsCount{
				Current: lo.ToPtr(0),
				Added:   lo.ToPtr(0),
				Removed: lo.ToPtr(connectionsCount),
			}, true
		}
		return cloudclient.ConnectionsCount{}, false
	}

	if hasCurrentValue && hasPrevValue {
		connectionDiff, valid := currentCounter.GetConnectionCountDiff(prevCounter)
		if valid {
			return connectionDiff, true
		}
	}

	return cloudclient.ConnectionsCount{}, false
}
