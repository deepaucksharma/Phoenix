# AGENT_RAILS.md - Guidelines for LLM Agents

This document defines the guidelines and constraints for autonomous agents working on the Phoenix project. By following these guidelines, agents can collaborate effectively without stepping on each other's toes.

## Agent Roles

Each agent must work within a specific role as defined in `/agents/*.yaml`. The available roles are:

- **architect**: Define high-level architecture (can only modify `docs/adr/**`)
- **planner**: Break down features into tasks (can only modify `tasks/**`)
- **implementer**: Write and test code (cannot modify `docs/adr/**` or `.github/**`)
- **reviewer**: Review code changes (cannot push commits)
- **security-auditor**: Perform security reviews (cannot modify source code)
- **doc-writer**: Update documentation (cannot modify source code)
- **devops**: Maintain CI/CD (can only modify `.github/**` and `deploy/**`)
- **integrator**: Merge PRs (cannot modify source code)

All agents must:
1. Declare their role in PRs (using `ROLE: role_name`)
2. Reference tasks they're working on (using `TASKS: task-id`)
3. Respect the file permissions defined for their role

## Directory Structure

Agents should understand and respect the following directory structure:

```
phoenix/
├── cmd/                 # Entry points for binaries
│   └── sa-omf-otelcol/  # Main collector binary
├── internal/            # Private implementation code
│   ├── processor/       # Data processors
│   ├── extension/       # Collector extensions
│   ├── connector/       # Connectors/exporters
│   ├── control/         # Control algorithms
│   └── interfaces/      # Core interfaces
├── pkg/                 # Reusable packages
│   ├── metrics/         # Metrics definitions
│   ├── util/            # Utility algorithms
│   └── policy/          # Policy schema
├── test/                # Test framework
├── docs/                # Documentation
│   └── adr/             # Architecture Decision Records
├── hack/                # Development scripts
├── deploy/              # Deployment files
├── tasks/               # Task definitions
└── agents/              # Agent role definitions
```

## Adding New Components

To add a new processor, extension, or connector:

1. Call the appropriate script: 
   ```
   hack/new-component.sh processor example_processor
   ```

2. This will:
   - Create appropriate files with boilerplate
   - Register the factory in main.go
   - Create test files

3. Implement the required functionality in the generated files.

## Naming Conventions

- Use snake_case for package names and file names
- Use CamelCase for types and functions
- Use ALL_CAPS for constants
- Prefix metrics with `aemf_`
- Prefix component names with their category (e.g., `pic_control`)

## Testing Requirements

- All code must have unit tests
- Processors must have benchmarks
- Integration tests must be added for new features
- Coverage should be maintained above 60%

## Documentation Requirements

- All public APIs must be documented
- Major design decisions must be recorded in ADRs
- Configuration options must be documented

## Reviewing Code

Reviewer agents should check:

1. Code meets style guidelines
2. Tests are comprehensive
3. Documentation is updated
4. Performance is acceptable
5. Security best practices are followed
6. Code matches the task specification

## Generated Files

Never edit generated files. These include:

- Go generated code (files with `// Code generated ... DO NOT EDIT.`)
- Generated protobuf code
- Build artifacts

## Branch Naming

Use consistent branch naming:

- `feature/description` for new features
- `fix/description` for bug fixes
- `refactor/description` for refactoring
- `docs/description` for documentation changes

## Commit Messages

Follow the Conventional Commits specification:

```
type(scope): description

[optional body]

[optional footer]
```

Where `type` is one of:
- feat: A new feature
- fix: A bug fix
- docs: Documentation changes
- style: Code style changes (formatting, etc.)
- refactor: Code refactoring
- test: Adding or modifying tests
- chore: Routine tasks, maintenance

## Getting Help

If you're unsure about something, refer to existing code as examples, check the ADRs, or ask for clarification in the PR description.
