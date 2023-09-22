package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	socketScanReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "socketscan_reported_connections",
		Help: "The total number of socket scan-based reported connections",
	})
	dnsCaptureReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dns_reported_connections",
		Help: "The total number of DNS-based reported connections",
	})
)

func IncrementSocketScanReports(count int) {
	socketScanReports.Add(float64(count))
}

func IncrementDNSCaptureReports(count int) {
	dnsCaptureReports.Add(float64(count))
}
