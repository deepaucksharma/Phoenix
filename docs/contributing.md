# Contributing to Phoenix

Thank you for considering contributing to the Phoenix project! This document outlines the process for contributing and the standards we expect.

## Development Environment

- Go 1.22 or higher
- Docker for containerized testing
- Make for build automation

## Project Structure

The project is organized as follows:
- `cmd/`: Application entrypoints
- `configs/`: Configuration files
- `deploy/`: Deployment resources
- `docs/`: Documentation
- `internal/`: Internal packages
- `pkg/`: Public packages
- `scripts/`: Helper scripts
- `test/`: Test code
- `tasks/`: Task definitions

See the [README.md](../README.md) for more detailed information about the repository structure.

## Development Workflow

1. First, set up your development environment:
   ```bash
   # Clone the repository
   git clone https://github.com/yourorg/sa-omf.git
   cd sa-omf
   
   # Set up development environment
   make dev-setup
   ```

2. Pick a task from the `tasks/` directory or create a new one
   ```bash
   ./scripts/dev/create-task.sh "Your task description"
   ```

3. Create a feature branch from `main`
   ```bash
   ./scripts/dev/create-branch.sh feature "Your feature description"
   ```

4. Implement your changes
   
5. Verify your changes:
   ```bash
   # Run most important checks
   make verify
   
   # Or run individual checks
   make lint
   make test
   make drift-check
   make schema-check
   ```

6. Submit a Pull Request

7. Address review comments

## Agent-Based Workflow

Phoenix uses a structured agent-based workflow where contributors take on specific roles. See [Agent Roles](./agents/README.md) for details on roles and responsibilities.

## Pull Request Process

1. Ensure your PR has a clear title and description
2. Fill in the PR template completely including:
   - Your role (`ROLE:`)
   - Tasks you're addressing (`TASKS:`)
3. Make sure CI checks pass
4. Request review from appropriate team members
5. Address all feedback
6. Maintain a clean commit history
7. Rebase onto main before merge if needed

## Coding Standards

- Follow Go best practices
- Run `gofmt` and `golangci-lint` on your code
- Write comprehensive tests
- Maintain backward compatibility when possible
- Document public APIs
- Add appropriate logging and metrics

## Vendoring Dependencies

When adding or updating Go dependencies, the project relies on a vendored
`vendor/` directory to ensure offline builds work consistently. After you modify
`go.mod` or run `go get`, execute:

```bash
go mod tidy
go mod vendor
```

Commit the resulting `vendor/` folder along with your `go.mod` and `go.sum`
changes so that CI and other developers use the same dependency set.

## Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
type(scope): description

[optional body]

[optional footer]
```

Examples:
- `feat(pid): add anti-windup mechanism to controller`
- `fix(topk): correct buffer overflow in space-saving algorithm`
- `docs(readme): update installation instructions`

## Testing

All code should be thoroughly tested. At minimum:

- Unit tests for all functionality
- Integration tests for component interactions
- End-to-end tests for critical paths
- Benchmark tests for performance-sensitive code

## Documentation

Update documentation when changing functionality:

- Code comments (including package documentation)
- README updates
- ADRs for significant design decisions
- Configuration examples

## Development with Docker

For a consistent development environment, you can use Docker:

```bash
# Start the development container
docker-compose up -d dev

# Enter the container
docker-compose exec dev bash

# Run commands inside the container
make build
make test
```

For more information, see the [Development Guide](./development-guide.md).

## License

All contributions will be licensed under the Apache License 2.0 as found in the LICENSE file.

## Code of Conduct

Please follow our Code of Conduct in all interactions with the project.

## Questions?

If you have any questions about contributing, please open an issue or contact the project maintainers.
