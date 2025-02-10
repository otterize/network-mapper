package traffic

import (
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/sirupsen/logrus"
)

type trafficLevelKey struct {
	source      serviceidentity.ServiceIdentity
	destination serviceidentity.ServiceIdentity
}

type trafficLevel struct {
	bytes int
	flows int
}

type Collector struct {
	trafficLevels map[trafficLevelKey]*trafficLevel
}

func NewCollector() *Collector {
	return &Collector{
		trafficLevels: make(map[trafficLevelKey]*trafficLevel),
	}
}

func (c *Collector) Add(source, destination serviceidentity.ServiceIdentity, bytes, flows int) {
	trafficKey := trafficLevelKey{
		source:      source,
		destination: destination,
	}

	value, found := c.trafficLevels[trafficKey]

	if !found {
		value = &trafficLevel{}
		c.trafficLevels[trafficKey] = value
	}

	value.bytes += bytes
	value.flows += flows

	logrus.Infof(
		"Traffic levels: %s - %s: %d bytes, %d flows",
		source.String(),
		destination.String(),
		value.bytes,
		value.flows,
	)
}
