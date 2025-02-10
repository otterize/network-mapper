package traffic

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
)

type trafficLevelKey struct {
	sourceIP      types.NamespacedName
	destinationIP types.NamespacedName
}

type trafficLevel struct {
	bytes int
	flows int
}

type Collector struct {
	trafficLevels map[trafficLevelKey]*trafficLevel
}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) Add(sourceIP, destinationIP string, bytes, flows int) {
	//trafficKey := trafficLevelKey{
	//	source:      source.AsNamespacedName(),
	//	destination: destination.AsNamespacedName(),
	//}
	//
	//value, found := c.trafficLevels[trafficKey]
	//
	//if !found {
	//	value = &trafficLevel{}
	//	c.trafficLevels[trafficKey] = value
	//}
	//
	//value.bytes += bytes
	//value.flows += flows

	logrus.Warnf("Traffic level: %+v", 0)
}
