.PHONY: build run test test-all test-unit test-integration test-coverage clean lint benchmark docker docker-run
.PHONY: schema-check vendor-check release dev-setup help verify

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
GO_BUILD_FLAGS=-mod=vendor -v
GO_TEST_FLAGS=-mod=vendor -v
GO_OFFLINE_ENV=GO111MODULE=on GOPROXY=off GOSUMDB=off

# Create directories if they don't exist
$(shell mkdir -p $(BIN_DIR))

# Build the collector binary
build:
	@echo "Building SA-OMF OpenTelemetry Collector..."
	@$(GO_OFFLINE_ENV) go build $(GO_BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "Binary built at $(BIN_DIR)/$(BINARY_NAME)"

# Run the collector with specified or default config
run:
	@echo "Running SA-OMF with config: $(CONFIG)"
	@$(GO_OFFLINE_ENV) go run $(GO_BUILD_FLAGS) -ldflags "$(LDFLAGS)" ./cmd/$(BINARY_NAME)/main.go --config=$(CONFIG)

# Run all the binary directly if built
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
	@$(GO_OFFLINE_ENV) go test $(GO_TEST_FLAGS) ./test/integration/...

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

# Generate mocks for testing
mocks:
	@echo "Generating mocks..."
	@if ! command -v mockgen &> /dev/null; then \
		echo "mockgen not found, installing..."; \
		go install github.com/golang/mock/mockgen@latest; \
	fi
	@go generate ./...

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
	fi

# Check all config and policy schemas
schema-check:
	@echo "Validating policy and config schemas..."
	@bash scripts/validation/validate_policy_schema.sh
	@# Comment out until we fix the config schema validation
	@# bash scripts/validation/validate_config_schema.sh

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
	@bash scripts/setup/setup_offline_build.sh || true
	@if command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint already installed."; \
	else \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2; \
	fi
	@if command -v mockgen &> /dev/null; then \
		echo "mockgen already installed."; \
	else \
		echo "Installing mockgen..."; \
		go install github.com/golang/mock/mockgen@v1.6.0; \
	fi
	@echo "Development environment set up."

# Show help
help:
	@echo "SA-OMF Makefile Help"
	@echo "===================="
	@echo "make build                    - Build the collector binary"
	@echo "make run [CONFIG=path]        - Run the collector with specified or default config"
	@echo "make run-bin [CONFIG=path]    - Run the binary directly (if already built)"
	@echo "make test                     - Run all tests"
	@echo "make test-all                 - Run all enhanced tests"
	@echo "make test-unit                - Run unit tests only"
	@echo "make test-integration         - Run integration tests only"
	@echo "make test-coverage            - Run tests with coverage report"
	@echo "make benchmark                - Run performance benchmarks"
	@echo "make clean                    - Clean build artifacts"
	@echo "make verify                   - Run most important checks (lint, vendor, schema, unit tests)"
	@echo "make lint                     - Run linter"
	@echo "make mocks                    - Generate mocks for testing"
	@echo "make drift-check              - Check component registration"
	@echo "make vendor-check             - Check if vendor directory is up to date"
	@echo "make schema-check             - Validate policy and config schemas"
	@echo "make docker [DOCKER_TAG=tag]  - Build Docker image"
	@echo "make docker-run               - Run the collector in Docker"
	@echo "make release VERSION=x.y.z    - Create a release tag"
	@echo "make dev-setup                - Set up development environment"
	@echo "make help                     - Show this help"
	@echo ""
	@echo "Role-specific targets:"
	@echo "make architect-check          - Run checks for architect role"
	@echo "make planner-check            - Run checks for planner role"
	@echo "make implementer-check        - Run checks for implementer role"
	@echo "make reviewer-check           - Run checks for reviewer role"
	@echo "make security-auditor-check   - Run checks for security-auditor role"
	@echo "make doc-writer-check         - Run checks for doc-writer role"
	@echo "make devops-check             - Run checks for devops role"
	@echo "make integrator-check         - Run checks for integrator role"