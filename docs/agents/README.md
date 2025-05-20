# Phoenix Agent Framework

This document serves as an entry point to the Phoenix Agent Framework documentation.

## Overview

The Phoenix Agent Framework defines roles, responsibilities, workflows, and guidelines for both automated agents and human contributors working on the project. The framework enables efficient collaboration between different contributors and ensures that each works within their boundaries.

## Documentation Structure

| Document | Description |
|----------|-------------|
| [Agent Roles and Responsibilities](/agents/AGENTS.md) | Basic role definitions and responsibilities |
| [Comprehensive Agent Framework](/agents/CONSOLIDATED_AGENTS.md) | **Primary Document** - Detailed guide to agent roles, workflows, and decision frameworks |
| [Technical Guidelines](/agents/AGENT_RAILS.md) | Technical guidelines for agent implementation and constraints |
| [Agent Metrics](/agents/AGENT_METRICS.md) | Metrics emitted by agents and monitoring guidelines |

## Agent Role Definitions

The framework includes the following agent roles:

- **Architect**: Defines system architecture and major design decisions
- **Planner**: Breaks down features into implementable tasks
- **Implementer**: Writes and tests code
- **Reviewer**: Reviews code for quality and correctness
- **Security Auditor**: Evaluates code for security issues
- **Doc Writer**: Creates and updates documentation
- **DevOps**: Manages CI/CD pipelines and deployment
- **Integrator**: Merges PRs and resolves conflicts
- **Tester**: Creates and maintains tests
- **Dependabot**: Handles automated dependency updates

## Using the Agent Framework

When contributing to the Phoenix project, either as a human or an LLM agent:

1. **Identify your role** from the available agent roles
2. **Declare your role** in PRs using `ROLE: role_name`
3. **Reference tasks** using `TASKS: task-id` or `TASKS: N/A`
4. **Follow the guidelines** in the relevant documentation
5. **Respect file permissions** as defined in the role's YAML file

## Agent Configuration Files

The agent role definitions are stored as YAML files in the `/agents` directory. Each YAML file defines:

- Role description and purpose
- File permissions (can_touch, blocked_files, must_touch)
- Required validation checks
- Other role-specific configurations
