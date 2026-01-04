.PHONY: help build build-linux run test clean lint fmt vet deps tools check \
        docker-build docker-run docker-push docker-clean \
        test-coverage test-unit test-integration \
        install release

# Configuration
APP_NAME := kafka-event-blob-store
BINARY_NAME := event-store
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -w -s \
	-X main.Version=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.GitCommit=$(GIT_COMMIT)

# Docker configuration
DOCKER_REGISTRY ?= docker.io
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(APP_NAME)
DOCKER_TAG ?= $(VERSION)

# Default target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build          - Build the application for current OS"
	@echo "  build-linux    - Build Linux binary (for Docker/containers)"
	@echo "  install        - Install the binary to GOPATH/bin"
	@echo "  release        - Build release binaries for multiple platforms"
	@echo ""
	@echo "Run targets:"
	@echo "  run            - Run the application"
	@echo ""
	@echo "Test targets:"
	@echo "  test           - Run all tests with race detection"
	@echo "  test-unit      - Run only unit tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo ""
	@echo "Code quality targets:"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run linter"
	@echo "  check          - Run all checks (fmt, vet, lint, test)"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  docker-push    - Push Docker image to registry"
	@echo "  docker-clean   - Remove local Docker images"
	@echo ""
	@echo "Utility targets:"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download and tidy dependencies"
	@echo "  tools          - Install development tools"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION        = $(VERSION)"
	@echo "  DOCKER_IMAGE   = $(DOCKER_IMAGE)"
	@echo "  DOCKER_TAG     = $(DOCKER_TAG)"

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	@go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/main.go
	@echo "Build complete: bin/$(BINARY_NAME)"

# Build Linux binary (for Docker/containers)
build-linux:
	@echo "Building Linux binary..."
	@mkdir -p bin
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags "$(LDFLAGS) -extldflags '-static'" \
		-a -installsuffix cgo \
		-o bin/$(BINARY_NAME)-linux-amd64 \
		./cmd/main.go
	@echo "Build complete: bin/$(BINARY_NAME)-linux-amd64"

# Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install -ldflags "$(LDFLAGS)" ./cmd/main.go

# Build release binaries for multiple platforms
release:
	@echo "Building release binaries..."
	@mkdir -p bin/release
	@echo "Building Linux amd64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o bin/release/$(BINARY_NAME)-linux-amd64 ./cmd/main.go
	@echo "Building Linux arm64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" \
		-o bin/release/$(BINARY_NAME)-linux-arm64 ./cmd/main.go
	@echo "Building macOS amd64..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o bin/release/$(BINARY_NAME)-darwin-amd64 ./cmd/main.go
	@echo "Building macOS arm64..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" \
		-o bin/release/$(BINARY_NAME)-darwin-arm64 ./cmd/main.go
	@echo "Building Windows amd64..."
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o bin/release/$(BINARY_NAME)-windows-amd64.exe ./cmd/main.go
	@echo "Release build complete in bin/release/"

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	@go run ./cmd/main.go

# Run tests
test:
	@echo "Running tests with race detection..."
	@go test -v -race -cover -timeout 5m ./...

# Run unit tests only (fast)
test-unit:
	@echo "Running unit tests..."
	@go test -v -short -cover ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p bin build artifacts..."
	@rm -rf bin/
	@rm -f coverage.txt coverage.html
	@echo "Clean complete".txt -o bin/coverage.html
	@echo "Coverage report generated: bin/coverage.html"
	@go tool cover -func=bin/coverage.txt | grep total | awk '{print "Total coverage: " $$3}'

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.txt coverage.html

# Run linter
lint:
	@echo "Running linter..."
	@go tool golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running  (CI pipeline)
check: fmt vet lint test
	@echo "All checks passed!"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Run all checks
check: fmt vet lint test

# Install/sync tools (Go 1.25+ with tool directive)
tools: install-tools
install-tools:
	@echo "Installing development tools from go.mod..."
	@go get github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go mod tidy
	@echo "Tools installed. Use 'go tool <toolname>' to run them."

# Docker build $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		.
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Docker run
docker-run:
	@echo "Running Docker container..."
	@docker run --rm \
		-p 8080:8080 \
		-v $(PWD)/config:/app/config:ro \
		--name $(APP_NAME) \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

# Docker push
docker-push:
	@echo "Pushing Docker image to registry..."
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_IMAGE):latest
	@echo "Docker image pushed: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Docker clean
docker-clean:
	@echo "Removing Docker images..."
	@docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) || true
	@docker rmi $(DOCKER_IMAGE):latest || true
	@echo "Docker images removed"

# Docker compose up (if you add docker-compose.yml later)
docker-compose-up:
	@docker-compose up -d

# Docker compose down
docker-compose-down:
	@docker-compose down
	@echo "Running Docker container..."
	@docker run --rm -v $(PWD)/config:/app/config kafka-event-store:latest
