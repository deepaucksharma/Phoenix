# SA-OMF Testing Framework

This directory contains a comprehensive testing framework for the SA-OMF (Phoenix) project. The test structure follows best practices outlined in the [Validation Framework](../docs/testing/validation-framework.md) and includes enhancements to support the self-adapting nature of the system.

## Directory Structure

- `unit/` - Tests for core algorithms and utility components
  - `hll/` - HyperLogLog algorithm tests
  - `reservoir/` - Reservoir sampling algorithm tests
  - `topk/` - Space-Saving algorithm tests
  - `pid/` - PID controller tests
  - `causality/` - Causality detection algorithm tests
  - `timeseries/` - Time series analysis tests
  - `metrics/` - Metrics utilities tests
  - `policy/` - Policy validation tests

- `benchmarks/` - Performance and benchmarking tests
  - `algorithms/` - Algorithm performance tests
  - `component/` - Component benchmark tests

- `e2e/` - End-to-end tests
  - `integration/` - Integration tests for component interactions
  - `benchmarks/` - End-to-end performance benchmarks

- `interfaces/` - Interface contract tests

- `processors/` - Tests for specific metric processors
  - `adaptive_pid/` - Tests for the PID decision processor
  - `templates/` - Shared test templates for processors

- `extensions/` - Tests for custom extensions

- `testutils/` - Shared testing utilities
  - `metrics_generator.go` - Generates test metrics
  - `metrics_helper.go` - Helpers for testing with metrics
  - `pid_helper.go` - Helpers for testing PID controllers

- `generator/` - Test data generation utilities

- `chaos/` - Chaos testing framework for system resilience

## Running Tests

### All Tests

To run all tests:

```bash
make test
```

### Specific Test Types

```bash
# Unit tests only
make test-unit

# Integration tests only
make test-integration

# Performance benchmarks
make benchmark
```

### Targeted Testing

```bash
# Test specific components
go test -v ./test/processors/adaptive_pid/...
go test -v ./test/unit/hll/...

# Run e2e tests
go test -v ./test/e2e/...

# Run benchmarks for specific algorithms
go test -v ./test/benchmarks/algorithms/... -bench=.
```

## Test Coverage

Generate test coverage reports:

```bash
make test-coverage
```

## Writing New Tests

### Test Design Principles

1. **Isolation**: Each test should be independent and not rely on other tests
2. **Determinism**: Tests should produce the same results on every run
3. **Comprehensive**: Cover normal, edge, and error cases
4. **Performance-Aware**: Include benchmarks for performance-critical components

### Testing Dynamic Configuration

All processors supporting dynamic configuration should use the appropriate test utilities:

```go
import "github.com/yourorg/sa-omf/test/interfaces"

func TestMyProcessor(t *testing.T) {
    processor := createTestProcessor(t)
    // Test dynamic configuration
}
```

### Test Data Generation

Use the utilities in the `generator` and `testutils` packages to create consistent test data:

```go
import "github.com/yourorg/sa-omf/test/testutils"

metrics := testutils.GenerateTestMetrics(10, 5) // 10 resources, 5 metrics each
```

## Test Templates

Processor tests should follow the templates in `processors/templates/` to ensure consistency across all processor implementations.