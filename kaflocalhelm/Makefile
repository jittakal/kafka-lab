.PHONY: help helm-lint helm-package helm-login helm-push helm-release helm-clean helm-version helm-install-test helm-pull helm-show bump-patch bump-minor bump-major clean

# Variables
CHART_NAME := kafka-kraft-sasl-ssl
CHART_DIR := ./$(CHART_NAME)
DOCKERHUB_USER := jittakal

# Registry Variables
HELM_REGISTRY := registry-1.docker.io
HELM_OCI_REGISTRY := oci://$(HELM_REGISTRY)/$(DOCKERHUB_USER)

CHART_VERSION := $(shell grep '^version:' $(CHART_DIR)/Chart.yaml | awk '{print $$2}')
PACKAGE_FILE := $(CHART_NAME)-$(CHART_VERSION).tgz

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

helm-version: ## Display current Helm chart version
	@echo "Current chart version: $(CHART_VERSION)"
	@echo "Package file: $(PACKAGE_FILE)"
	@echo "DockerHub repository: $(HELM_REGISTRY)/$(DOCKERHUB_USER)/$(CHART_NAME):$(CHART_VERSION)"

helm-lint: ## Lint the Helm chart
	@echo "Linting Helm chart..."
	helm lint $(CHART_DIR)
	@echo "Lint complete"

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
	@echo "  helm install my-kafka $(HELM_OCI_REGISTRY)/$(CHART_NAME) --version $(CHART_VERSION)"
	@echo ""

helm-clean: ## Clean packaged Helm chart files
	@echo "Cleaning packaged chart files..."
	rm -f *.tgz
	@echo "Clean complete"

helm-install-test: ## Test Helm chart installation (dry-run)
	@echo "Testing local installation..."
	helm install test-kafka $(CHART_DIR) --dry-run --debug

helm-pull: ## Pull Helm chart from DockerHub
	@echo "Pulling chart from DockerHub..."
	helm pull $(HELM_OCI_REGISTRY)/$(CHART_NAME) --version $(CHART_VERSION)

helm-show: ## Show Helm chart information from DockerHub
	@echo "Showing chart information from DockerHub..."
	helm show chart $(HELM_OCI_REGISTRY)/$(CHART_NAME) --version $(CHART_VERSION)

bump-patch: ## Bump patch version (x.x.X)
	@echo "Bumping patch version..."
	@current_version=$$(grep '^version:' $(CHART_DIR)/Chart.yaml | awk '{print $$2}'); \
	major=$$(echo $$current_version | cut -d. -f1); \
	minor=$$(echo $$current_version | cut -d. -f2); \
	patch=$$(echo $$current_version | cut -d. -f3); \
	new_patch=$$((patch + 1)); \
	new_version="$$major.$$minor.$$new_patch"; \
	sed -i.bak "s/^version: .*/version: $$new_version/" $(CHART_DIR)/Chart.yaml && rm $(CHART_DIR)/Chart.yaml.bak; \
	echo "Version bumped: $$current_version → $$new_version"

bump-minor: ## Bump minor version (x.X.0)
	@echo "Bumping minor version..."
	@current_version=$$(grep '^version:' $(CHART_DIR)/Chart.yaml | awk '{print $$2}'); \
	major=$$(echo $$current_version | cut -d. -f1); \
	minor=$$(echo $$current_version | cut -d. -f2); \
	new_minor=$$((minor + 1)); \
	new_version="$$major.$$new_minor.0"; \
	sed -i.bak "s/^version: .*/version: $$new_version/" $(CHART_DIR)/Chart.yaml && rm $(CHART_DIR)/Chart.yaml.bak; \
	echo "Version bumped: $$current_version → $$new_version"

bump-major: ## Bump major version (X.0.0)
	@echo "Bumping major version..."
	@current_version=$$(grep '^version:' $(CHART_DIR)/Chart.yaml | awk '{print $$2}'); \
	major=$$(echo $$current_version | cut -d. -f1); \
	new_major=$$((major + 1)); \
	new_version="$$new_major.0.0"; \
	sed -i.bak "s/^version: .*/version: $$new_version/" $(CHART_DIR)/Chart.yaml && rm $(CHART_DIR)/Chart.yaml.bak; \
	echo "Version bumped: $$current_version → $$new_version"

clean: helm-clean ## Alias for helm-clean

.DEFAULT_GOAL := help
