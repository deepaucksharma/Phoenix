# Changelog

All notable changes to the Phoenix (SA-OMF) project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Added benchmarking CI workflow for continuous performance monitoring
- Added agent definition for tester role
- Added agent-specific issue template
- Enhanced PR template with better agent role integration
- Added comprehensive development guide and documentation
- Added Docker Compose setup for development environment
- Added Dev Container configuration for VS Code
- Consolidated GitHub Actions workflows into a single workflow

### Changed
- Updated Dependabot configuration to include dedicated security updates
- Updated GoReleaser configuration with better version information
- Updated CODEOWNERS file to include tester role
- Improved task creation script with more options and role validation
- Enhanced Makefile with better offline support and documentation
- Standardized Go version to 1.22 across all workflows and Docker
- Improved offline build documentation

### Fixed
- Removed Windows-based GitHub tasks to reduce CI load
- Fixed Docker image build process with proper versioning
- Ensured consistent vendored dependencies usage

### Security
- Improved Docker security with non-root users

## [0.1.0] - 2025-05-19

### Added
- Initial implementation of PID controller
- Basic processor framework with UpdateableProcessor interface
- Adaptive TopK processor
- Priority tagging processor
- CI pipeline with testing and linting
- Documentation structure and initial ADRs

### Changed

### Fixed

### Security