package otelexporter

import (
	"context"
	"time"

	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/shared/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type OtelExporter struct {
	config        Config
	meterProvider *sdk.MeterProvider
	counter       metric.Int64Counter
	intentsHolder *intentsstore.IntentsHolder
}

func newResource() (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.OTelLibraryName("otterize/network-mapper"),
			semconv.OTelLibraryVersion(version.Version()),
		))
}

// uses same name as expected in opentelemetry-collector-contrib's servicegraphprocessor
const CounterMetricName = "traces_service_graph_request_total"
const ClientAttributeName = "client"
const ServerAttributeName = "server"

func newMeterProvider(ctx context.Context, res *resource.Resource, exportInterval time.Duration) (*sdk.MeterProvider, error) {
	// SDK automatically configured via environment variables:
	// - OTEL_EXPORTER_OTLP_ENDPOINT
	// - OTEL_EXPORTER_OTLP_HEADERS
	// - OTEL_EXPORTER_OTLP_TIMEOUT (...)
	metricExporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	if viper.GetBool(sharedconfig.DebugKey) {
		stdOutExporter, err := stdoutmetric.New()
		if err != nil {
			return nil, err
		}
		meterProvider := sdk.NewMeterProvider(
			sdk.WithResource(res),
			sdk.WithReader(sdk.NewPeriodicReader(stdOutExporter,
				sdk.WithInterval(1*time.Second))),
			sdk.WithReader(sdk.NewPeriodicReader(metricExporter,
				sdk.WithInterval(exportInterval))),
		)
		return meterProvider, nil
	}

	meterProvider := sdk.NewMeterProvider(
		sdk.WithResource(res),
		sdk.WithReader(sdk.NewPeriodicReader(metricExporter,
			sdk.WithInterval(exportInterval))),
	)
	return meterProvider, nil
}

func NewOtelExporter(ctx context.Context, ih *intentsstore.IntentsHolder, config Config) (*OtelExporter, error) {
	res, err := newResource()
	if err != nil {
		return nil, err
	}

	meterProvider, err := newMeterProvider(ctx, res, config.ExportInterval)
	if err != nil {
		return nil, err
	}

	var meter = meterProvider.Meter("otelexporter")
	edgeCounter, err := meter.Int64Counter(
		CounterMetricName,
		metric.WithDescription("Count of edges between two nodes"),
	)
	if err != nil {
		return nil, err
	}

	return &OtelExporter{
		intentsHolder: ih,
		config:        config,
		counter:       edgeCounter,
		meterProvider: meterProvider,
	}, nil
}

func (o *OtelExporter) countDiscoveredIntents(ctx context.Context) {
	for _, intent := range o.intentsHolder.GetNewIntentsSinceLastGet() {
		clientName := intent.Intent.Client.Name
		serverName := intent.Intent.Server.Name
		logrus.Debugf("incremeting otel metric counter: %s -> %s", clientName, serverName)
		o.counter.Add(ctx, 1, metric.WithAttributes(attribute.String(ClientAttributeName, clientName), attribute.String(ServerAttributeName, serverName)))
	}
}

func (o *OtelExporter) PeriodicIntentsExport(ctx context.Context) {
	for {
		select {
		case <-time.After(o.config.ExportInterval):
			o.countDiscoveredIntents(ctx)

		case <-ctx.Done():
			o.meterProvider.Shutdown(ctx)
			return
		}
	}
}
