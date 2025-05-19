# Development Guide

This document provides comprehensive guidelines for developing the SA-OMF (Phoenix) project.

## Development Environment Setup

### Prerequisites

- Go 1.22 or later
- Docker and Docker Compose
- Make
- Git

### Quick Start

The easiest way to get started is to use our development container:

```bash
# Clone the repository
git clone https://github.com/yourorg/sa-omf.git
cd sa-omf

# Start the development container
docker-compose up -d dev

# Enter the container
docker-compose exec dev bash

# Inside the container, setup and build
make dev-setup
make build
```

Alternatively, use VS Code with the Dev Containers extension and open the project folder.

### Local Setup

If you prefer to develop locally:

```bash
# Set up development environment
make dev-setup

# Build the project
make build

# Run with development configuration
make run
```

## Project Structure

The project follows a standard Go project layout with some additions:

- `cmd/` - Main applications
- `internal/` - Private application code
  - `processor/` - Custom processors
  - `extension/` - Custom extensions
  - `connector/` - Custom connectors
  - `control/` - Control logic
- `pkg/` - Public libraries
- `test/` - Test code and utilities
- `configs/` - Configuration files
- `deploy/` - Deployment configurations
- `docs/` - Documentation
- `scripts/` - Development and CI scripts

## Development Workflow

### 1. Create a New Component

Use our script to generate a new component with proper boilerplate:

```bash
scripts/dev/new-component.sh processor my_processor
```

This creates:
- Factory file
- Config struct
- Implementation file
- Test file

### 2. Running Tests

We support different types of tests:

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Generate test coverage report
make test-coverage
```

### 3. Code Verification

Before submitting changes, verify your code:

```bash
# Run all verification checks
make verify

# Individual checks
make lint
make drift-check
make schema-check
make vendor-check
```

### 4. Docker Builds

The project supports different Docker setups:

```bash
# Build main Docker image
make docker

# Run Docker container with development config
make docker-run

# Run with Docker Compose (multiple services)
docker-compose up collector-development prometheus grafana
```

## Configuration Management

The project uses two primary configuration files:

1. `config.yaml` - Standard OpenTelemetry Collector configuration
2. `policy.yaml` - Self-adaptive behavior configuration

Environment-specific configurations are available in `configs/[environment]/`.

## Working with the Project

### Adding Dependencies

When adding new dependencies, follow these steps to maintain offline build capability:

1. In a network-enabled environment:
   ```bash
   go get <new-dependency>
   go mod tidy
   go mod vendor
   ```

2. Commit the updated vendor directory

### Creating a Release

To create a new release:

```bash
make release VERSION=1.0.0
```

This tags the current commit and pushes the tag to the remote repository.

## Development Best Practices

1. **Testing**: Write both unit and integration tests for all components
2. **Documentation**: Update relevant documentation when changing features
3. **PID Controllers**: When modifying PID controllers, adjust parameters conservatively
4. **Configuration**: Test configuration changes with both default and development configs
5. **Metrics**: Include observability for new components and features

## Troubleshooting

### Common Issues

1. **Build failures**: Ensure Go version 1.22+ is installed
2. **Vendor issues**: Run `go mod vendor` to refresh vendored dependencies
3. **Configuration errors**: Validate config files with `make schema-check`

### Getting Help

If you encounter issues, check:
1. Project documentation in `/docs`
2. OpenTelemetry documentation
3. Open an issue in the GitHub repository