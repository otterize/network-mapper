package metricexporter

import "context"

type EdgeMetric interface {
	Record(ctx context.Context, from string, to string)
}
