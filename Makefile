.PHONY: all build run test clean lint verify docker release help

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

# Setup build flags
GO_BUILD_FLAGS?=-mod=vendor -v
GO_TEST_FLAGS?=-mod=vendor -v
GO_OFFLINE_ENV?=GO111MODULE=on GOPROXY=off GOSUMDB=off

# Create directories if they don't exist
$(shell mkdir -p $(BIN_DIR))

# Default target
all: build

# Build the collector
build:
	@echo "Building SA-OMF OpenTelemetry Collector..."
	@$(GO_OFFLINE_ENV) go build $(GO_BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "Binary built at $(BIN_DIR)/$(BINARY_NAME)"

# Run the collector
run:
	@echo "Running SA-OMF with config: $(CONFIG)"
	@$(GO_OFFLINE_ENV) go run $(GO_BUILD_FLAGS) -ldflags "$(LDFLAGS)" ./cmd/$(BINARY_NAME)/main.go --config=$(CONFIG)

# Run built binary directly
run-bin:
	@echo "Running binary with config: $(CONFIG)"
	@$(BIN_DIR)/$(BINARY_NAME) --config=$(CONFIG)

# Run tests
test:
	@echo "Running tests..."
	@$(GO_OFFLINE_ENV) go test $(GO_TEST_FLAGS) ./...

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

# Run linting
lint:
	@echo "Running linter..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2; \
	fi
	@golangci-lint run --timeout=5m --config=.golangci.yml ./...

# Ensure vendor directory exists
vendor:
	@echo "Creating/updating vendor directory..."
	@go mod vendor
	@echo "Vendor directory ready"

# Check schemas validity
schema-check:
	@echo "Validating policy schemas..."
	@bash scripts/validation/validate_policy_schema.sh

# Verify project
verify: lint vendor schema-check test-unit
	@echo "All verification checks passed"

# Security check
security-check:
	@echo "Running security checks..."
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Docker commands
docker:
	@echo "Building Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-f deploy/docker/Dockerfile .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-run:
	@echo "Running Docker container: $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker run --rm -it \
		-p 8888:8888 \
		-p 13133:13133 \
		-v $(PWD)/configs:/etc/sa-omf \
		$(DOCKER_IMAGE):$(DOCKER_TAG) \
		--config=/etc/sa-omf/development/config.yaml

docker-compose:
	@echo "Running Docker Compose stack..."
	@docker-compose up -d

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

# Setup development environment
dev-setup:
	@echo "Setting up development environment..."
	@bash scripts/setup/setup-offline-build.sh || true
	@go mod vendor
	@echo "Development environment set up"

# Show help
help:
	@echo "SA-OMF Makefile Commands"
	@echo "======================="
	@echo "Core Commands:"
	@echo "  make                       - Build the collector"
	@echo "  make run [CONFIG=path]     - Run the collector with specified config"
	@echo "  make run-bin [CONFIG=path] - Run built binary with specified config"
	@echo "  make test                  - Run all tests"
	@echo "  make clean                 - Clean build artifacts"
	@echo ""
	@echo "Development Commands:"
	@echo "  make lint                  - Run linter"
	@echo "  make verify                - Run all verification checks"
	@echo "  make vendor                - Create/update vendor directory"
	@echo "  make dev-setup             - Set up development environment"
	@echo "  make security-check        - Run security checks"
	@echo ""
	@echo "Test Commands:"
	@echo "  make test-unit             - Run unit tests only"
	@echo "  make test-integration      - Run integration tests only"
	@echo "  make test-coverage         - Run tests with coverage report"
	@echo "  make benchmark             - Run performance benchmarks"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker                - Build Docker image"
	@echo "  make docker-run            - Run the collector in Docker"
	@echo "  make docker-compose        - Run the full Docker Compose stack"
	@echo ""
	@echo "Release Command:"
	@echo "  make release VERSION=x.y.z - Create a release tag"