# Phoenix Build Process Guide

This document provides a comprehensive guide to the Phoenix build process, explaining the various build commands, options, and best practices.

## Build System Overview

Phoenix uses a Make-based build system with carefully designed targets for different development scenarios. The build system is optimized for:

1. **Development Speed**: Fast builds for quick iteration
2. **Offline Builds**: Support for environments without internet access
3. **Containerization**: Easy Docker-based building and testing
4. **Reliability**: Consistent builds across environments

## Makefile Structure

The Makefile is organized into several sections:

- **Core Build Targets**: Basic build and run commands
- **Test Targets**: Commands for different test scenarios
- **Docker Targets**: Container-based builds and runs
- **Utility Targets**: Helper commands for development
- **Role-specific Targets**: Commands for different dev roles

## Build Commands Explained

### Core Build Commands

#### `make build`

The standard build command creates a production-ready binary using vendored dependencies:

```bash
make build
```

This command:
- Uses `-mod=vendor` for offline building
- Sets version information via LDFLAGS
- Outputs the binary to `bin/sa-omf-otelcol`

#### `make fast-build`

Optimized for development speed, this command builds without vendor directory:

```bash
make fast-build
```

This command:
- Skips vendor dependency checks for faster builds
- Still sets version information
- Ideal for iterative development

### Running Commands

#### `make run`

Runs the collector with the vendor directory:

```bash
make run [CONFIG=path/to/config.yaml]
```

- Default config: `configs/development/config.yaml`
- Override with CONFIG parameter

#### `make fast-run`

Builds and runs in one step, optimized for development:

```bash
make fast-run [CONFIG=path/to/config.yaml]
```

- Skips vendor dependency checks
- Faster startup for development
- Default config: `configs/development/config.yaml`

#### `make run-bin`

Runs the pre-built binary directly:

```bash
make run-bin [CONFIG=path/to/config.yaml]
```

- Skips build step entirely
- Requires binary to be already built
- Fastest startup if binary exists

### Docker Build Commands

#### `make docker`

Builds a Docker image:

```bash
make docker [DOCKER_TAG=mytag]
```

- Default tag: `latest`
- Uses multi-stage build for smaller images
- Sets version information via build args

#### `make docker-run`

Runs the collector in Docker:

```bash
make docker-run
```

- Mounts configs for easy configuration change
- Exposes ports 8888 and 13133

#### `make hot-reload`

Provides hot-reload functionality for rapid development:

```bash
make hot-reload
```

- Automatically rebuilds on code changes
- Restarts the application immediately
- Configured via `.air.toml`

## Offline Building

Phoenix supports fully offline builds for air-gapped environments:

### Preparation for Offline Building

```bash
# Run once to set up vendor directory
make dev-setup

# Verify vendor directory is complete
make vendor-check
```

### Offline Build Process

```bash
# Build using vendor directory
make build

# Run tests offline
make test
```

The build system uses these offline-build flags:
```
GO_OFFLINE_ENV=GO111MODULE=on GOPROXY=off GOSUMDB=off
```

## Test Commands

Phoenix provides a rich set of test commands:

### Basic Test Commands

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Run with test coverage
make test-coverage
```

### Advanced Test Commands

```bash
# Run benchmarks
make benchmark

# Run tests in Docker
make docker-test

# Run specialized role-based tests
make implementer-check
make security-auditor-check
```

## Verification Commands

Phoenix includes several commands to verify code quality:

```bash
# Run full verification
make verify

# Run linter only
make lint

# Check for schema validity
make schema-check

# Check for code drift between interdependent files
make drift-check
```

## Build Pipeline Customization

### Changing the Default Config

You can specify a different config file:

```bash
make run CONFIG=configs/custom/my-config.yaml
```

### Building with Different Tags

```bash
make docker DOCKER_TAG=v1.0.0
```

### Build Environment Variables

You can override these environment variables:

- `VERSION`: Version string (default: git describe)
- `COMMIT`: Commit hash (default: git rev-parse)
- `BUILD_DATE`: Build date (default: current date)
- `CONFIG`: Config file path (default: configs/development/config.yaml)
- `DOCKER_TAG`: Docker image tag (default: latest)

## Best Practices

1. **Development Workflow**:
   - Use `make fast-build` and `make fast-run` during development
   - Use `make hot-reload` for maximum iteration speed
   - Use `make verify` before submitting changes

2. **Testing Strategy**:
   - Run `make test-unit` frequently during development
   - Run `make test` before commits
   - Run `make test-coverage` to identify untested code

3. **Docker Workflow**:
   - Use `make docker-dev` to start a development container
   - Use `docker-compose exec dev bash` to connect to the container
   - Run make commands inside the container

4. **Role-Specific Workflows**:
   - Architects: Use `make architect-check` to validate architectural decisions
   - Security Auditors: Use `make security-auditor-check` for security testing
   - Implementers: Use `make implementer-check` for code quality checks
   - Integrators: Use `make integrator-check` for comprehensive verification

## Common Issues and Solutions

### Vendor Directory Issues

**Problem**: Build fails with module errors
**Solution**: Update vendor directory with:
```bash
go mod tidy
go mod vendor
```

### Docker Build Issues

**Problem**: Docker build fails
**Solution**: Check Docker daemon is running and try:
```bash
make clean
docker system prune -f
make docker
```

### Test Failures

**Problem**: Tests failing with timeout errors
**Solution**: Run specific test with increased timeout:
```bash
cd test && go test -timeout 5m ./unit/...
```

## Build Directory Structure

The build process creates these directories:

- `bin/`: Contains the compiled binary
- `.gocache/`: Local Go build cache
- `vendor/`: Vendored dependencies
- `.air-tmp/`: Temporary files for hot reload

## Conclusion

The Phoenix build system provides a flexible, powerful set of commands for different development scenarios. By using the appropriate make targets, you can optimize your workflow for speed, reliability, or completeness as needed.

For a quick reference, use `make help` to see all available commands.