# Phoenix Agent Framework

This document provides a comprehensive guide to the agent roles, responsibilities, and workflows used in the Phoenix project. This framework enables efficient collaboration between both autonomous agents and human contributors.

## Agent Roles and Responsibilities

| Role ID | Purpose | Responsibilities | Outputs | Permissions | Boundaries |
|---------|---------|------------------|---------|------------|------------|
| **Architect** | Define system architecture | High-level design decisions, architecture planning | ADRs, architecture diagrams | Can modify architecture docs | Focus on system design, not implementation details |
| **Planner** | Break down features into tasks | Issue triage, task decomposition | `tasks/*.yaml` files, roadmap updates | Can modify task files | Does not write production code |
| **Implementer** | Write and test code | Code implementation, unit testing | Code, tests | Can modify code and tests | Works on assigned tasks only, cannot modify ADRs |
| **Reviewer** | Ensure code quality | Code review, validation | Review comments, approval/rejection | Cannot push commits (except minor fixes) | Focus on code quality assessment |
| **Security-Auditor** | Identify security issues | Security scanning, vulnerability assessment | Security reports | Cannot modify source code | Limited to security analysis |
| **Doc-Writer** | Create/update documentation | Technical writing, API documentation | Markdown files, diagrams | Limited to documentation files | Cannot modify source code |
| **DevOps** | CI/CD, deployment | Pipeline maintenance, deployment scripts | Workflow files, Dockerfiles | Limited to CI/CD files | Focused on operational aspects |
| **Integrator** | Merge PRs, resolve conflicts | Conflict resolution, release management | Merged PRs, release notes | Can merge PRs | Only merges approved PRs |
| **Tester** | Create and maintain tests | Test development, validation framework | Test files, quality reports | Limited to test files | Cannot modify core implementation |
| **Dependabot** | Dependency management | Update dependencies | PR for dependency updates | Limited to package files | Automated dependency updates only |

## File Permission Boundaries

Each agent role has specific file permissions defined in their respective YAML files:

- **can_touch**: Files that the agent is allowed to modify
- **blocked_files**: Files that the agent is not allowed to modify
- **must_touch**: Files that the agent must modify as part of their work

These permissions use glob patterns and apply project-wide. For example:

```yaml
# Implementer permissions
can_touch:
  - cmd/**
  - internal/**
  - pkg/**
  - test/**
must_touch:
  - test/**
blocked_files:
  - agents/**
  - .github/workflows/**
  - docs/adr/**
```

## Workflow and Collaboration

### Development Process

1. **Creation Phase**:
   - **Architect** defines the architecture or changes through ADRs
   - **Planner** breaks down features into tasks and updates the task list

2. **Implementation Phase**:
   - **Implementer** works on assigned tasks
   - **Tester** creates or updates related tests
   - **Reviewer** validates the implementation
   - **Security-Auditor** checks for vulnerabilities

3. **Documentation & Integration Phase**:
   - **Doc-Writer** updates documentation
   - **DevOps** ensures CI/CD works correctly
   - **Integrator** merges approved PRs

### PR Process

All contributors (human or agent) must:

1. Declare their role in PRs (using `ROLE: role_name`)
2. Reference tasks they're working on (using `TASKS: task-id` or `TASKS: N/A`)
3. Respect the file permissions defined for their role
4. Ensure all required checks pass before merging

### Branch Naming Conventions

Use consistent branch naming:

- `feature/description` for new features
- `fix/description` for bug fixes
- `refactor/description` for refactoring
- `docs/description` for documentation changes

### Commit Message Standards

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

## Technical Standards

- Go version: 1.24+
- Code style: Follow `golangci-lint` rules
- Test coverage: Maintain above 60%
- Documentation: All public APIs must be documented
- PR process: Create branch → Implement → CI checks → Review → Merge

## Repository Structure

Agents should understand and respect the following repository structure:

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
│   └── architecture/    # Architecture documentation
│     └── adr/           # Architecture Decision Records
├── scripts/             # Development scripts
├── deploy/              # Deployment files
├── tasks/               # Task definitions
└── agents/              # Agent role definitions
```

## Adding New Components

To add a new processor, extension, or connector:

1. Call the appropriate script: 
   ```
   scripts/dev/new-component.sh processor example_processor
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

## Metrics and KPIs

The team's progress is tracked using these metrics:
- PR cycle time (creation to merge)
- Test coverage percentage
- Open issues count
- Documentation completeness

## Tooling

Use provided scripts in the `scripts` directory:
- `scripts/dev/new-component.sh` - Create a new component with proper boilerplate
- `scripts/dev/new-adr.sh` - Create a new ADR
- `scripts/dev/create-branch.sh` - Create a new branch with proper naming
- `scripts/dev/create-task.sh` - Create a new task
- `scripts/dev/validate-task.sh` - Validate a task specification

## Generated Files

Never edit generated files. These include:

- Go generated code (files with `// Code generated ... DO NOT EDIT.`)
- Generated protobuf code
- Build artifacts

## Getting Help

If you're unsure about something:
- Refer to existing code as examples
- Check the ADRs
- Consult the Phoenix project documentation
- Ask for clarification in PR descriptions