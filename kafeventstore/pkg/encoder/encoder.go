// Package encoder defines interfaces for encoding events to various file formats.
package encoder

import "github.com/jittakal/kafeventstore/pkg/event"

// Encoder encodes records to a specific file format.
type Encoder interface {
	// Encode writes records to a file and returns file statistics.
	Encode(filePath string, records []event.Record) (*event.FileStats, error)

	// Format returns the file format this encoder produces.
	Format() event.FileFormat

	// FileExtension returns the file extension (e.g., ".parquet", ".avro").
	FileExtension() string
}
