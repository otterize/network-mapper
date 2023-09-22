package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	topicReports = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_reported_topics",
		Help: "The total number of Kafka topics reported.",
	})
)

func IncrementKafkaTopicReports(count int) {
	topicReports.Add(float64(count))
}
