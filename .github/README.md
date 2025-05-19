# GitHub Workflows

This directory contains the GitHub Actions workflows and configuration for the Phoenix project.

## Workflows

- **CI (ci.yml)**: Runs tests, builds, and validates components for PRs and main branch commits
- **Agent Role Validation (agent-role.yml)**: Validates that PR changes adhere to the declared agent role
- **CodeQL Analysis (codeql-analysis.yml)**: Security scanning for code vulnerabilities
- **PR Labeler (pr-labeler.yml)**: Automatically labels PRs based on files changed
- **Update Task State (update-task-state.yml)**: Updates task state when PRs are merged

## Configuration Files

- **dependabot.yml**: Configuration for automated dependency updates
- **mergify.yml**: Rules for automatic PR merging
- **labeler.yml**: Rules for automatic PR labeling
- **PULL_REQUEST_TEMPLATE.md**: Template for new PRs

## Agent Roles

The project uses an agent role system to manage permissions and responsibilities. Agent role definitions are stored in `docs/agents/`.

Each PR must declare a role using the format `` `ROLE: role_name` `` in the PR description. The agent-role-validation workflow enforces that changes in the PR are allowed for the declared role.

## Dependabot Integration

Dependabot PRs are automatically assigned the `dependabot` role and are configured for automatic merging when checks pass.

## Troubleshooting

If the CI or agent role validation is failing:

1. Check that your PR properly declares a role using the correct format
2. Verify that your changes are allowed for the declared role
3. Ensure your changes follow the project style and pass all relevant tests
