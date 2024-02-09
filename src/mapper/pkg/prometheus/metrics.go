package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	socketScanReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "socketscan_reported_connections",
		Help: "The total number of socket scan-sourced successfully reported connections",
	})
	dnsCaptureReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dns_reported_connections",
		Help: "The total number of DNS-sourced successfully reported connections",
	})
	kafkaReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_reported_topics",
		Help: "The total number of Kafka-sourced successfully reported topics",
	})
	istioReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "istio_reported_connections",
		Help: "The total number of Istio-sourced successfully reported connections",
	})
	socketScanIncomingReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "socketscan_reported_connections",
		Help: "The total number of socket scan-sourced incoming connections",
	})
	dnsCaptureIncomingReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dns_incoming_connections",
		Help: "The total number of DNS-sourced incoming connections",
	})
	kafkaIncomingReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_incoming_topics",
		Help: "The total number of Kafka-sourced incoming topics",
	})
	istioIncomingReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "istio_incoming_connections",
		Help: "The total number of Istio-sourced incoming connections",
	})
	socketScanDrops = promauto.NewCounter(prometheus.CounterOpts{
		Name: "socketscan_dropped_connections",
		Help: "The total number of socket scan-sourced reported connections that were dropped for performance",
	})
	dnsCaptureDrops = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dns_dropped_connections",
		Help: "The total number of DNS-sourced reported connections that were dropped for performance",
	})
	kafkaReportsDrops = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_dropped_topics",
		Help: "The total number of Kafka-sourced reported topics that were dropped for performance",
	})
	istioReportsDrops = promauto.NewCounter(prometheus.CounterOpts{
		Name: "istio_dropped_connections",
		Help: "The total number of Istio-sourced reported connections that were dropped for performance",
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

func IncrementSocketScanIncomingReports(count int) {
	socketScanIncomingReports.Add(float64(count))
}

func IncrementDNSCaptureIncomingReports(count int) {
	dnsCaptureIncomingReports.Add(float64(count))
}

func IncrementKafkaIncomingReports(count int) {
	kafkaIncomingReports.Add(float64(count))
}

func IncrementIstioIncomingReports(count int) {
	istioIncomingReports.Add(float64(count))
}

func IncrementSocketScanDrops(count int) {
	socketScanDrops.Add(float64(count))
}

func IncrementDNSCaptureDrops(count int) {
	dnsCaptureDrops.Add(float64(count))
}

func IncrementKafkaDrops(count int) {
	kafkaReportsDrops.Add(float64(count))
}

func IncrementIstioDrops(count int) {
	istioReportsDrops.Add(float64(count))
}
