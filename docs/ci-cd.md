# CI/CD Workflows

This document describes the CI/CD workflows used in the Phoenix project.

## Overview

The Phoenix project uses GitHub Actions for continuous integration and deployment. The workflows are designed to be efficient, reliable, and maintainable.

## Workflow Structure

The project uses three main workflow files:

1. **CI Workflow** (`.github/workflows/ci.yml`)
   - Handles the core build, test, and deployment processes
   - Runs on every push to main and on pull requests
   - Contains three jobs:
     - **build**: Runs tests, linting, and builds the binary
     - **benchmarks**: Runs performance benchmarks (only when triggered)
     - **docker**: Builds and pushes Docker images (only for main branch)

2. **Security Workflow** (`.github/workflows/security.yml`)
   - Runs security scanning using CodeQL
   - Executes weekly and on code changes
   - Identifies potential security vulnerabilities

3. **PR Validation Workflow** (`.github/workflows/pr-validation.yml`)
   - Validates pull request metadata
   - Enforces semantic PR titles
   - Checks agent role permissions when applicable

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