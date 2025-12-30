package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector holds Prometheus metrics collectors
type Collector struct {
	eventsProducedTotal *prometheus.CounterVec
	eventsFailedTotal   *prometheus.CounterVec
	producerDuration    *prometheus.HistogramVec
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		eventsProducedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafeventproducer_events_produced_total",
				Help: "Total number of events produced to Kafka",
			},
			[]string{"topic", "type"},
		),
		eventsFailedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafeventproducer_events_failed_total",
				Help: "Total number of failed event productions",
			},
			[]string{"topic", "type"},
		),
		producerDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kafeventproducer_producer_duration_seconds",
				Help:    "Duration of event production in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"topic"},
		),
	}
}

// IncEventsProducedTotal increments the events produced counter
func (c *Collector) IncEventsProducedTotal(topic, eventType string) {
	c.eventsProducedTotal.WithLabelValues(topic, eventType).Inc()
}

// IncEventsFailedTotal increments the events failed counter
func (c *Collector) IncEventsFailedTotal(topic, eventType string) {
	c.eventsFailedTotal.WithLabelValues(topic, eventType).Inc()
}

// ObserveProducerDuration records the duration of event production
func (c *Collector) ObserveProducerDuration(topic string, duration float64) {
	c.producerDuration.WithLabelValues(topic).Observe(duration)
}
