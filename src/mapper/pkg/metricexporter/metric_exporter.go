package metricexporter

import (
	"context"

	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/sirupsen/logrus"
)

type MetricExporter struct {
	config     Config
	edgeMetric EdgeMetric
}

func NewMetricExporter(ctx context.Context, config Config) (*MetricExporter, error) {
	em, err := NewOtelEdgeMetric(ctx, config.ExportInterval)
	if err != nil {
		return nil, err
	}

	return &MetricExporter{
		config:     config,
		edgeMetric: em,
	}, nil
}

func (o *MetricExporter) GetIntentCallback(ctx context.Context, intents []intentsstore.TimestampedIntent) {
	for _, intent := range intents {
		clientName := intent.Intent.Client.Name
		serverName := intent.Intent.Server.Name
		logrus.Debugf("recording metric counter: %s -> %s", clientName, serverName)
		o.edgeMetric.Record(ctx, clientName, serverName)
	}
}
