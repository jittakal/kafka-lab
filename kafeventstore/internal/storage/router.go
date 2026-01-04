// Package storage implements storage-related functionality.
package storage

import (
	"fmt"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
	"github.com/jittakal/kafeventstore/pkg/storage"
)

// Ensure implementations satisfy interfaces.
var (
	_ storage.Router         = (*DefaultRouter)(nil)
	_ storage.RotationPolicy = (*CompositePolicy)(nil)
)

// DefaultRouter implements Hive-style partitioning for storage paths.
type DefaultRouter struct {
	protocol string
	bucket   string
	basePath string
	version  string
}

// NewRouter creates a new storage router.
func NewRouter(protocol, bucket, basePath, version string) *DefaultRouter {
	return &DefaultRouter{
		protocol: protocol,
		bucket:   bucket,
		basePath: basePath,
		version:  version,
	}
}

// Route returns the storage path for a partition at the given timestamp.
// Format: protocol://bucket/basePath/topic/version/dt=YYYY-MM-DD/pid=N/
// Uses event time (from CloudEvent.Time) instead of processing time for partitioning.
// If specVersion is provided, it overrides the default version.
// SpecVersion transformation: "1.0" -> "v10", "1.1" -> "v11", "2.0" -> "v20", etc.
func (r *DefaultRouter) Route(partitionID event.PartitionID, timestamp int64, specVersion string) string {
	t := time.Unix(timestamp, 0).UTC()
	date := t.Format("2006-01-02")

	// Use spec_version from event if provided, otherwise use default version
	version := r.version
	if specVersion != "" {
		// Transform spec version: remove dots and prepend 'v'
		// Examples: "1.0" -> "v10", "1.1" -> "v11", "2.0" -> "v20"
		versionStr := ""
		for _, ch := range specVersion {
			if ch != '.' {
				versionStr += string(ch)
			}
		}
		if versionStr != "" {
			version = "v" + versionStr
		}
	}

	return fmt.Sprintf("%s://%s/%s/%s/%s/dt=%s/pid=%d/",
		r.protocol,
		r.bucket,
		r.basePath,
		partitionID.Topic,
		version,
		date,
		partitionID.Partition,
	)
}

// NewPolicy creates a new rotation policy (alias for NewCompositePolicy).
func NewPolicy(config PolicyConfig) *CompositePolicy {
	return NewCompositePolicy(config)
}

// RotationStrategy determines when to rotate files.
type RotationStrategy string

const (
	StrategyComposite RotationStrategy = "composite"
	StrategySizeOnly  RotationStrategy = "size"
	StrategyTimeOnly  RotationStrategy = "time"
	StrategyCount     RotationStrategy = "count"
)

// PolicyConfig configures rotation behavior.
type PolicyConfig struct {
	MaxFileSizeMB      int64
	MaxRecordsPerFile  int
	MaxDurationSeconds int
	Strategy           string
}

// CompositePolicy rotates based on multiple criteria.
type CompositePolicy struct {
	maxSizeBytes int64
	maxRecords   int
	maxDuration  time.Duration
}

// NewCompositePolicy creates a new composite rotation policy.
func NewCompositePolicy(config PolicyConfig) *CompositePolicy {
	return &CompositePolicy{
		maxSizeBytes: config.MaxFileSizeMB * 1024 * 1024,
		maxRecords:   config.MaxRecordsPerFile,
		maxDuration:  time.Duration(config.MaxDurationSeconds) * time.Second,
	}
}

// ShouldRotate returns true if any rotation condition is met.
func (p *CompositePolicy) ShouldRotate(stats event.FileStats) bool {
	// Size-based rotation
	if p.maxSizeBytes > 0 && stats.SizeBytes >= p.maxSizeBytes {
		return true
	}

	// Count-based rotation
	if p.maxRecords > 0 && stats.RecordCount >= p.maxRecords {
		return true
	}

	// Time-based rotation
	if p.maxDuration > 0 && !stats.FirstWriteTime.IsZero() {
		age := time.Since(stats.FirstWriteTime)
		if age >= p.maxDuration {
			return true
		}
	}

	return false
}
