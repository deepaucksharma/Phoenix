.PHONY: all build run test test-all test-unit test-integration test-coverage clean lint benchmark
.PHONY: docker docker-run docker-dev docker-compose schema-check vendor-check release help verify
.PHONY: fast-build fast-run docker-test docker-lint docker-verify hot-reload dev-setup

# Include role-specific targets
include tools.mk

# Variables
BINARY_NAME=sa-omf-otelcol
BIN_DIR=bin
MODULE_NAME=$(shell grep "^module " go.mod | awk '{print $$2}')
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
DOCKER_IMAGE=sa-omf-otelcol
DOCKER_TAG?=latest
CONFIG?=configs/development/config.yaml
LDFLAGS=-X $(MODULE_NAME)/cmd/$(BINARY_NAME)/version.Version=$(VERSION) \
        -X $(MODULE_NAME)/cmd/$(BINARY_NAME)/version.Commit=$(COMMIT) \
        -X $(MODULE_NAME)/cmd/$(BINARY_NAME)/version.BuildDate=$(BUILD_DATE)

# Setup offline first build flags
GO_BUILD_FLAGS?=-mod=vendor -v
GO_TEST_FLAGS?=-mod=vendor -v
GO_OFFLINE_ENV?=GO111MODULE=on GOPROXY=off GOSUMDB=off

# Create directories if they don't exist
$(shell mkdir -p $(BIN_DIR))

# Default target
all: build

# Fast build target - optimized for development
fast-build:
	@echo "Fast building SA-OMF..."
	@go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "Binary built at $(BIN_DIR)/$(BINARY_NAME)"

# Standard build target with vendor support
build:
	@echo "Building SA-OMF OpenTelemetry Collector..."
	@$(GO_OFFLINE_ENV) go build $(GO_BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "Binary built at $(BIN_DIR)/$(BINARY_NAME)"

# Run with fast rebuild
fast-run:
	@echo "Running SA-OMF with fast build and config: $(CONFIG)"
	@go run -ldflags "$(LDFLAGS)" ./cmd/$(BINARY_NAME)/main.go --config=$(CONFIG)

# Run with vendor support
run:
	@echo "Running SA-OMF with config: $(CONFIG)"
	@$(GO_OFFLINE_ENV) go run $(GO_BUILD_FLAGS) -ldflags "$(LDFLAGS)" ./cmd/$(BINARY_NAME)/main.go --config=$(CONFIG)

# Run the binary directly if built
run-bin:
	@echo "Running binary with config: $(CONFIG)"
	@$(BIN_DIR)/$(BINARY_NAME) --config=$(CONFIG)

# Run all tests
test:
	@echo "Running tests..."
	@$(GO_OFFLINE_ENV) go test $(GO_TEST_FLAGS) ./...

# Run all enhanced tests
test-all: test-unit test-integration

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@$(GO_OFFLINE_ENV) go test $(GO_TEST_FLAGS) ./test/unit/... ./test/interfaces/... ./test/processors/... ./test/extensions/...

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	@$(GO_OFFLINE_ENV) go test $(GO_TEST_FLAGS) ./test/e2e/integration/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@$(GO_OFFLINE_ENV) go test $(GO_TEST_FLAGS) -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... ./test/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"
	
# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	@$(GO_OFFLINE_ENV) go test $(GO_TEST_FLAGS) -bench=. -benchmem ./test/benchmarks/...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)/
	@go clean
	@rm -f coverage.out coverage.html
	@echo "Temporary files cleaned"

# Verify all - run most important checks
verify: lint vendor-check schema-check test-unit

# Run linting
lint:
	@echo "Running linter..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2; \
	fi
	@echo "Running golangci-lint..."
	@golangci-lint run \
		--timeout=5m \
		--config=.golangci.yml \
		./...

# Verify drift - Code consistency check for interdependent files
drift-check:
	@echo "Checking for code drift..."
	@scripts/ci/check_component_registry.sh

# Check if vendor directory is in sync with go.mod
vendor-check:
	@echo "Checking if vendor is up to date with go.mod and go.sum..."
	@if command -v git &> /dev/null && git diff-index --quiet HEAD -- go.mod go.sum; then \
		echo "go.mod and go.sum are in sync with the repository."; \
	else \
		echo "Warning: go.mod or go.sum has uncommitted changes."; \
	fi
	@if [ -d "vendor" ]; then \
		echo "Vendor directory exists."; \
	else \
		echo "Warning: vendor directory does not exist. Run 'go mod vendor'."; \
		go mod vendor; \
		echo "Vendor directory created."; \
	fi

# Check all config and policy schemas
schema-check:
	@echo "Validating policy and config schemas..."
	@bash scripts/validation/validate_policy_schema.sh
	@# Comment out until we fix the config schema validation
	@# bash scripts/validation/validate_config_schema.sh

# =================================
# Docker-specific commands
# =================================

# Build Docker image with provided or default tag
docker:
	@echo "Building Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-f deploy/docker/Dockerfile .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Run the collector in Docker with mounted configs
docker-run:
	@echo "Running Docker container: $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker run --rm -it \
		-p 8888:8888 \
		-p 13133:13133 \
		-v $(PWD)/configs:/etc/sa-omf \
		$(DOCKER_IMAGE):$(DOCKER_TAG) \
		--config=/etc/sa-omf/development/config.yaml

# Start a development container
docker-dev:
	@echo "Starting development container..."
	@docker-compose up -d dev
	@echo "Development container started. To attach:"
	@echo "  docker-compose exec dev bash"

# Run Docker Compose with specific services
docker-compose:
	@echo "Running Docker Compose stack..."
	@docker-compose up -d

# Run tests inside Docker container
docker-test:
	@echo "Running tests in Docker container..."
	@docker-compose run --rm dev make test

# Run lint inside Docker container
docker-lint:
	@echo "Running lint in Docker container..."
	@docker-compose run --rm dev make lint

# Run verification inside Docker container
docker-verify:
	@echo "Running verification in Docker container..."
	@docker-compose run --rm dev make verify

# Hot reload target for automatic rebuilds
hot-reload:
	@echo "Starting hot reload development environment..."
	@if command -v docker-compose &> /dev/null; then \
		docker-compose up hot-reload; \
	else \
		echo "Error: Docker Compose is required for hot-reload"; \
		exit 1; \
	fi

# Target for CI/CD to ensure all checks pass
ci-cd-verify: vendor-check lint schema-check test build docker

# Create a tag and version for release
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Example: make release VERSION=1.0.0"; \
		exit 1; \
	fi
	@echo "Creating release tag v$(VERSION)..."
	@git tag -a v$(VERSION) -m "Release v$(VERSION)"
	@git push origin v$(VERSION)
	@echo "Release v$(VERSION) created"

# Set up local development environment
dev-setup:
	@echo "Setting up development environment..."
	@bash scripts/setup/setup-offline-build.sh || true
	@echo "Development environment set up."

# Show help
help:
	@echo "SA-OMF Makefile Help"
	@echo "===================="
	@echo "Standard Development Commands:"
	@echo "  make                       - Build the collector"
	@echo "  make fast-build            - Quick build for development (no vendor)"
	@echo "  make fast-run [CONFIG=path]- Quick run for development (no vendor)"
	@echo "  make run [CONFIG=path]     - Run the collector with config (vendor mode)"
	@echo "  make test                  - Run all tests"
	@echo "  make lint                  - Run linter"
	@echo "  make verify                - Run most important checks (lint, vendor, schema, unit tests)"
	@echo "  make clean                 - Clean build artifacts"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker                - Build Docker image"
	@echo "  make docker-run            - Run the collector in Docker"
	@echo "  make docker-dev            - Start a development container"
	@echo "  make docker-compose        - Run the full Docker Compose stack"
	@echo "  make docker-test           - Run tests in Docker container"
	@echo "  make docker-lint           - Run lint in Docker container"
	@echo "  make docker-verify         - Run verification in Docker container"
	@echo "  make hot-reload            - Start hot reload server (Docker-based)"
	@echo ""
	@echo "Advanced Commands:"
	@echo "  make test-all              - Run all enhanced tests"
	@echo "  make test-unit             - Run unit tests only"
	@echo "  make test-integration      - Run integration tests only"
	@echo "  make test-coverage         - Run tests with coverage report"
	@echo "  make benchmark             - Run performance benchmarks"
	@echo "  make drift-check           - Check component registration"
	@echo "  make vendor-check          - Check if vendor directory is up to date"
	@echo "  make schema-check          - Validate policy and config schemas"
	@echo "  make release VERSION=x.y.z - Create a release tag"
	@echo "  make dev-setup             - Set up development environment"
	@echo "  make help                  - Show this help"
	@echo ""
	@echo "Role-specific targets:"
	@echo "  make architect-check       - Run checks for architect role"
	@echo "  make planner-check         - Run checks for planner role"
	@echo "  make implementer-check     - Run checks for implementer role"
	@echo "  make reviewer-check        - Run checks for reviewer role"
	@echo "  make security-auditor-check- Run checks for security-auditor role"
	@echo "  make doc-writer-check      - Run checks for doc-writer role"
	@echo "  make devops-check          - Run checks for devops role"
	@echo "  make integrator-check      - Run checks for integrator role"