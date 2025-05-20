# Phoenix CI/CD Workflows

This directory contains GitHub workflow configurations for the Phoenix CI/CD pipeline.

## Workflow Structure

The CI/CD pipeline has been organized into separate workflow files to improve maintainability and clarity:

### Main Workflow

- [`workflow.yml`](./workflow.yml): The main meta-workflow that serves as an entry point and triggers the individual specialized workflows.

### Core Workflows

- [`build.yml`](./build.yml): Handles compilation, linting, and Docker image creation.
- [`test.yml`](./test.yml): Runs unit tests, integration tests, generates coverage reports, and conducts benchmarks.
- [`pr-validation.yml`](./pr-validation.yml): Validates pull request title formats and agent role permissions.
- [`scheduled-security-scan.yml`](./scheduled-security-scan.yml): Performs comprehensive security scanning on a scheduled basis.

## Workflow Triggers

Each workflow is configured with the following triggers:

- **Push to main**: Triggers builds, tests, and docker image creation
- **Pull Requests**: Triggers builds, tests, and PR validation
- **Manual**: All workflows can be triggered manually via `workflow_dispatch`
- **Scheduled**: Security scanning runs weekly on Mondays at 16:00 UTC

## Build Process

The build workflow:

1. Checks out the code
2. Sets up Go environment
3. Runs linting
4. Validates configuration schemas 
5. Builds the binary
6. Uploads build artifacts
7. (For main branch) Builds and pushes Docker images

## Testing Process

The test workflow:

1. Runs unit tests with race detection
2. Runs integration tests (on main branch or manual trigger)
3. Checks for code drift
4. Generates and uploads coverage reports
5. (When labeled) Runs benchmarks
6. Performs security scanning with CodeQL

## PR Validation

The PR validation workflow:

1. Validates PR title format using semantic conventions
2. Checks that file changes comply with the agent role specified in the PR

## Agent Role-Based Permissions

PR validation enforces agent role permissions defined in the `/agents/*.yaml` files:

- Each PR must specify a role in the PR body with `ROLE: <!-- rolename -->`
- File changes are checked against the `can_touch` and `blocked_files` patterns
- Agent roles provide specialized permissions (e.g., architects can modify ADRs, implementers can modify code)

## Artifacts

The following artifacts are generated:

- **Binary**: The compiled `sa-omf-otelcol` binary (retained for 7 days)
- **Coverage Report**: HTML coverage report showing test coverage (retained for 7 days)

## Docker Images

For the main branch, Docker images are built and pushed to GitHub Container Registry with tags:

- `latest`: The most recent build from the main branch
- `<commit-sha>`: The specific commit SHA for version tracking

## Manual Triggers

All workflows can be triggered manually via the GitHub Actions UI for testing or troubleshooting purposes.
