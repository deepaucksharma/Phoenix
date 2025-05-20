# Benchmarks Directory

This directory contains performance benchmarks for the SA-OMF components.

## Running Benchmarks

Benchmarks can be run using:

```bash
# Run all benchmarks
make benchmark

# Run specific benchmarks
go test -bench=. -benchmem ./test/benchmarks/...
```

## Adding New Benchmarks

When adding new benchmarks:

1. Place benchmark files in appropriate subdirectories matching the component structure
2. Use the naming convention `component_benchmark_test.go`
3. Follow Go's standard benchmarking patterns using the `testing.B` type

## Benchmark Categories

- **Processors**: Benchmarks for individual processors
- **Algorithms**: Benchmarks for core algorithms like HLL, TopK, etc.
- **Performance**: End-to-end performance tests for typical workflows
