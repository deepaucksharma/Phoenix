# Changelog

All notable changes to the Phoenix (SA-OMF) project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Enhanced PID controller with oscillation detection and circuit breaking
- Added Bayesian optimization for multi-dimensional parameter spaces
- Added Policy-In-Code (PIC) control extension for configuration governance
- Added comprehensive PID controller documentation and tuning guidelines
- Added benchmarking CI workflow for continuous performance monitoring
- Enhanced PR template with better integration
- Added comprehensive development guide and documentation
- Added Docker Compose setup for development environment
- Added Dev Container configuration for VS Code
- Consolidated GitHub Actions workflows into a single workflow

### Changed
- Simplified metrics package to remove OpenTelemetry Collector dependencies
- Made PID controller components standalone and reusable
- Cleaned up and consolidated project files
- Updated Dependabot configuration to include dedicated security updates
- Updated GoReleaser configuration with better version information
- Updated CODEOWNERS file with appropriate roles
- Enhanced Makefile with better offline support and documentation
- Standardized Go version to 1.22 across all workflows and Docker
- Improved offline build documentation

### Fixed
- Fixed low-pass filtering for derivative term to reduce noise sensitivity
- Fixed Space-Saving algorithm for accurate frequency tracking
- Fixed thread safety issues in controller code
- Removed Windows-based GitHub workflows to reduce CI load
- Fixed Docker image build process with proper versioning
- Ensured consistent vendored dependencies usage

### Security
- Added resource usage monitoring with automatic safe mode
- Added rate limiting for configuration changes
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