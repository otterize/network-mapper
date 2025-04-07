package intentsstore

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
)

type SourcePortsSet map[int64]struct{}

type CountMethod int

const (
	CountMethodUnset      CountMethod = 0
	CountMethodDNS        CountMethod = 1
	CountMethodSourcePort CountMethod = 2
)

type CounterInput struct {
	Intent      model.Intent
	SourcePorts []int64
}

type ConnectionCounter struct {
	SourcePorts SourcePortsSet
	DNSCounter  int
	countMethod CountMethod
}

func NewConnectionCounter() *ConnectionCounter {
	return &ConnectionCounter{
		SourcePorts: make(SourcePortsSet),
		DNSCounter:  0,
		countMethod: CountMethodUnset,
	}
}

func (c *ConnectionCounter) AddConnection(input CounterInput) {
	if c.shouldHandleIntentAsSrcPortCount(input.Intent) {
		// TCP source port connections wins over DNS (in terms of connections count)
		c.countMethod = CountMethodSourcePort
		lo.ForEach(input.SourcePorts, func(port int64, _ int) {
			c.SourcePorts[port] = struct{}{}
		})
		return
	}

	if (c.countMethod == CountMethodUnset || c.countMethod == CountMethodDNS) && c.shouldHandleIntentAsDNSCount(input.Intent) {
		c.countMethod = CountMethodDNS
		c.DNSCounter++
		return
	}

	// otherwise we do not count this intent. Either because it is a DNS intent and we are already counting by source ports,
	// or because it is an unknown intent type
}

func (c *ConnectionCounter) GetConnectionCount() (int, bool) {
	if c.countMethod == CountMethodSourcePort {
		return len(c.SourcePorts), true
	}

	if c.countMethod == CountMethodDNS {
		return c.DNSCounter, true
	}

	return 0, false
}

func (c *ConnectionCounter) shouldHandleIntentAsDNSCount(intent model.Intent) bool {
	return intent.ResolutionData != nil && *(intent.ResolutionData) == DNSTrafficIntentResolution
}

func (c *ConnectionCounter) shouldHandleIntentAsSrcPortCount(intent model.Intent) bool {
	return intent.ResolutionData != nil &&
		(*(intent.ResolutionData) == SocketScanServiceIntentResolution ||
			*intent.ResolutionData == SocketScanPodIntentResolution ||
			*intent.ResolutionData == TCPTrafficIntentResolution)
}

func (c *ConnectionCounter) GetConnectionCountDiff(other *ConnectionCounter) (cloudclient.ConnectionsCount, bool) {
	if c.countMethod == CountMethodUnset || other.countMethod == CountMethodUnset {
		return cloudclient.ConnectionsCount{}, false
	}

	currentCount, _ := c.GetConnectionCount()
	otherCount, _ := other.GetConnectionCount()

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
