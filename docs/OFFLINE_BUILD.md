# Offline Building

The Phoenix project is configured to support offline builds in network-restricted environments through vendored dependencies.

## Vendored Dependencies

All required Go modules are included in the `vendor/` directory, eliminating the need for network access during builds.

## Building Offline

To build the project without internet access:

```bash
# All builds automatically use vendor directory
make build
```

## Adding New Dependencies

When adding new dependencies:

1. In a network-enabled environment, run:
   ```bash
   go get <new-dependency>
   go mod tidy
   go mod vendor
   ```
2. Commit the updated vendor directory

Always rerun `go mod vendor` and commit the `vendor/` folder any time
`go.mod` or `go.sum` changes so that offline builds stay reproducible.

## CI/CD Integration

All CI workflows are configured to use the vendored dependencies, ensuring consistent builds in all environments.