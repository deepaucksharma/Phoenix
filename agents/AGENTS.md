# Phoenix – Agent Role Framework

This document defines the roles, responsibilities, and boundaries for agents and human contributors working on the Phoenix project. These roles and their associated permissions (defined in their respective `.yaml` files in this directory) apply project-wide and are enforced by CI checks. By following these guidelines, we can maintain code quality and consistency while enabling parallel development.

## Role Catalogue

| Role ID | Purpose | Responsibilities | Outputs | Boundaries |
|---------|---------|------------------|---------|------------|
| **Architect** | Define system architecture | High-level design decisions | ADRs, architecture diagrams | Focus on system design, not implementation details |
| **Planner** | Break down features into tasks | Issue triage, task decomposition | `tasks/*.yaml` files, roadmap updates | Does not write production code |
| **Implementer** | Write and test code | Code implementation, unit testing | Code, tests, small documentation updates | Works on assigned tasks only |
| **Reviewer** | Ensure code quality | Code review, validation | Review comments, approval/rejection | Does not push commits (except minor fixes) |
| **Security-Auditor** | Identify security issues | Security scanning, vulnerability assessment | Security reports, PR comments | Does not modify source code directly |
| **Doc-Writer** | Create/update documentation | Technical writing, API documentation | Markdown files, diagrams, examples | Restricted to documentation areas |
| **DevOps** | CI/CD, deployment | Pipeline maintenance, deployment scripts | Workflow files, scripts, Dockerfile updates | Focused on operational aspects |
| **Integrator** | Merge PRs, resolve conflicts | Conflict resolution, release management | Merged PRs, release notes | Only merges approved PRs |
| **Tester** | Create and maintain tests | Test development, validation framework | Test files, quality reports | Limited to test files, cannot modify core implementation |
| **Dependabot** | Dependency management | Update dependencies | PR for dependency updates | Limited to package files, automated dependency updates only |

## Workflow

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
   - **Dependabot** keeps dependencies up to date

## Collaboration Guidelines

- Each PR must have a declared role and reference to task(s)
- All changes must pass automated checks appropriate for the role
- Respect the boundaries defined for each role
- Use the template scripts in `/scripts` for generating new components
- Follow the code structure and style guidelines

## Technical Standards

- Go version: 1.24+
- Code style: Follow `golangci-lint` rules
- Commit style: [Conventional Commits](https://www.conventionalcommits.org/)
- PR process: Create branch → Implement → CI checks → Review → Merge

## Metrics and KPIs

The team's progress is tracked using these metrics:
- PR cycle time (creation to merge)
- Test coverage percentage
- Open issues count
- Documentation completeness

An automated dashboard tracks these metrics in real-time.

## Documentation

All major design decisions must be documented in Architecture Decision Records (ADRs) in the `docs/architecture/adr` directory.

## Tooling

Use provided scripts in the `scripts` directory:
- `scripts/dev/new-component.sh` - Create a new component with proper boilerplate
- `scripts/dev/new-adr.sh` - Create a new ADR
- `scripts/dev/create-branch.sh` - Create a new branch with proper naming
- `scripts/dev/create-task.sh` - Create a new task
- `scripts/dev/validate-task.sh` - Validate a task specification

## Further Information

For more detailed information about agent roles, workflows, and decision frameworks, please refer to:

- [CONSOLIDATED_AGENTS.md](./CONSOLIDATED_AGENTS.md) - Comprehensive guide with all roles, workflows, and decision frameworks
- [AGENT_RAILS.md](./AGENT_RAILS.md) - Technical guidelines for agent implementation
- [AGENT_METRICS.md](./AGENT_METRICS.md) - Metrics emitted by agents and monitoring guidelines