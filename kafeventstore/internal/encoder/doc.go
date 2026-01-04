// Package encoder provides event encoding to various file formats.
//
// This package implements encoders for converting event records into file formats
// suitable for storage and analytics, with configurable compression.
//
// # Supported Formats
//
// The package supports two file formats:
//
//   - Parquet: Columnar format optimized for analytics and Athena queries
//   - Avro: Row-based format with embedded schema
//
// # Encoder Factory
//
// Use Factory to create encoder instances:
//
//	factory := encoder.NewFactory(event.FormatParquet, "snappy")
//	enc, err := factory.CreateEncoder()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Direct Encoder Creation
//
// Create encoders directly when format is known:
//
//	// Parquet with Snappy compression
//	parquetEnc := encoder.NewParquetEncoder("snappy")
//
//	// Avro with GZIP compression
//	avroEnc, err := encoder.NewAvroEncoder("gzip")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Encoding Records
//
// All encoders implement the pkg/encoder.Encoder interface:
//
//	stats, err := encoder.Encode(filePath, records)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Encoded %d records, %d bytes\n",
//	    stats.RecordCount, stats.SizeBytes)
//
// # Parquet Encoder
//
// Produces columnar Parquet files compatible with AWS Athena:
//
//   - Snappy compression (default, fastest queries)
//   - GZIP compression (better compression ratio)
//   - Schema optimized for CloudEvents + Kafka metadata
//   - Automatic type conversion (timestamps to int64 unix)
//
// # Avro Encoder
//
// Produces row-based Avro files with embedded schema:
//
//   - GZIP compression (default)
//   - Deflate compression
//   - Schema includes CloudEvents 1.0 fields
//   - Supports both required and optional fields
//
// # Compression Options
//
// Supported compression codecs:
//
//	Parquet: "snappy", "gzip", "zstd", "none"
//	Avro:    "gzip", "deflate", "snappy", "none"
//
// # File Extensions
//
// Encoders provide appropriate file extensions:
//
//	parquetEnc.FileExtension()  // ".parquet"
//	avroEnc.FileExtension()     // ".avro.gz" (with gzip)
//
// # Schema Management
//
// Schemas are embedded in the encoder implementations:
//
//   - Parquet: Uses parquet-go/parquet package with struct tags
//   - Avro: Uses predefined Avro schema JSON
//
// Both schemas are optimized for the CloudEvent + Kafka metadata structure.
//
// # Thread Safety
//
// Encoder instances are safe for concurrent use. Factory.CreateEncoder()
// creates independent encoder instances.
//
// # Performance
//
// Benchmarks show encoding performance:
//
//	BenchmarkParquetEncoder_Encode  (100 records): ~50ms
//	BenchmarkAvroEncoder_Encode     (100 records): ~30ms
//
// Run benchmarks with: go test -bench=. ./internal/encoder/...
package encoder
