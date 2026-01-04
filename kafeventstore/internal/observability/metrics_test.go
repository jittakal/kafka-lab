package observability

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	if metrics == nil {
		t.Fatal("NewMetrics returned nil")
	}
}

func TestMetrics_IncMessagesConsumed(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	// Should not panic
	metrics.IncMessagesConsumed("test-topic", 0)
	metrics.IncMessagesConsumed("test-topic", 1)
	metrics.IncMessagesConsumed("another-topic", 0)
}

func TestMetrics_IncFilesWritten(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metrics.IncFilesWritten("test-topic", 0, "parquet", "success")
	metrics.IncFilesWritten("test-topic", 0, "avro", "success")
	metrics.IncFilesWritten("test-topic", 1, "parquet", "failure")
}

func TestMetrics_ObserveFileSize(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metrics.ObserveFileSize("test-topic", 0, "parquet", 1024.0)
	metrics.ObserveFileSize("test-topic", 1, "parquet", 2048.0)
}

func TestMetrics_ObserveStorageWriteDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metrics.ObserveStorageWriteDuration("test-topic", 0, 0.5)
	metrics.ObserveStorageWriteDuration("test-topic", 1, 1.2)
}

func TestMetrics_IncStorageErrors(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metrics.IncStorageErrors("s3", "upload")
	metrics.IncStorageErrors("azure", "encode")
	metrics.IncStorageErrors("file", "write")
}

func TestMetrics_ObserveCommitLatency(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metrics.ObserveCommitLatency("test-topic", 0, 0.1)
	metrics.ObserveCommitLatency("test-topic", 1, 0.2)
}

func TestMetrics_AllOperations(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	// Test a complete workflow
	metrics.IncMessagesConsumed("workflow-topic", 0)
	metrics.ObserveFileSize("workflow-topic", 0, "parquet", 5120.0)
	metrics.IncFilesWritten("workflow-topic", 0, "parquet", "success")
	metrics.ObserveStorageWriteDuration("workflow-topic", 0, 0.8)
	metrics.ObserveCommitLatency("workflow-topic", 0, 0.05)

	// Verify metrics were registered
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	if len(metricFamilies) == 0 {
		t.Error("No metrics were registered")
	}
}

func TestMetrics_IncRebalances(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metrics.IncRebalances("consumer-group-1")
	metrics.IncRebalances("consumer-group-2")
	metrics.IncRebalances("consumer-group-1")

	// Verify metric exists
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range metricFamilies {
		if *mf.Name == "kafka_rebalance_total" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected rebalances metric to be registered")
	}
}

func TestMetrics_IncOffsetCommits(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metrics.IncOffsetCommits("test-topic", 0, "success")
	metrics.IncOffsetCommits("test-topic", 1, "failure")
	metrics.IncOffsetCommits("test-topic", 0, "success")
}

func TestMetrics_ObserveRebalanceDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metrics.ObserveRebalanceDuration("consumer-group", 2.5)
	metrics.ObserveRebalanceDuration("consumer-group", 1.8)
}

func TestMetrics_SetPartitionsAssigned(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	metrics.SetPartitionsAssigned("test-topic", 5.0)
	metrics.SetPartitionsAssigned("test-topic", 3.0)
	metrics.SetPartitionsAssigned("another-topic", 10.0)
}

func TestMetrics_MultipleTopicsAndPartitions(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	topics := []string{"topic-1", "topic-2", "topic-3"}
	partitions := []int32{0, 1, 2}

	for _, topic := range topics {
		for _, partition := range partitions {
			metrics.IncMessagesConsumed(topic, partition)
			metrics.ObserveFileSize(topic, partition, "parquet", 1024.0)
			metrics.IncFilesWritten(topic, partition, "parquet", "success")
		}
	}

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	if len(metricFamilies) < 3 {
		t.Errorf("Expected at least 3 metric families, got %d", len(metricFamilies))
	}
}

func TestMetrics_ErrorScenarios(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	backends := []string{"s3", "azure", "file"}
	operations := []string{"upload", "encode", "write", "connect"}

	for _, backend := range backends {
		for _, operation := range operations {
			metrics.IncStorageErrors(backend, operation)
		}
	}

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range metricFamilies {
		if *mf.Name == "storage_errors_total" {
			found = true
			if len(mf.Metric) == 0 {
				t.Error("Expected error metrics to be recorded")
			}
			break
		}
	}
	if !found {
		t.Error("Expected storage errors metric to be registered")
	}
}

func TestMetrics_HighVolume(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	// Simulate high volume of metrics
	for i := 0; i < 1000; i++ {
		metrics.IncMessagesConsumed("high-volume-topic", int32(i%10))
	}

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	if len(metricFamilies) == 0 {
		t.Error("Metrics should be recorded")
	}
}
