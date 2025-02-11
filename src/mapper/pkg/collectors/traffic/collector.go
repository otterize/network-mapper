package traffic

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
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
	client        cloudclient.CloudClient
	trafficLevels map[trafficLevelKey]*trafficLevel
}

func NewCollector(client cloudclient.CloudClient) *Collector {
	return &Collector{
		client:        client,
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

	err := c.client.ReportTrafficLevels(
		context.TODO(),
		source.Name,
		source.Namespace,
		destination.Name,
		destination.Namespace,
		cloudclient.TrafficLevelInput{
			Data:  value.bytes,
			Flows: value.flows,
		},
	)
	if err != nil {
		logrus.WithError(err).Error("Failed to update traffic info")
	}
}
