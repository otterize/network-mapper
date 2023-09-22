package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	istioReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "istio_reported_connections",
		Help: "The total number of Istio-sourced reported connections",
	})
)

func IncrementIstioReports(count int) {
	istioReports.Add(float64(count))
}
