# Contributing to Phoenix

Thank you for considering contributing to the Phoenix project! This document outlines the process for contributing and the standards we expect.

## Development Environment

- Go 1.21 or higher
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

See the [README.md](README.md) for more detailed information about the repository structure.

## Development Workflow

1. Pick a task from the `tasks/` directory or create a new one
   ```bash
   ./scripts/dev/create-task.sh "Your task description"
   ```
2. Create a feature branch from `main`
   ```bash
   ./scripts/dev/create-branch.sh feature "Your feature description"
   ```
3. Implement your changes
4. Run tests locally (`make test`)
5. Run linting (`make lint`)
6. Submit a Pull Request
7. Address review comments

## Agent-Based Workflow

Phoenix uses a structured agent-based workflow where contributors take on specific roles. See `docs/agents/AGENTS.md` for details on roles and responsibilities.

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

## License

All contributions will be licensed under the Apache License 2.0 as found in the LICENSE file.

## Code of Conduct

Please follow our Code of Conduct in all interactions with the project.

## Questions?

If you have any questions about contributing, please open an issue or contact the project maintainers.
