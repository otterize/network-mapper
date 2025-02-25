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
		var sumBytes, sumFlows int
		var count int

		for _, data := range v {
			if time.Since(data.at) < time.Hour {
				// count only data within the last hour
				sumBytes += data.Bytes
				sumFlows += data.Flows
				count++
			} else {
				// drop data older than an hour
				c.trafficLevels[k] = c.trafficLevels[k][1:]
			}
		}

		if count > 0 {
			trafficLevelMap[k] = TrafficLevelData{
				Bytes: sumBytes / count,
				Flows: sumFlows / count,
			}
		}
	}

	return trafficLevelMap
}
