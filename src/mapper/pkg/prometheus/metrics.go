package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	socketScanReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "socketscan_reported_connections",
		Help: "The total number of socket scan-sourced reported connections",
	})
	dnsCaptureReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dns_reported_connections",
		Help: "The total number of DNS-sourced reported connections",
	})
	kafkaReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_reported_topics",
		Help: "The total number of Kafka-sourced topics",
	})
	istioReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "istio_reported_connections",
		Help: "The total number of Istio-sourced connections",
	})
)

func IncrementSocketScanReports(count int) {
	socketScanReports.Add(float64(count))
}

func IncrementDNSCaptureReports(count int) {
	dnsCaptureReports.Add(float64(count))
}

func IncrementKafkaReports(count int) {
	kafkaReports.Add(float64(count))
}

func IncrementIstioReports(count int) {
	istioReports.Add(float64(count))
}
