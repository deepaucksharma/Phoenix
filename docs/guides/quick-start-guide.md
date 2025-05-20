# Phoenix Quick Start Guide

This guide will help you get started with Phoenix (SA-OMF) quickly using the streamlined build process.

## Prerequisites

Before you begin, ensure you have the following installed:

- Go 1.24 or higher
- Git
- Docker (optional, for containerized development)
- VS Code (optional, for dev container workflow)

## Clone the Repository

```bash
git clone https://github.com/deepaucksharma/Phoenix.git
cd Phoenix
```

## Development Approaches

Phoenix supports multiple development approaches. Choose the one that best fits your workflow:

### Option 1: Direct Development (Fastest Start)

This approach works directly on your local machine:

```bash
# Set up development environment (install required tools)
make dev-setup

# Build the project quickly (good for iterative development)
make fast-build

# Run with development config
make fast-run
```

### Option 2: Docker-based Development

This approach uses Docker for a consistent environment:

```bash
# Start a development container
make docker-dev

# Connect to the container
docker-compose exec dev bash

# Inside the container, build and run
make fast-build
make fast-run
```

### Option 3: VS Code Dev Container (Recommended for New Developers)

This approach provides a fully configured development environment:

1. Open the project in VS Code
2. Install the Dev Containers extension if not already installed
3. Click "Reopen in Container" when prompted
4. Inside the container, build and run:
   ```bash
   make fast-build
   make fast-run
   ```

## Hot Reload Development (Fastest Development Cycle)

For maximum development speed, use hot reload to automatically rebuild and restart on code changes:

```bash
# Start hot reload server
make hot-reload
```

Your changes will automatically be detected, rebuilt, and the application restarted.

## Common Tasks

### Building the Project

```bash
# Fast build for development (no vendor directory)
make fast-build

# Production build with vendor directory
make build
```

### Running the Project

```bash
# Run with default development config
make fast-run

# Run with specific config
make fast-run CONFIG=configs/production/config.yaml

# Run the built binary directly
make run-bin CONFIG=configs/custom/config.yaml
```

### Testing

```bash
# Run all tests
make test

# Run unit tests only (faster)
make test-unit

# Run integration tests
make test-integration

# Run with test coverage
make test-coverage
```

### Linting and Verification

```bash
# Run linter
make lint

# Run full verification (lint, schema check, tests)
make verify
```

### Docker Tasks

```bash
# Build Docker image
make docker

# Run in Docker
make docker-run

# Run tests in Docker
make docker-test
```

## Configuration

Phoenix uses two primary configuration files:

1. **config.yaml**: Defines the pipeline structure (receivers, processors, exporters)
2. **policy.yaml**: Defines adaptive behavior parameters (PID controllers, targets, limits)

Sample configs can be found in the `configs/` directory:
- `configs/default/` - Balanced configuration for general use
- `configs/development/` - Verbose logging, faster adaptation for development
- `configs/production/` - Conservative settings for stability in production

## Getting Help

```bash
# Show all available make commands
make help
```

## Next Steps

- Read the [Development Guide](../development-guide.md) for more detailed instructions
- Explore the [Configuration Reference](../configuration-reference.md) to understand configuration options
- Review [Current Architecture](../architecture/CURRENT_STATE.md) to understand the system design
- Check [Adaptive Processing Concepts](../concepts/adaptive-processing.md) to learn the core concepts