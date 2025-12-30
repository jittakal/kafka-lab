.PHONY: help build test test-coverage lint clean docker-build docker-push run docker-login docker-run fmt vet tidy helm-lint helm-package helm-login helm-push helm-release helm-clean helm-version install-tools all

# Variables
APP_NAME := kafeventproducer
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "latest")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-w -s -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Docker Variables
DOCKERHUB_USER := jittakal
DOCKER_REGISTRY := docker.io
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(DOCKERHUB_USER)/$(APP_NAME)
DOCKER_TAG := $(VERSION)

# Helm Variables
CHART_NAME := $(APP_NAME)chart
CHART_DIR := ./helm/$(CHART_NAME)
HELM_REGISTRY := registry-1.docker.io
HELM_OCI_REGISTRY := oci://$(HELM_REGISTRY)/$(DOCKERHUB_USER)
CHART_VERSION := $(shell grep '^version:' $(CHART_DIR)/Chart.yaml | awk '{print $$2}')
PACKAGE_FILE := $(CHART_NAME)-$(CHART_VERSION).tgz

# Directory Variables
BIN_DIR := bin
CMD_DIR := cmd/$(APP_NAME)

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet
GOMOD := $(GOCMD) mod

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BIN_DIR)/$(APP_NAME)"

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete"

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	$(GOCMD) tool golangci-lint run --timeout=5m ./...
	@echo "Lint complete"

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Format complete"

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Vet complete"

tidy: ## Tidy go modules
	@echo "Tidying modules..."
	$(GOMOD) tidy
	@echo "Tidy complete"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

run: build ## Build and run the application
	@echo "Running $(APP_NAME)..."
	./$(BIN_DIR)/$(APP_NAME) --config config/local-kafka-kraft.yaml

docker-build: ## Build Docker image
	@echo "Building Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)"
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		.
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-login: ## Login to Docker Hub
	@echo "Logging in to Docker Hub..."
	@if [ -z "$$DOCKERHUB_USERNAME" ]; then \
		echo "Error: DOCKERHUB_USERNAME environment variable not set"; \
		echo "Usage: export DOCKERHUB_USERNAME=your-username"; \
		echo "       export DOCKERHUB_TOKEN=your-token"; \
		exit 1; \
	fi
	@if [ -z "$$DOCKERHUB_TOKEN" ]; then \
		echo "Error: DOCKERHUB_TOKEN environment variable not set"; \
		echo "Usage: export DOCKERHUB_USERNAME=your-username"; \
		echo "       export DOCKERHUB_TOKEN=your-token"; \
		exit 1; \
	fi
	@echo $$DOCKERHUB_TOKEN | docker login $(DOCKER_REGISTRY) -u $$DOCKERHUB_USERNAME --password-stdin
	@echo "Successfully logged in to Docker Hub"

docker-push: docker-build ## Push Docker image to Docker Hub
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest
	@echo "Docker image pushed: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-run: docker-build ## Run Docker container locally
	@echo "Running Docker container..."
	docker run --rm -it \
		-p 9090:9090 \
		-v $(PWD)/config:/app/config \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

install-tools: ## Install development tools
	@echo "Installing development tools from go.mod tool directive..."
	$(GOCMD) get -tool
	@echo "Tools installed"

helm-version: ## Display current Helm chart version
	@echo "Current chart version: $(CHART_VERSION)"
	@echo "Package file: $(PACKAGE_FILE)"
	@echo "DockerHub repository: $(HELM_REGISTRY)/$(DOCKERHUB_USER)/$(CHART_NAME):$(CHART_VERSION)"

helm-lint: ## Lint the Helm chart
	@echo "Linting Helm chart..."
	helm lint $(CHART_DIR)

helm-package: helm-lint ## Package the Helm chart
	@echo "Packaging Helm chart: $(CHART_NAME) version $(CHART_VERSION)"
	helm package $(CHART_DIR)
	@echo "Package created: $(PACKAGE_FILE)"

helm-login: ## Login to DockerHub (OCI registry)
	@echo "Logging in to DockerHub registry..."
	@if [ -z "$$DOCKERHUB_USERNAME" ]; then \
		echo "Error: DOCKERHUB_USERNAME environment variable not set"; \
		echo "Usage: export DOCKERHUB_USERNAME=your-username"; \
		echo "       export DOCKERHUB_TOKEN=your-token"; \
		exit 1; \
	fi
	@if [ -z "$$DOCKERHUB_TOKEN" ]; then \
		echo "Error: DOCKERHUB_TOKEN environment variable not set"; \
		echo "Usage: export DOCKERHUB_USERNAME=your-username"; \
		echo "       export DOCKERHUB_TOKEN=your-token"; \
		exit 1; \
	fi
	@echo $$DOCKERHUB_TOKEN | helm registry login $(HELM_REGISTRY) -u $$DOCKERHUB_USERNAME --password-stdin
	@echo "Successfully logged in to DockerHub"

helm-push: ## Push the Helm chart to DockerHub
	@if [ ! -f "$(PACKAGE_FILE)" ]; then \
		echo "Error: Package file $(PACKAGE_FILE) not found. Run 'make helm-package' first."; \
		exit 1; \
	fi
	@echo "Pushing chart to DockerHub: $(DOCKERHUB_USER)/$(CHART_NAME):$(CHART_VERSION)"
	helm push $(PACKAGE_FILE) $(HELM_OCI_REGISTRY)
	@echo "Chart successfully pushed to $(HELM_REGISTRY)/$(DOCKERHUB_USER)/$(CHART_NAME):$(CHART_VERSION)"

helm-release: helm-package helm-login helm-push ## Full Helm release workflow: package, login, and push
	@echo ""
	@echo "=========================================="
	@echo "Release Complete!"
	@echo "=========================================="
	@echo "Chart: $(CHART_NAME)"
	@echo "Version: $(CHART_VERSION)"
	@echo "Repository: $(HELM_REGISTRY)/$(DOCKERHUB_USER)/$(CHART_NAME):$(CHART_VERSION)"
	@echo ""
	@echo "To install this chart:"
	@echo "  helm install my-$(CHART_NAME) $(HELM_OCI_REGISTRY)/$(CHART_NAME) --version $(CHART_VERSION)"
	@echo ""

helm-clean: ## Clean packaged Helm chart files
	@echo "Cleaning packaged chart files..."
	rm -f *.tgz
	@echo "Clean complete"
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"

all: clean fmt vet lint test build ## Run all checks and build

.DEFAULT_GOAL := help
