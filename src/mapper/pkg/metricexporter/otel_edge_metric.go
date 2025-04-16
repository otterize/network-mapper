package metricexporter

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"time"

	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type OtelEdgeMetric struct {
	meterProvider metric.MeterProvider
	counter       metric.Int64Counter
}

func newResource() (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.OTelScopeName("otterize/network-mapper"),
		))
}

const ClientAttributeName = "client"
const ServerAttributeName = "server"

func newMeterProvider(ctx context.Context, res *resource.Resource) (*sdk.MeterProvider, error) {
	// SDK automatically configured via environment variables:
	// - OTEL_EXPORTER_OTLP_ENDPOINT
	// - OTEL_EXPORTER_OTLP_HEADERS
	// - OTEL_EXPORTER_OTLP_TIMEOUT (...)
	metricExporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if viper.GetBool(sharedconfig.DebugKey) {
		stdOutExporter, err := stdoutmetric.New()
		if err != nil {
			return nil, errors.Wrap(err)
		}
		meterProvider := sdk.NewMeterProvider(
			sdk.WithResource(res),
			sdk.WithReader(sdk.NewPeriodicReader(stdOutExporter,
				sdk.WithInterval(1*time.Second))),
			sdk.WithReader(sdk.NewPeriodicReader(metricExporter)),
		)
		return meterProvider, nil
	}

	meterProvider := sdk.NewMeterProvider(
		sdk.WithResource(res),
		sdk.WithReader(sdk.NewPeriodicReader(metricExporter)),
	)
	return meterProvider, nil
}

func (o *OtelEdgeMetric) Record(ctx context.Context, from string, to string) {
	o.counter.Add(ctx, 1, metric.WithAttributes(attribute.String(ClientAttributeName, from), attribute.String(ServerAttributeName, to)))
}

func NewOtelEdgeMetric(ctx context.Context) (*OtelEdgeMetric, error) {
	res, err := newResource()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	meterProvider, err := newMeterProvider(ctx, res)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	var meter = meterProvider.Meter("otelexporter")
	edgeCounter, err := meter.Int64Counter(
		viper.GetString(config.OTelMetricKey),
		metric.WithDescription("Count of edges between two nodes"),
	)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &OtelEdgeMetric{
		counter:       edgeCounter,
		meterProvider: meterProvider,
	}, nil
}
