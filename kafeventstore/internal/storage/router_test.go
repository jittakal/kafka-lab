package storage

import (
	"testing"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter("s3", "my-bucket", "events", "v1")

	if router.protocol != "s3" {
		t.Errorf("protocol = %v, want s3", router.protocol)
	}
	if router.bucket != "my-bucket" {
		t.Errorf("bucket = %v, want my-bucket", router.bucket)
	}
	if router.basePath != "events" {
		t.Errorf("basePath = %v, want events", router.basePath)
	}
	if router.version != "v1" {
		t.Errorf("version = %v, want v1", router.version)
	}
}

func TestDefaultRouter_Route(t *testing.T) {
	router := NewRouter("s3", "test-bucket", "base", "v1")

	partitionID := event.PartitionID{
		Topic:     "test-topic",
		Partition: 3,
	}

	// Use a fixed timestamp for consistent testing
	timestamp := time.Date(2025, 12, 18, 10, 30, 0, 0, time.UTC).Unix()

	tests := []struct {
		name        string
		specVersion string
		want        string
	}{
		{
			name:        "default version when specVersion is empty",
			specVersion: "",
			want:        "s3://test-bucket/base/test-topic/v1/dt=2025-12-18/pid=3/",
		},
		{
			name:        "spec version 1.0 transforms to v10",
			specVersion: "1.0",
			want:        "s3://test-bucket/base/test-topic/v10/dt=2025-12-18/pid=3/",
		},
		{
			name:        "spec version 1.1 transforms to v11",
			specVersion: "1.1",
			want:        "s3://test-bucket/base/test-topic/v11/dt=2025-12-18/pid=3/",
		},
		{
			name:        "spec version 2.0 transforms to v20",
			specVersion: "2.0",
			want:        "s3://test-bucket/base/test-topic/v20/dt=2025-12-18/pid=3/",
		},
		{
			name:        "spec version 1.0.0 transforms to v100",
			specVersion: "1.0.0",
			want:        "s3://test-bucket/base/test-topic/v100/dt=2025-12-18/pid=3/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.Route(partitionID, timestamp, tt.specVersion)
			if got != tt.want {
				t.Errorf("Route() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPolicy(t *testing.T) {
	config := PolicyConfig{
		MaxFileSizeMB:      100,
		MaxRecordsPerFile:  1000,
		MaxDurationSeconds: 300,
		Strategy:           "composite",
	}

	policy := NewPolicy(config)

	if policy == nil {
		t.Fatal("NewPolicy returned nil")
	}
	if policy.maxSizeBytes != 100*1024*1024 {
		t.Errorf("maxSizeBytes = %v, want %v", policy.maxSizeBytes, 100*1024*1024)
	}
	if policy.maxRecords != 1000 {
		t.Errorf("maxRecords = %v, want 1000", policy.maxRecords)
	}
}

func TestCompositePolicy_ShouldRotate_Size(t *testing.T) {
	config := PolicyConfig{
		MaxFileSizeMB:      10,
		MaxRecordsPerFile:  0,
		MaxDurationSeconds: 0,
	}
	policy := NewCompositePolicy(config)

	tests := []struct {
		name  string
		stats event.FileStats
		want  bool
	}{
		{
			name: "under size limit",
			stats: event.FileStats{
				SizeBytes:   5 * 1024 * 1024, // 5MB
				RecordCount: 100,
			},
			want: false,
		},
		{
			name: "at size limit",
			stats: event.FileStats{
				SizeBytes:   10 * 1024 * 1024, // 10MB
				RecordCount: 100,
			},
			want: true,
		},
		{
			name: "over size limit",
			stats: event.FileStats{
				SizeBytes:   15 * 1024 * 1024, // 15MB
				RecordCount: 100,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.ShouldRotate(tt.stats); got != tt.want {
				t.Errorf("ShouldRotate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompositePolicy_ShouldRotate_Count(t *testing.T) {
	config := PolicyConfig{
		MaxFileSizeMB:      0,
		MaxRecordsPerFile:  100,
		MaxDurationSeconds: 0,
	}
	policy := NewCompositePolicy(config)

	tests := []struct {
		name  string
		stats event.FileStats
		want  bool
	}{
		{
			name: "under count limit",
			stats: event.FileStats{
				RecordCount: 50,
			},
			want: false,
		},
		{
			name: "at count limit",
			stats: event.FileStats{
				RecordCount: 100,
			},
			want: true,
		},
		{
			name: "over count limit",
			stats: event.FileStats{
				RecordCount: 150,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.ShouldRotate(tt.stats); got != tt.want {
				t.Errorf("ShouldRotate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompositePolicy_ShouldRotate_Duration(t *testing.T) {
	config := PolicyConfig{
		MaxFileSizeMB:      0,
		MaxRecordsPerFile:  0,
		MaxDurationSeconds: 60, // 1 minute
	}
	policy := NewCompositePolicy(config)

	now := time.Now()

	tests := []struct {
		name  string
		stats event.FileStats
		want  bool
	}{
		{
			name: "recent file",
			stats: event.FileStats{
				FirstWriteTime: now.Add(-30 * time.Second),
			},
			want: false,
		},
		{
			name: "at duration limit",
			stats: event.FileStats{
				FirstWriteTime: now.Add(-60 * time.Second),
			},
			want: true,
		},
		{
			name: "old file",
			stats: event.FileStats{
				FirstWriteTime: now.Add(-120 * time.Second),
			},
			want: true,
		},
		{
			name: "zero first write time",
			stats: event.FileStats{
				FirstWriteTime: time.Time{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.ShouldRotate(tt.stats); got != tt.want {
				t.Errorf("ShouldRotate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompositePolicy_ShouldRotate_Composite(t *testing.T) {
	config := PolicyConfig{
		MaxFileSizeMB:      10,
		MaxRecordsPerFile:  100,
		MaxDurationSeconds: 60,
	}
	policy := NewCompositePolicy(config)

	now := time.Now()

	tests := []struct {
		name  string
		stats event.FileStats
		want  bool
	}{
		{
			name: "all under limits",
			stats: event.FileStats{
				SizeBytes:      5 * 1024 * 1024,
				RecordCount:    50,
				FirstWriteTime: now.Add(-30 * time.Second),
			},
			want: false,
		},
		{
			name: "size exceeds",
			stats: event.FileStats{
				SizeBytes:      11 * 1024 * 1024,
				RecordCount:    50,
				FirstWriteTime: now.Add(-30 * time.Second),
			},
			want: true,
		},
		{
			name: "count exceeds",
			stats: event.FileStats{
				SizeBytes:      5 * 1024 * 1024,
				RecordCount:    150,
				FirstWriteTime: now.Add(-30 * time.Second),
			},
			want: true,
		},
		{
			name: "duration exceeds",
			stats: event.FileStats{
				SizeBytes:      5 * 1024 * 1024,
				RecordCount:    50,
				FirstWriteTime: now.Add(-90 * time.Second),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.ShouldRotate(tt.stats); got != tt.want {
				t.Errorf("ShouldRotate() = %v, want %v", got, tt.want)
			}
		})
	}
}
