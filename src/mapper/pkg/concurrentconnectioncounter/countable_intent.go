package concurrentconnectioncounter

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
)

type CountableIntent interface {
	ShouldCountUsingSrcPortMethod() bool
	ShouldCountUsingDNSMethod() bool
}

type CountableIntentIntent struct {
	intent model.Intent
}

func NewCountableIntentIntent(intent model.Intent) *CountableIntentIntent {
	return &CountableIntentIntent{
		intent: intent,
	}
}

func (c *CountableIntentIntent) ShouldCountUsingSrcPortMethod() bool {
	return c.intent.ResolutionData != nil &&
		(*(c.intent.ResolutionData) == SocketScanServiceIntentResolution ||
			*c.intent.ResolutionData == SocketScanPodIntentResolution ||
			*c.intent.ResolutionData == TCPTrafficIntentResolution)
}

func (c *CountableIntentIntent) ShouldCountUsingDNSMethod() bool {
	return c.intent.ResolutionData != nil && *(c.intent.ResolutionData) == DNSTrafficIntentResolution
}

type CountableIntentExternalTrafficIntent struct {
}

func NewCountableIntentExternalTrafficIntent() *CountableIntentExternalTrafficIntent {
	return &CountableIntentExternalTrafficIntent{}
}

func (c *CountableIntentExternalTrafficIntent) ShouldCountUsingSrcPortMethod() bool {
	return false
}

func (c *CountableIntentExternalTrafficIntent) ShouldCountUsingDNSMethod() bool {
	return true
}
