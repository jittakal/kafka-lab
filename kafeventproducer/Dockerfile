# Multi-stage Dockerfile for kafeventproducer
# Stage 1: Build
FROM golang:1.25.5-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates && update-ca-certificates

# Set working directory
WORKDIR /build

# Copy everything first (includes go.sum with hashes)
COPY . .

# Download dependencies (using vendor or go.sum for verification)
RUN go mod download || true

# Build the application anyway
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION:-dev} -X main.commit=${COMMIT:-none} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /build/bin/kafeventproducer \
    ./cmd/kafeventproducer

# Stage 2: Runtime
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bin/kafeventproducer /app/kafeventproducer

# Copy configuration files
COPY --from=builder /build/config /app/config

# Set ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose metrics port
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9090/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/kafeventproducer"]

# Default command
CMD ["--config", "/app/config/local-kafka-kraft.yaml"]
