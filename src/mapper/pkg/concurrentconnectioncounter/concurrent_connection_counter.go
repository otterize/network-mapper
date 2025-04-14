package concurrentconnectioncounter

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
	"sync"
)

type SourcePortsSet map[int64]struct{}

type CountMethod int

const (
	CountMethodUnset      CountMethod = 0
	CountMethodDNS        CountMethod = 1
	CountMethodSourcePort CountMethod = 2
)

type CounterInput[T CountableIntent] struct {
	Intent      T
	SourcePorts []int64
}

type ConnectionCounter[T CountableIntent] struct {
	SourcePorts SourcePortsSet
	DNSCounter  int
	countMethod CountMethod
	lock        sync.Mutex
}

func NewConnectionCounter[T CountableIntent]() *ConnectionCounter[T] {
	return &ConnectionCounter[T]{
		SourcePorts: make(SourcePortsSet),
		DNSCounter:  0,
		countMethod: CountMethodUnset,
		lock:        sync.Mutex{},
	}
}

func (c *ConnectionCounter[T]) AddConnection(input CounterInput[T]) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if input.Intent.ShouldCountUsingSrcPortMethod() {
		// TCP source port connections wins over DNS (in terms of connections count)
		c.countMethod = CountMethodSourcePort
		lo.ForEach(input.SourcePorts, func(port int64, _ int) {
			c.SourcePorts[port] = struct{}{}
		})
		return
	}

	if (c.countMethod == CountMethodUnset || c.countMethod == CountMethodDNS) && input.Intent.ShouldCountUsingDNSMethod() {
		c.countMethod = CountMethodDNS
		c.DNSCounter++
		return
	}

	// otherwise we do not count this intent. Either because it is a DNS intent and we are already counting by source ports,
	// or because it is an unknown intent type
}

func (c *ConnectionCounter[T]) GetConnectionCount() (int, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.getConnectionCountUnsafe()
}

func (c *ConnectionCounter[T]) getConnectionCountUnsafe() (int, bool) {
	if c.countMethod == CountMethodSourcePort {
		return len(c.SourcePorts), true
	}

	if c.countMethod == CountMethodDNS {
		return c.DNSCounter, true
	}

	return 0, false
}

func (c *ConnectionCounter[T]) GetConnectionCountDiff(other *ConnectionCounter[T]) (cloudclient.ConnectionsCount, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.countMethod == CountMethodUnset || other.countMethod == CountMethodUnset {
		return cloudclient.ConnectionsCount{}, false
	}

	// Note that we call the usafe version since we already locked the lock ar the function entrance, wnad mutex lock are
	// not reentrant in GO.
	currentCount, _ := c.getConnectionCountUnsafe()
	otherCount, _ := other.getConnectionCountUnsafe()

	if c.countMethod != other.countMethod {
		return cloudclient.ConnectionsCount{
			Current: lo.ToPtr(currentCount),
			Added:   lo.ToPtr(currentCount),
			Removed: lo.ToPtr(otherCount),
		}, true
	}

	// For here onwards we can assume that both counters use the same counting method
	if c.countMethod == CountMethodDNS {
		return cloudclient.ConnectionsCount{
			Current: lo.ToPtr(currentCount),
			Added:   lo.ToPtr(currentCount),
			Removed: lo.ToPtr(otherCount),
		}, true
	}

	var missingFromSelfCount, missingFromOtherCount int

	for key := range c.SourcePorts {
		if _, ok := other.SourcePorts[key]; !ok {
			missingFromOtherCount += 1
		}
	}

	for key := range other.SourcePorts {
		if _, ok := c.SourcePorts[key]; !ok {
			missingFromSelfCount += 1
		}
	}

	return cloudclient.ConnectionsCount{
		Current: lo.ToPtr(len(c.SourcePorts)),
		Added:   lo.ToPtr(missingFromOtherCount),
		Removed: lo.ToPtr(missingFromSelfCount),
	}, true

}
