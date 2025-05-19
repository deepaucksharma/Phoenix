# CI/CD Workflows

This document describes the CI/CD workflow used in the Phoenix project.

## Overview

The Phoenix project uses GitHub Actions for continuous integration and deployment. The workflow is designed to be efficient, reliable, and maintainable.

## Workflow Structure

The project uses a single comprehensive workflow file (`.github/workflows/workflow.yml`) that contains multiple jobs:

1. **Build & Test**
   - Handles linting, testing, and building the binary
   - Runs on every push to main and on pull requests
   - Skips integration tests on PRs for faster feedback
   - Uploads build artifacts on main branch pushes

2. **Security Scan**
   - Runs security scanning using CodeQL
   - Executes weekly, on main branch pushes, and when manually triggered
   - Identifies potential security vulnerabilities

3. **Benchmarks**
   - Runs performance benchmarks
   - Only executes when manually triggered or when a PR has the "run-benchmarks" label

4. **PR Validation**
   - Validates pull request metadata
   - Enforces semantic PR titles
   - Checks agent role permissions when applicable
   - Only runs on pull requests

5. **Docker**
   - Builds and pushes Docker images
   - Only runs on main branch pushes
   - Creates multi-architecture images (amd64 and arm64)

## Key Features

- **Vendored Dependencies**: All workflows use the vendored dependencies with `-mod=vendor` flag to support offline builds
- **Conditional Testing**: Integration tests are skipped for PRs to speed up feedback
- **Optimized Caching**: Go module caching is enabled to speed up builds
- **Concurrent Job Management**: Avoids redundant builds by canceling in-progress runs
- **Docker Multi-platform**: Builds for both amd64 and arm64 architectures

## Usage

### Running Benchmarks

To run benchmarks for a PR:
1. Add the `run-benchmarks` label to the PR
2. The benchmarks job will automatically execute

### Manual Workflow Execution

Both the CI and Security workflows can be manually triggered through the GitHub Actions UI using the "workflow_dispatch" event.

### Agent Roles

For PRs from agent roles, add the role information to the PR description:
```
ROLE: <!-- role_name -->
```

The PR validation workflow will check if the changed files comply with the role's permissions defined in the corresponding agent YAML file.