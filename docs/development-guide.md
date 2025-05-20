# Phoenix Development Guide

This guide provides comprehensive instructions for developing Phoenix (Self-Aware OpenTelemetry Metrics Fabric).

## Quick Start

The fastest way to get started is to use our universal build script:

```bash
# Build and run in one step (local mode)
./build.sh run

# Build and run with hot reload (Docker mode)
./build.sh --hot-reload

# Run tests (local mode)
./build.sh --test

# For more options
./build.sh --help
```

## Development Environments

Phoenix offers three development approaches to suit your preferences:

### 1. VS Code Dev Container (Recommended)

The most consistent development experience with all tools pre-configured:

1. Install [VS Code](https://code.visualstudio.com/) and the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Open the repository in VS Code
3. Click "Reopen in Container" when prompted
4. The container will set up all dependencies automatically

### 2. Docker-based Development

Use Docker without VS Code integration:

```bash
# Start development container
make docker-dev

# Or with the enhanced docker-compose
docker-compose -f docker-compose.enhanced.yml up -d dev

# Connect to the running container
docker-compose exec dev bash

# When inside container, build and run
make fast-build
make fast-run
```

### 3. Local Development

For developers who prefer to work directly on their host machine:

```bash
# Set up development environment
make dev-setup

# Build the project
make fast-build

# Run with development config
make fast-run
```

## Hot Reload Development

For rapid iteration with automatic rebuilds on code changes:

```bash
# Start the hot reload server
docker-compose -f docker-compose.enhanced.yml up hot-reload

# Or use the build script
./build.sh --hot-reload
```

The hot reload server will automatically rebuild and restart the application when code changes are detected.

## Testing

Phoenix includes comprehensive tests to ensure code quality:

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Run tests with coverage
make test-coverage

# Run benchmarks
make benchmark
```

You can also run tests in Docker:

```bash
# Run tests in Docker
make docker-test

# Or with build script
./build.sh --docker --test
```

## Linting and Verification

Ensure code quality with the following commands:

```bash
# Run linter
make lint

# Run full verification (lint, schema check, unit tests)
make verify

# Run specific role-based checks
make implementer-check  # For implementer role
```

## Docker Workflow

The project includes comprehensive Docker support:

```bash
# Build Docker image
make docker

# Run in Docker
make docker-run

# Run the full stack with Docker Compose
docker-compose -f docker-compose.enhanced.yml up

# Build for multiple architectures
make multi-arch-docker
```

## Directory Structure

```
sa-omf/
├── cmd/                    # Main application entrypoints
├── configs/                # Configuration files
├── internal/               # Internal packages
│   ├── interfaces/         # Core interfaces
│   ├── extension/          # Extensions implementation
│   ├── connector/          # Connectors implementation
│   ├── processor/          # Processors implementation
│   └── control/            # Control logic helpers
├── pkg/                    # Public API packages
├── test/                   # Test code and utilities
├── deploy/                 # Deployment configurations
└── docs/                   # Documentation
```

## Configuration Management

The system uses two main types of configuration files:

1. **config.yaml**: Standard OpenTelemetry Collector configuration
   - Defines receivers, processors, exporters, and pipelines
   - Sets up component connections

2. **policy.yaml**: Self-adaptive behavior configuration
   - Defines KPIs and target values
   - Configures PID controller parameters
   - Sets safety thresholds and limits

## Common Development Tasks

### Adding a New Processor

1. Create a directory in `internal/processor/yourprocessor/`
2. Implement the `UpdateableProcessor` interface
3. Create factory, config, and processor files
4. Add appropriate tests in `test/processors/yourprocessor/`
5. Register your processor in the collector factory

### Modifying PID Controllers

PID controllers are defined in the policy.yaml file:
- Adjust `kp`, `ki`, `kd` values to change control behavior
- Set `target_value` to define the desired KPI state
- Configure `hysteresis_percent` to prevent oscillation
- Set `integral_windup_limit` to prevent integral term growth

## Troubleshooting

### Common Issues

1. **Build Failures**
   - Run `make clean` and try again
   - Check if `vendor` directory is up to date with `make vendor-check`
   - Try building in Docker with `./build.sh --docker`

2. **Test Failures**
   - Check if recent code changes might affect other components
   - Run specific test file: `go test -v ./path/to/failing/test`
   - Check test environment configuration

3. **Docker Issues**
   - Ensure Docker is running and has sufficient resources
   - Reset Docker environment: `docker-compose down -v && docker-compose up`

## Additional Resources

- [Architecture Documentation](architecture/README.md)
- [Component Documentation](components/README.md)
- [Configuration Reference](configuration-reference.md)
- [Concept Documentation](concepts/README.md)
- [CI/CD Pipeline](ci-cd.md)

## Development Best Practices

1. **Follow Go Conventions**: Use `gofmt`, follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
2. **Write Tests**: Maintain high test coverage for new code
3. **Document Changes**: Update relevant documentation for significant changes
4. **Use Interfaces**: Favor dependency injection and interfaces for testability
5. **Validate Configs**: Run schema validation for configuration changes
6. **Run Verification**: Always run `make verify` before submitting changes