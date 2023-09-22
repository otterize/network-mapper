package otelexporter

import (
	"context"
	"os"
	"time"

	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/otterize/network-mapper/src/shared/version"
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
	counter       metric.Int64Counter
	intentsHolder *intentsstore.IntentsHolder
}

func newResource() (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.OTelLibraryName("otterize/network-mapper"),
			semconv.ServiceVersion(version.Version()),
		))
}

const DefaultMetricEndpoint = "ingest.lightstep.com:443"

// uses same name as expected in opentelemetry-collector-contrib's servicegraphprocessor
const CounterMetricName = "traces_service_graph_request_total"
const ClientAttributeName = "client"
const ServerAttributeName = "server"

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func newMeterProvider(ctx context.Context, res *resource.Resource, exportInterval time.Duration) (*sdk.MeterProvider, error) {
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(getenv("OTEL_EXPORTER_OTLP_ENDPOINT", DefaultMetricEndpoint)),
		otlpmetricgrpc.WithHeaders(map[string]string{
			"lightstep-access-token": os.Getenv("LS_ACCESS_TOKEN"),
		}),
		otlpmetricgrpc.WithTimeout(7*time.Second),
	)
	if err != nil {
		return nil, err
	}

	stdOutExporter, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}

	meterProvider := sdk.NewMeterProvider(
		sdk.WithResource(res),
		sdk.WithReader(sdk.NewPeriodicReader(stdOutExporter,
			sdk.WithInterval(exportInterval))),
		sdk.WithReader(sdk.NewPeriodicReader(metricExporter,
			sdk.WithInterval(exportInterval))),
	)
	return meterProvider, nil
}

func NewOtelExporter(ctx context.Context, ih *intentsstore.IntentsHolder, config Config) *OtelExporter {
	res, err := newResource()
	if err != nil {
		panic(err)
	}

	meterProvider, err := newMeterProvider(ctx, res, config.ExportInterval)
	if err != nil {
		panic(err)
	}

	// TODO: this is not the right place to handle shutdown
	// defer func() {
	// 	err := meterProvider.Shutdown(context.Background())
	// 	if err != nil {
	// 		logrus.Fatalln(err)
	// 	}
	// }()

	var meter = meterProvider.Meter("otelexporter")
	edgeCounter, err := meter.Int64Counter(
		CounterMetricName,
		metric.WithDescription("Count of edges between two nodes"),
	)
	if err != nil {
		panic(err)
	}

	return &OtelExporter{
		intentsHolder: ih,
		config:        config,
		counter:       edgeCounter,
	}
}

func (o *OtelExporter) countDiscoveredIntents(_ context.Context) {
	for _, intent := range o.intentsHolder.GetNewIntentsSinceLastGet() {
		clientName := intent.Intent.Client.Name
		serverName := intent.Intent.Server.Name
		o.counter.Add(context.Background(), 1, metric.WithAttributes(attribute.String(ClientAttributeName, clientName), attribute.String(ServerAttributeName, serverName)))
	}
}

func (o *OtelExporter) PeriodicIntentsExport(ctx context.Context) {
	for {
		select {
		case <-time.After(o.config.ExportInterval):
			o.countDiscoveredIntents(ctx)

		case <-ctx.Done():
			return
		}
	}
}
