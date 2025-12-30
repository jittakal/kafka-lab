package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the consumer
type Metrics struct {
	EventsConsumed     *prometheus.CounterVec
	ConsumerErrors     *prometheus.CounterVec
	LastConsumedOffset *prometheus.GaugeVec
}

// NewMetrics creates and registers Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		EventsConsumed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafeventconsumer_events_consumed_total",
				Help: "Total number of events consumed",
			},
			[]string{"topic", "event_type"},
		),
		ConsumerErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafeventconsumer_errors_total",
				Help: "Total number of consumer errors",
			},
			[]string{"topic", "error_type"},
		),
		LastConsumedOffset: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafeventconsumer_last_consumed_offset",
				Help: "Last consumed offset by topic and partition",
			},
			[]string{"topic", "partition"},
		),
	}
}
