.PHONY: build test test-all test-unit test-integration test-coverage clean lint benchmark

include tools.mk

# Build the collector binary
build:
	@echo "Building SA-OMF OpenTelemetry Collector..."
	@go build -o bin/sa-omf-otelcol ./cmd/sa-omf-otelcol

# Run standard tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run all enhanced tests
test-all: test-unit test-integration

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@go test -v ./test/unit/... ./test/interfaces/... ./test/processors/... ./test/extensions/...

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	@go test -v ./test/integration/...

# Verify drift - Code consistency check for interdependent files
drift-check:
	@echo "Checking for code drift..."
	@hack/check_component_registry.sh
	@go mod tidy

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... ./test/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"
	
# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./test/benchmark/...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

# Run linting
lint:
	@echo "Running linter..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@golangci-lint run ./...

# Generate mocks for testing
mocks:
	@echo "Generating mocks..."
	@if ! command -v mockgen &> /dev/null; then \
		echo "mockgen not found, installing..."; \
		go install github.com/golang/mock/mockgen@latest; \
	fi
	@go generate ./...

# Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -t sa-omf-otelcol:latest -f deploy/docker/Dockerfile .

# Run collector with default config
run:
	@echo "Running SA-OMF with default config..."
	@go run ./cmd/sa-omf-otelcol/main.go --config=config/config.yaml

# Create a tag and version for release
release:
	@echo "Creating release tag v$(VERSION)..."
	@git tag -a v$(VERSION) -m "Release v$(VERSION)"
	@git push origin v$(VERSION)
	@echo "Release v$(VERSION) created"

# Show help
help:
	@echo "SA-OMF Makefile Help"
	@echo "===================="
	@echo "make build          - Build the collector binary"
	@echo "make test           - Run all standard tests"
	@echo "make test-all       - Run all enhanced tests"
	@echo "make test-unit      - Run unit tests only"
	@echo "make test-integration - Run integration tests only"
	@echo "make test-coverage  - Run tests with coverage report"
	@echo "make benchmark      - Run performance benchmarks"
	@echo "make clean          - Clean build artifacts"
	@echo "make lint           - Run linter"
	@echo "make mocks          - Generate mocks for testing"
	@echo "make docker         - Build Docker image"
	@echo "make run            - Run collector with default config"
	@echo "make release VERSION=x.y.z - Create a release tag"
	@echo "make help           - Show this help"
