# Test Makefile for the Phoenix project
# This Makefile helps running various tests for the Phoenix project

.PHONY: all test unit integration clean coverage report benchmark

# Default target
all: test

# Run all tests
test: unit integration

# Run unit tests
unit:
	@echo "Running unit tests..."
	go test -v ./unit/...
	go test -v ./interfaces/...
	go test -v ./processors/...
	go test -v ./extensions/...

# Run integration tests
integration:
	@echo "Running integration tests..."
	go test -v ./integration/...

# Clean test artifacts
clean:
	@echo "Cleaning test artifacts..."
	rm -f coverage.out
	rm -f coverage.html
	rm -rf ./tmp

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out -covermode=atomic ./unit/... ./interfaces/... ./processors/... ./extensions/... ./integration/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

# Generate test report
report: coverage
	@echo "Generating test report..."
	go run ../cmd/report-generator/main.go -coverage coverage.out -output test-report

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./benchmark/...
	go test -bench=. -benchmem ./unit/...