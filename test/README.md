# SA-OMF Testing Framework

This directory contains a comprehensive testing framework for the SA-OMF (Phoenix) project. The test structure follows best practices outlined in the [Validation Framework](../docs/testing/validation-framework.md) and includes enhancements to support the self-adapting nature of the system.

## Directory Structure

- `unit/` - Tests for core algorithms and utility components
  - `hll/` - HyperLogLog algorithm tests
  - `reservoir/` - Reservoir sampling algorithm tests
  - `topk/` - Space-Saving algorithm tests
  - `pid/` - PID controller tests
- `interfaces/` - Interface contract tests (e.g., UpdateableProcessor)
- `processors/` - Tests for specific metric processors
  - `adaptive_pid/` - Tests for the PID decision processor
  - `adaptive_topk/` - Tests for the adaptive TopK processor
  - `prioritytagger/` - Tests for the priority tagging processor
- `integration/` - End-to-end and integration tests
- `testutils/` - Shared testing utilities
- `benchmarks/` - Performance and benchmarking tests

## Running Tests

### All Tests

To run all tests:

```bash
make test
```

### Unit Tests Only

To run only unit tests:

```bash
go test -v ./test/unit/...
```

### Processor Tests

To run tests for specific processors:

```bash
go test -v ./test/processors/...
```

### Integration Tests

To run integration tests (requires Docker):

```bash
go test -v ./test/integration/...
```

### Benchmarks

To run performance benchmarks:

```bash
go test -v ./test/benchmark/... -bench=.
```

## Writing New Tests

### Adding a New Unit Test

1. Create a new test file in the appropriate directory.
2. Import necessary testing packages.
3. Write test functions following the Go testing conventions.
4. Use testutils for common functionality.

### Testing UpdateableProcessor Components

All processors implementing the UpdateableProcessor interface should use the common test utilities in the `interfaces` package:

```go
import "github.com/yourorg/sa-omf/test/interfaces"

func TestMyProcessor(t *testing.T) {
    processor := createTestProcessor(t)
    interfaces.TestUpdateableProcessor(t, processor)
}
```

### Writing Integration Tests

Integration tests should:

1. Set up a complete mini-environment.
2. Generate synthetic workload metrics.
3. Verify that the system adapts correctly.
4. Check metrics and configuration changes.

## Test Coverage

Generate test coverage reports:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Data

Test data generators and sample metrics are available in the `testutils` package.