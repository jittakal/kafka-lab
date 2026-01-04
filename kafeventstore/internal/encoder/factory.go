// Package encoder implements encoder factory for creating file format encoders.
package encoder

import (
	"fmt"

	"github.com/jittakal/kafeventstore/pkg/encoder"
	"github.com/jittakal/kafeventstore/pkg/event"
)

// Factory creates encoders based on format and configuration.
type Factory struct {
	format      event.FileFormat
	compression string
}

// NewFactory creates a new encoder factory.
func NewFactory(format event.FileFormat, compression string) *Factory {
	return &Factory{
		format:      format,
		compression: compression,
	}
}

// CreateEncoder creates an encoder based on the configured format.
func (f *Factory) CreateEncoder() (encoder.Encoder, error) {
	switch f.format {
	case event.FormatParquet:
		return NewParquetEncoder(f.compression), nil
	case event.FormatAvro:
		return NewAvroEncoder(f.compression)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", f.format)
	}
}

// SupportedFormats returns a list of supported file formats.
func SupportedFormats() []event.FileFormat {
	return []event.FileFormat{
		event.FormatParquet,
		event.FormatAvro,
	}
}

// SupportedCompressions returns supported compression codecs for a given format.
func SupportedCompressions(format event.FileFormat) []string {
	switch format {
	case event.FormatParquet:
		return []string{"uncompressed", "snappy", "gzip", "lz4", "zstd"}
	case event.FormatAvro:
		return []string{"uncompressed", "gzip"}
	default:
		return []string{}
	}
}

// DefaultCompression returns the default compression for a format.
func DefaultCompression(format event.FileFormat) string {
	switch format {
	case event.FormatParquet:
		return "snappy"
	case event.FormatAvro:
		return "gzip"
	default:
		return "uncompressed"
	}
}
