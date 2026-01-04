package observability

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics.
type Metrics struct {
	// Consumer metrics
	MessagesConsumed   *prometheus.CounterVec
	ConsumerLag        *prometheus.GaugeVec
	OffsetCommits      *prometheus.CounterVec
	Rebalances         *prometheus.CounterVec
	RebalanceDuration  *prometheus.HistogramVec
	PartitionsAssigned *prometheus.GaugeVec
	CommitLatency      *prometheus.HistogramVec

	// Processing metrics
	EventsProcessed    *prometheus.CounterVec
	ProcessingDuration *prometheus.HistogramVec
	BufferSize         *prometheus.GaugeVec
	BufferRecordCount  *prometheus.GaugeVec

	// Storage metrics
	FilesWritten         *prometheus.CounterVec
	FileWriteDuration    *prometheus.HistogramVec
	StorageWriteDuration *prometheus.HistogramVec
	FileSize             *prometheus.HistogramVec
	StorageErrors        *prometheus.CounterVec
}

// NewMetrics creates and registers all Prometheus metrics.
func NewMetrics(registry *prometheus.Registry) *Metrics {
	factory := promauto.With(registry)

	return &Metrics{
		// Consumer metrics
		MessagesConsumed: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafka_messages_consumed_total",
				Help: "Total number of messages consumed from Kafka",
			},
			[]string{"topic", "partition"},
		),
		ConsumerLag: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_consumer_lag",
				Help: "Current consumer lag",
			},
			[]string{"topic", "partition"},
		),
		OffsetCommits: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafka_offset_commit_total",
				Help: "Total number of offset commits",
			},
			[]string{"topic", "partition", "status"},
		),
		Rebalances: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafka_rebalance_total",
				Help: "Total number of consumer group rebalances",
			},
			[]string{"group"},
		),
		RebalanceDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kafka_rebalance_duration_seconds",
				Help:    "Duration of consumer group rebalances",
				Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0},
			},
			[]string{"group"},
		),
		PartitionsAssigned: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_partitions_assigned",
				Help: "Number of partitions currently assigned to this consumer",
			},
			[]string{"topic"},
		),
		CommitLatency: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kafka_commit_latency_seconds",
				Help:    "Latency of offset commit operations",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
			[]string{"topic", "partition"},
		),

		// Processing metrics
		EventsProcessed: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "events_processed_total",
				Help: "Total number of events processed",
			},
			[]string{"topic", "partition", "status"},
		),
		ProcessingDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "processing_duration_seconds",
				Help:    "Duration of event processing operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"topic", "operation"},
		),
		BufferSize: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "buffer_size_bytes",
				Help: "Current buffer size in bytes",
			},
			[]string{"topic", "partition"},
		),
		BufferRecordCount: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "buffer_record_count",
				Help: "Current number of records in buffer",
			},
			[]string{"topic", "partition"},
		),

		// Storage metrics
		FilesWritten: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "files_written_total",
				Help: "Total number of files written to storage",
			},
			[]string{"topic", "partition", "format", "status"},
		),
		FileWriteDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "file_write_duration_seconds",
				Help:    "Duration of file write operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"backend", "format"},
		),
		StorageWriteDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "storage_write_duration_seconds",
				Help:    "Duration of complete storage write operations including encoding",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"topic", "partition"},
		),
		FileSize: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "file_size_bytes",
				Help:    "Size of files written to storage",
				Buckets: prometheus.ExponentialBuckets(1024*1024, 2, 10), // 1MB to 1GB
			},
			[]string{"topic", "partition", "format"},
		),
		StorageErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "storage_errors_total",
				Help: "Total number of storage errors",
			},
			[]string{"backend", "error_type"},
		),
	}
}

// IncMessagesConsumed increments messages consumed counter.
func (m *Metrics) IncMessagesConsumed(topic string, partition int32) {
	m.MessagesConsumed.WithLabelValues(topic, fmt.Sprintf("%d", partition)).Inc()
}

// IncRebalances increments rebalances counter.
func (m *Metrics) IncRebalances(groupID string) {
	m.Rebalances.WithLabelValues(groupID).Inc()
}

// IncOffsetCommits increments offset commits counter.
func (m *Metrics) IncOffsetCommits(topic string, partition int32, status string) {
	m.OffsetCommits.WithLabelValues(topic, fmt.Sprintf("%d", partition), status).Inc()
}

// ObserveRebalanceDuration observes rebalance duration.
func (m *Metrics) ObserveRebalanceDuration(groupID string, duration float64) {
	m.RebalanceDuration.WithLabelValues(groupID).Observe(duration)
}

// ObserveCommitLatency observes commit latency.
func (m *Metrics) ObserveCommitLatency(topic string, partition int32, duration float64) {
	m.CommitLatency.WithLabelValues(topic, fmt.Sprintf("%d", partition)).Observe(duration)
}

// SetPartitionsAssigned sets partitions assigned gauge.
func (m *Metrics) SetPartitionsAssigned(topic string, count float64) {
	m.PartitionsAssigned.WithLabelValues(topic).Set(count)
}

// IncFilesWritten increments files written counter.
func (m *Metrics) IncFilesWritten(topic string, partition int32, format string, status string) {
	m.FilesWritten.WithLabelValues(topic, fmt.Sprintf("%d", partition), format, status).Inc()
}

// ObserveFileSize observes file size.
func (m *Metrics) ObserveFileSize(topic string, partition int32, format string, size float64) {
	m.FileSize.WithLabelValues(topic, fmt.Sprintf("%d", partition), format).Observe(size)
}

// ObserveStorageWriteDuration observes storage write duration.
func (m *Metrics) ObserveStorageWriteDuration(topic string, partition int32, duration float64) {
	m.StorageWriteDuration.WithLabelValues(topic, fmt.Sprintf("%d", partition)).Observe(duration)
}

// IncStorageErrors increments storage errors counter.
func (m *Metrics) IncStorageErrors(backend string, operation string) {
	m.StorageErrors.WithLabelValues(backend, operation).Inc()
}
