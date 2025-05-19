.PHONY: build test clean lint

include tools.mk

# Build the collector binary
build:
	@echo "Building SA-OMF OpenTelemetry Collector..."
	@go build -o bin/sa-omf-otelcol ./cmd/sa-omf-otelcol

# Run all tests
test:
	@echo "Running tests..."
	@go test -v ./...

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
	@go run ./cmd/sa-omf-otelcol/main.go --config=deploy/config.yaml

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
	@echo "make build    - Build the collector binary"
	@echo "make test     - Run all tests"
	@echo "make clean    - Clean build artifacts"
	@echo "make lint     - Run linter"
	@echo "make mocks    - Generate mocks for testing"
	@echo "make docker   - Build Docker image"
	@echo "make run      - Run collector with default config"
	@echo "make release VERSION=x.y.z - Create a release tag"
	@echo "make help     - Show this help"
