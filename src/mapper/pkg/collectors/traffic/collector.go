package traffic

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"time"
)

type TrafficLevelKey struct {
	Source      serviceidentity.ServiceIdentity
	Destination serviceidentity.ServiceIdentity
}

type TrafficLevelData struct {
	Bytes int
	Flows int
	at    time.Time
}

type TrafficLevelCounter map[TrafficLevelKey][]TrafficLevelData
type TrafficLevelMap map[TrafficLevelKey]TrafficLevelData
type TrafficLevelCallbackFunc func(context.Context, TrafficLevelMap)

type Collector struct {
	trafficLevels TrafficLevelCounter
	callbacks     []TrafficLevelCallbackFunc
}

func NewCollector() *Collector {
	return &Collector{
		trafficLevels: make(TrafficLevelCounter),
	}
}

func (c *Collector) Add(source, destination serviceidentity.ServiceIdentity, bytes, flows int) {
	trafficKey := TrafficLevelKey{
		Source:      source,
		Destination: destination,
	}

	c.trafficLevels[trafficKey] = append(c.trafficLevels[trafficKey], TrafficLevelData{
		Bytes: bytes,
		Flows: flows,
		at:    time.Now(),
	})
}

func (c *Collector) RegisterNotifyTraffic(callback TrafficLevelCallbackFunc) {
	c.callbacks = append(c.callbacks, callback)
}

func (c *Collector) PeriodicUpload(ctx context.Context, interval time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			for _, callback := range c.callbacks {
				callback(ctx, c.getTrafficMap())
			}
		}
	}
}

func (c *Collector) getTrafficMap() TrafficLevelMap {
	trafficLevelMap := make(TrafficLevelMap)

	for k, v := range c.trafficLevels {
		var averageBytes, averageFlows int
		var count int

		for _, data := range v {
			if time.Since(data.at) < time.Hour {
				averageBytes += data.Bytes
				averageFlows += data.Flows
				count++
			} else {
				c.trafficLevels[k] = c.trafficLevels[k][1:]
			}
		}

		if count > 0 {
			trafficLevelMap[k] = TrafficLevelData{
				Bytes: averageBytes / count,
				Flows: averageFlows / count,
			}
		}
	}

	return trafficLevelMap
}
