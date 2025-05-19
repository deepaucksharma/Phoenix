# Phoenix Agent Framework

This directory contains the agent framework definitions for the Phoenix project. The agent framework defines roles, responsibilities, workflows, and guidelines for both automated agents and human contributors working on the project.

## Key Documents

| Document | Description |
|----------|-------------|
| [CONSOLIDATED_AGENTS.md](./CONSOLIDATED_AGENTS.md) | **Primary Document** - Comprehensive guide to agent roles, workflows, and decision frameworks |
| [AGENT_RAILS.md](./AGENT_RAILS.md) | Technical guidelines for agent implementation and constraints |
| [AGENTS.md](./AGENTS.md) | Basic role definitions and responsibilities |
| [AGENT_METRICS.md](./AGENT_METRICS.md) | Metrics emitted by agents and monitoring guidelines |

## Agent Role Definitions

This directory also contains YAML files that define each agent role and its permissions. Each YAML file follows the naming pattern `[role-name].yaml` and contains:

- Role description and purpose
- File permissions (can_touch, blocked_files, must_touch)
- Required validation checks
- Other role-specific configurations

The following agent roles are defined:

- **Architect** (`architect.yaml`): Defines system architecture and major design decisions
- **Planner** (`planner.yaml`): Breaks down features into implementable tasks
- **Implementer** (`implementer.yaml`): Writes and tests code
- **Reviewer** (`reviewer.yaml`): Reviews code for quality and correctness
- **Security Auditor** (`security-auditor.yaml`): Evaluates code for security issues
- **Doc Writer** (`doc-writer.yaml`): Creates and updates documentation
- **DevOps** (`devops.yaml`): Manages CI/CD pipelines and deployment
- **Integrator** (`integrator.yaml`): Merges PRs and resolves conflicts
- **Tester** (`tester.yaml`): Creates and maintains tests
- **Dependabot** (`dependabot.yaml`): Handles automated dependency updates

## Usage

When contributing to the Phoenix project, either as a human or an LLM agent:

1. **Identify your role** from the available agent roles
2. **Declare your role** in PRs using `ROLE: role_name`
3. **Reference tasks** using `TASKS: task-id` or `TASKS: N/A`
4. **Follow the guidelines** in the relevant documentation
5. **Respect file permissions** as defined in the role's YAML file

## Adding a New Agent Role

To add a new agent role:

1. Create a new YAML file in this directory using the naming pattern `[role-name].yaml`
2. Define the role's permissions and validation requirements
3. Update the CONSOLIDATED_AGENTS.md file to include the new role
4. Create any role-specific documentation as needed

## CI Integration

The agent role specifications in this directory are used by CI pipelines to validate PRs and enforce role boundaries. The validation ensures that:

1. Each PR has a declared role
2. The role has permission to modify the changed files
3. Required files (must_touch) are included in the changes
4. All role-specific validation checks pass

This enforcement helps maintain code quality and ensures contributors work within their defined responsibilities.