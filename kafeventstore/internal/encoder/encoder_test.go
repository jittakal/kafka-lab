package encoder

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
)

func TestNewFactory(t *testing.T) {
	tests := []struct {
		name        string
		format      event.FileFormat
		compression string
	}{
		{"parquet with snappy", event.FormatParquet, "snappy"},
		{"parquet with gzip", event.FormatParquet, "gzip"},
		{"avro with gzip", event.FormatAvro, "gzip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory(tt.format, tt.compression)
			if factory == nil {
				t.Fatal("expected non-nil factory")
			}
			if factory.format != tt.format {
				t.Errorf("format = %v, want %v", factory.format, tt.format)
			}
			if factory.compression != tt.compression {
				t.Errorf("compression = %v, want %v", factory.compression, tt.compression)
			}
		})
	}
}

func TestFactory_CreateEncoder(t *testing.T) {
	tests := []struct {
		name    string
		format  event.FileFormat
		wantErr bool
	}{
		{"parquet format", event.FormatParquet, false},
		{"avro format", event.FormatAvro, false},
		{"unsupported format", event.FileFormat("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory(tt.format, "snappy")
			encoder, err := factory.CreateEncoder()

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateEncoder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && encoder == nil {
				t.Error("expected non-nil encoder")
			}
		})
	}
}

func TestSupportedFormats(t *testing.T) {
	formats := SupportedFormats()

	if len(formats) == 0 {
		t.Error("expected non-empty supported formats")
	}

	// Verify expected formats
	hasParquet := false
	hasAvro := false
	for _, f := range formats {
		if f == event.FormatParquet {
			hasParquet = true
		}
		if f == event.FormatAvro {
			hasAvro = true
		}
	}

	if !hasParquet {
		t.Error("expected parquet format in supported formats")
	}
	if !hasAvro {
		t.Error("expected avro format in supported formats")
	}
}

func TestSupportedCompressions(t *testing.T) {
	tests := []struct {
		name   string
		format event.FileFormat
		want   []string
	}{
		{
			name:   "parquet compressions",
			format: event.FormatParquet,
			want:   []string{"uncompressed", "snappy", "gzip", "lz4", "zstd"},
		},
		{
			name:   "avro compressions",
			format: event.FormatAvro,
			want:   []string{"uncompressed", "gzip"},
		},
		{
			name:   "invalid format",
			format: event.FileFormat("invalid"),
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SupportedCompressions(tt.format)
			if len(got) != len(tt.want) {
				t.Errorf("len(SupportedCompressions()) = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, c := range got {
				if c != tt.want[i] {
					t.Errorf("SupportedCompressions()[%d] = %v, want %v", i, c, tt.want[i])
				}
			}
		})
	}
}

func TestDefaultCompression(t *testing.T) {
	tests := []struct {
		name   string
		format event.FileFormat
		want   string
	}{
		{"parquet default", event.FormatParquet, "snappy"},
		{"avro default", event.FormatAvro, "gzip"},
		{"invalid default", event.FileFormat("invalid"), "uncompressed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultCompression(tt.format)
			if got != tt.want {
				t.Errorf("DefaultCompression() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParquetEncoder_FileExtension(t *testing.T) {
	encoder := NewParquetEncoder("snappy")
	ext := encoder.FileExtension()

	if ext != ".parquet" {
		t.Errorf("FileExtension() = %v, want .parquet", ext)
	}
}

func TestParquetEncoder_Encode(t *testing.T) {
	tempDir := os.TempDir()
	testFile := filepath.Join(tempDir, "test-encode.parquet")
	defer os.Remove(testFile)

	encoder := NewParquetEncoder("snappy")

	now := time.Now()
	records := []event.Record{
		{
			Event: &event.CloudEvent{
				SpecVersion: "1.0",
				ID:          "test-id-1",
				Source:      "test-source",
				Type:        "test.event",
				Time:        &now,
				Data:        []byte(`{"message": "test"}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    100,
				Timestamp: now,
			},
			Offset:      100,
			ProcessedAt: now,
		},
	}

	stats, err := encoder.Encode(testFile, records)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	if stats.RecordCount != len(records) {
		t.Errorf("RecordCount = %d, want %d", stats.RecordCount, len(records))
	}

	if stats.SizeBytes <= 0 {
		t.Errorf("SizeBytes = %d, want > 0", stats.SizeBytes)
	}

	// Verify file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("expected file at %s", testFile)
	}
}

func TestParquetEncoder_EncodeEmptyRecords(t *testing.T) {
	tempDir := os.TempDir()
	testFile := filepath.Join(tempDir, "test-empty.parquet")
	defer os.Remove(testFile)

	encoder := NewParquetEncoder("snappy")
	records := []event.Record{}

	_, err := encoder.Encode(testFile, records)
	if err == nil {
		t.Error("expected error for empty records")
	}
}

func TestParseCompressionCodec(t *testing.T) {
	tests := []struct {
		name        string
		compression string
	}{
		{"snappy lowercase", "snappy"},
		{"snappy uppercase", "SNAPPY"},
		{"gzip", "gzip"},
		{"lz4", "lz4"},
		{"zstd", "zstd"},
		{"uncompressed", "uncompressed"},
		{"none", "none"},
		{"unknown defaults to snappy", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := compressionCodec(tt.compression)
			if option == nil {
				t.Errorf("compressionCodec(%s) returned nil option", tt.compression)
			}
		})
	}
}

// Benchmark tests

func BenchmarkParquetEncoder_Encode(b *testing.B) {
	tmpDir := b.TempDir()

	now := time.Now()
	records := make([]event.Record, 100)
	for i := 0; i < 100; i++ {
		records[i] = event.Record{
			Event: &event.CloudEvent{
				ID:          "bench-" + string(rune(i)),
				Source:      "benchmark",
				SpecVersion: "1.0",
				Type:        "bench.event",
				Time:        &now,
				Data:        []byte(`{"benchmark": "data with reasonable payload"}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "benchmark-topic",
				Partition: 0,
				Offset:    int64(i),
				Timestamp: now,
			},
		}
	}

	encoder := NewParquetEncoder("snappy")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filePath := filepath.Join(tmpDir, "bench.parquet")

		if _, err := encoder.Encode(filePath, records); err != nil {
			b.Fatal(err)
		}

		os.Remove(filePath) // Clean up
	}
}

func BenchmarkAvroEncoder_Encode(b *testing.B) {
	tmpDir := b.TempDir()

	now := time.Now()
	records := make([]event.Record, 100)
	for i := 0; i < 100; i++ {
		records[i] = event.Record{
			Event: &event.CloudEvent{
				ID:          "bench-" + string(rune(i)),
				Source:      "benchmark",
				SpecVersion: "1.0",
				Type:        "bench.event",
				Time:        &now,
				Data:        []byte(`{"benchmark": "data with reasonable payload"}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "benchmark-topic",
				Partition: 0,
				Offset:    int64(i),
				Timestamp: now,
			},
		}
	}

	encoder, err := NewAvroEncoder("gzip")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filePath := filepath.Join(tmpDir, "bench.avro")

		if _, err := encoder.Encode(filePath, records); err != nil {
			b.Fatal(err)
		}

		os.Remove(filePath) // Clean up
	}
}

func BenchmarkFactory_CreateEncoder(b *testing.B) {
	factory := NewFactory(event.FormatParquet, "snappy")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := factory.CreateEncoder()
		if err != nil {
			b.Fatal(err)
		}
	}
}
