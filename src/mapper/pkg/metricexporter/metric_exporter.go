package metricexporter

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"

	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/sirupsen/logrus"
)

type MetricExporter struct {
	edgeMetric EdgeMetric
}

func NewMetricExporter(ctx context.Context) (*MetricExporter, error) {
	em, err := NewOtelEdgeMetric(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &MetricExporter{
		edgeMetric: em,
	}, nil
}

func (o *MetricExporter) NotifyIntents(ctx context.Context, intents []intentsstore.TimestampedIntent) {
	for _, intent := range intents {
		clientName := intent.Intent.Client.Name
		serverName := intent.Intent.Server.Name
		logrus.Debugf("recording metric counter: %s -> %s", clientName, serverName)
		o.edgeMetric.Record(ctx, clientName, serverName)
	}
}
