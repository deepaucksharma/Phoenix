# Phoenix Testing Overview

This directory contains resources for validating the Phoenix project.

## Benchmarks

Benchmarks reside under `test/benchmark/` and cover both component-level and
end-to-end performance scenarios. Cached metric generation speeds up repeated
runs, and optional CPU profiling can be enabled by setting the `BENCH_PROFILE`
environment variable.

Run all benchmarks:

```bash
make benchmark
```

Or target a specific package:

```bash
go test -bench=. -benchmem ./test/benchmark/...
```

## Chaos and Security Testing

Chaos scenarios and security review scripts live in `test/chaos/`.

- `chaos_suite.go` executes disruptive scenarios such as configuration
  oscillation and resource starvation.
- `security_review.sh` runs `gosec` and `govulncheck` to highlight potential
  vulnerabilities.

The chaos suite includes several scenarios:

- Configuration oscillation
- Process explosion
- Cardinality bomb
- Resource starvation
- Network partition
- Out-of-memory stress

Execute the suite with optional environment and duration flags (defaults shown):

```bash
go run test/chaos/chaos_suite.go --env docker --duration 30m
```

Run a security review:

```bash
bash test/chaos/security_review.sh
```

For more detail on the overall framework see
[validation-framework.md](./validation-framework.md).
