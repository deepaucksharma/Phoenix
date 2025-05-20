# End-to-End Tests

This directory contains end-to-end tests for the Phoenix (SA-OMF) project.

## Directory Structure

- `/test/e2e/benchmarks/` - End-to-end benchmarks for measuring performance
- `/test/e2e/integration/` - Integration tests for verifying component interactions
- All end-to-end tests reside under `/test/e2e/`. The former `/test/e2e_tests/` directory has been removed.

## Running the Tests

### Benchmarks

```bash
go test -v ./test/e2e/benchmarks/... -bench=.
```

### Integration Tests

```bash
go test -v ./test/e2e/integration/...
```

## Adding New Tests

When adding new end-to-end tests:

1. Place benchmark tests in the `/test/e2e/benchmarks/` directory
2. Place integration tests in the `/test/e2e/integration/` directory
3. Follow the naming convention: `<component_name>_<test_type>_test.go`
4. Include proper documentation in the test file

