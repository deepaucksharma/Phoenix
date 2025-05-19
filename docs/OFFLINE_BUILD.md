# Offline Building

The Phoenix project is configured to support offline builds in network-restricted environments through vendored dependencies. This document explains how to build and run the project without internet access.

## Quick Start

The easiest way to get started with offline building is:

```bash
# Run the setup script to configure environment
./setup_offline_build.sh

# Build the project
make build
```

## Vendored Dependencies

All required Go modules are included in the `vendor/` directory, eliminating the need for network access during builds. This vendoring approach offers several benefits:

- **Air-gapped environments**: Build and run in completely isolated networks
- **Consistent builds**: Always use exactly the same dependency versions
- **Faster builds**: No need to download dependencies during build time
- **Long-term stability**: Protection against upstream dependency changes or removals

## Environment Variables

All build commands in the Makefile set the following environment variables:

```bash
GO111MODULE=on    # Use Go modules mode
GOPROXY=off       # Disable module proxy (force local dependencies)
GOSUMDB=off       # Disable checksum database (no network checks)
```

## Build Flags

The Makefile uses these flags for all Go commands:

```bash
-mod=vendor       # Use vendored dependencies
```

## Verifying Vendor Status

To verify that your vendor directory is up to date with go.mod:

```bash
make vendor-check
```

## Adding New Dependencies

When adding new dependencies, follow these steps to maintain offline capability:

1. In a network-enabled environment, run:
   ```bash
   go get <new-dependency>
   go mod tidy
   go mod vendor
   ```
2. Commit the updated vendor directory with git

Always rerun `go mod vendor` and commit the `vendor/` folder any time
`go.mod` or `go.sum` changes so that offline builds stay reproducible.

## Go Version Requirements

This project requires Go 1.22 or higher. The setup script checks for a compatible Go version automatically. You can download a suitable Go version from [golang.org/dl/](https://golang.org/dl/) and install it before running in offline mode.

## Docker Support

The project's Docker builds also use vendored dependencies. When building Docker images:

```bash
make docker
```

The Dockerfile copies the entire project, including the vendor directory, to build without network access.

## CI/CD Integration

All CI workflows are configured to use the vendored dependencies with the `-mod=vendor` flag, ensuring consistent builds in all environments. This means CI pipelines can run with minimal network access requirements.