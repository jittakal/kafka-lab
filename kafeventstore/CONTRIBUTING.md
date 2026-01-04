# Contributing to kafka-event-blob-store

Thank you for your interest in contributing! This document provides guidelines and standards for contributing to this project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Code Standards](#code-standards)
- [Testing Guidelines](#testing-guidelines)
- [Pull Request Process](#pull-request-process)
- [Code Review](#code-review)

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help maintain a positive community

## Getting Started

### Prerequisites

- Go 1.25+ ([Download](https://go.dev/dl/))
- Git

**Note**: Development tools like `golangci-lint` are managed via Go 1.25's `tool` directive in `go.mod`. Run `make tools` to install them.

### Fork and Clone

```bash
# Fork the repository on GitHub
# Clone your fork
git clone https://github.com/YOUR_USERNAME/kafka-event-blob-store.git
cd kafka-event-blob-store

# Add upstream remote
git remote add upstream https://github.com/jittakal/kafeventstore.git
```

## Development Setup

### Install Dependencies

```bash
# Download Go modules
go mod download

# Verify dependencies
go mod verify

# Install development tools (Go 1.25+ tool directive)
make tools
```

### Build the Project

```bash
# Build
go build ./cmd/...

# Or use Makefile if available
make build
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/buffer/...

# Run with race detector
go test -race ./...
```

## Code Standards

### Go Style Guide

We follow standard Go idioms and conventions:

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### Naming Conventions

#### Packages

- Use lowercase, single-word names
- Package name should match directory name
- Avoid `util`, `common`, `helper` package names

```go
// Good
package buffer
package encoder

// Avoid
package utils
package helpers
```

#### Functions and Methods

- Use MixedCaps (camelCase for unexported, PascalCase for exported)
- Method names should be clear and descriptive
- Avoid stutter (don't repeat package name)

```go
// Good
func (b *Buffer) Add(record event.Record) error
func newWriter() *Writer

// Avoid
func (b *Buffer) BufferAdd(record event.Record) error
func NewWriterWriter() *Writer
```

#### Variables

- Use short names for small scopes
- Use descriptive names for larger scopes
- Common abbreviations: `err`, `ctx`, `cfg`, `msg`

```go
// Good
for i, record := range records {
    if err := processRecord(record); err != nil {
        return err
    }
}

// For larger scope
bufferManager := NewManager()
```

### Code Organization

#### Package Structure

```
pkg/           - Public APIs (interfaces)
internal/      - Private implementations
cmd/           - Application entry points
config/        - Configuration files
docs/          - Documentation
```

#### Separation of Concerns

- Define interfaces in consumer packages (`pkg/`)
- Implement in provider packages (`internal/`)
- Use compile-time verification

```go
// pkg/buffer/buffer.go
type Buffer interface {
    Add(record event.Record) error
}

// internal/buffer/buffer.go
var _ buffer.Buffer = (*PartitionBuffer)(nil)
```

### Error Handling

#### Use Sentinel Errors

```go
var (
    ErrBufferFull = errors.New("buffer is full")
    ErrNotFound   = errors.New("not found")
)
```

#### Custom Error Types

```go
type ProcessingError struct {
    PartitionID event.PartitionID
    Offset      int64
    Err         error
}

func (e *ProcessingError) Error() string {
    return fmt.Sprintf("processing error at %s offset %d: %v",
        e.PartitionID, e.Offset, e.Err)
}

func (e *ProcessingError) Unwrap() error {
    return e.Err
}
```

#### Error Wrapping

```go
// Good - preserve error chain
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Avoid - loses error context
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %v", err)
}
```

### Documentation

#### Package Documentation

Every package must have a `doc.go` file:

```go
// Package buffer provides thread-safe buffering for event records.
//
// This package implements in-memory buffering with automatic size
// and count limits, designed for batching events before writing to storage.
//
// # Example Usage
//
//     buf := buffer.New(partitionID, maxSize, maxRecords)
//     buf.Add(record)
//     records := buf.Drain()
package buffer
```

#### Function Documentation

All exported functions must have godoc comments:

```go
// New creates a new PartitionBuffer with the specified limits.
// The buffer will reject new records once either maxSizeBytes or
// maxRecords is reached.
func New(partitionID event.PartitionID, maxSizeBytes int64, maxRecords int) *PartitionBuffer {
```

#### Example Tests

Provide example tests for public APIs:

```go
func ExamplePartitionBuffer() {
    buf := buffer.New(partitionID, 1024*1024, 1000)
    buf.Add(record)
    stats := buf.Stats()
    fmt.Printf("Buffered %d records\n", stats.RecordCount)
    // Output: Buffered 1 records
}
```

### Concurrency

#### Use Mutexes Correctly

```go
type Buffer struct {
    mu      sync.RWMutex
    records []event.Record
}

// Write operations use Lock
func (b *Buffer) Add(record event.Record) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    // modify state
}

// Read operations use RLock
func (b *Buffer) Stats() event.FileStats {
    b.mu.RLock()
    defer b.mu.RUnlock()
    // read state
}
```

#### Channel Usage

- Use buffered channels for producers
- Always document channel ownership
- Close channels from sender side

### Interface Design

#### Keep Interfaces Small

```go
// Good - focused interface
type Encoder interface {
    Encode(filePath string, records []event.Record) (*event.FileStats, error)
}

// Avoid - too many methods
type Encoder interface {
    Encode(...) error
    Decode(...) error
    Validate(...) error
    Convert(...) error
}
```

## Testing Guidelines

### Test Organization

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {
            name:  "valid input",
            input: validInput,
            want:  expectedOutput,
        },
        {
            name:    "invalid input",
            input:   invalidInput,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Coverage Requirements

- Minimum 90% coverage for all packages
- Test both success and error paths
- Test edge cases and boundary conditions

```bash
# Check coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

### Benchmark Tests

Provide benchmarks for performance-critical code:

```go
func BenchmarkBufferAdd(b *testing.B) {
    buf := New(partitionID, 1024*1024, 100000)
    record := createTestRecord()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        buf.Add(record)
    }
}
```

### Test Helpers

Use `t.Helper()` for test helper functions:

```go
func createTestRecord(t *testing.T, id string) event.Record {
    t.Helper()
    // create and return record
}
```

## Pull Request Process

### Before Submitting

1. **Run tests and linters**

```bash
# Run all tes (uses go tool from go.mod)
make lint
# or directly:
go tool go test ./...

# Run linters
golangci-lint run

# Check formatting
go fmt ./...
```

2. **Update documentation**
   - Update README if needed
   - Add/update godoc comments
   - Update CHANGELOG.md

3. **Write clear commit messages**

```
feat: Add support for Zstandard compression

- Implement zstd compression for Parquet encoder
- Add tests for new compression codec
- Update documentation

Closes #123
```

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding or updating tests
- `refactor:` Code refactoring
- `perf:` Performance improvement
- `chore:` Build process or tooling changes

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] All tests passing

## Checklist
- [ ] Code follows project style
- [ ] Self-review completed
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] No new warnings
```

## Code Review

### Review Checklist

#### Functionality
- [ ] Code works as intended
- [ ] Edge cases handled
- [ ] Error handling appropriate

#### Code Quality
- [ ] Follows Go idioms
- [ ] Clear and readable
- [ ] No unnecessary complexity
- [ ] Proper abstractions

#### Testing
- [ ] Tests cover new code
- [ ] Tests are meaningful
- [ ] Coverage maintained/improved

#### Documentation
- [ ] Godoc comments present
- [ ] Package docs updated
- [ ] Examples provided

### Review Process

1. Automated checks must pass
2. At least one approval required
3. All comments addressed
4. Squash and merge when approved

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for general questions
- Join our community chat (if available)

## License

By contributing, you agree that your contributions will be licensed under the project's license.
