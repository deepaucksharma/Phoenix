# Phoenix Consolidated Scripts

This directory contains all shell scripts used in the Phoenix project, organized by category for easy discovery and maintenance.

## Directory Structure

### 📦 Core Operations (`core/`)
Essential scripts for running and managing the Phoenix system
- `run-phoenix.sh` - Main system startup/shutdown script
- `initialize-environment.sh` - Environment setup and initialization

### 🚀 Deployment (`deployment/`)
Scripts for deploying and configuring Phoenix
- `deploy.sh` - Deployment orchestration script
- `generate_certs.sh` - TLS certificate generation
- `test-deployment.sh` - Deployment validation

### 🧪 Testing (`testing/`)
All testing and validation scripts
- `verify-services.sh` - Service availability verification
- `verify-apis.sh` - API endpoint testing
- `verify-configs.sh` - Configuration validation
- `full-verification.sh` - Complete system verification
- `test_core_functionality.sh` - Core functionality integration tests
- `functional-test.sh` - Functional testing suite
- `api-test.sh` - API testing utilities

### 📊 Monitoring (`monitoring/`)
Scripts for monitoring and observability
- `health_check_aggregator.sh` - Health check aggregation
- `validate-system.sh` - System validation checks

### 🔧 Maintenance (`maintenance/`)
Scripts for system maintenance and operations
- `cleanup.sh` - System cleanup operations
- `backup_phoenix_data.sh` - Data backup utilities
- `restore_phoenix_data.sh` - Data restoration utilities

### 🛠️ Utilities (`utils/`)
General purpose utility scripts
- `show-docs.sh` - Documentation display utilities
- `project-summary.sh` - Project summary generation
- `phoenix-metric-generator.sh` - Metric generation utilities
- `newrelic-integration.sh` - New Relic integration utilities

### 📁 Legacy (`legacy/`)
Older scripts and control plane components
- `control-loop-enhanced.sh` - Enhanced control loop (legacy)
- `update-control-file.sh` - Control file updates (legacy)
- `migrate-to-monorepo.sh` - Migration utilities (legacy)

## Usage

### Quick Operations
```bash
# Start Phoenix system
./scripts/consolidated/core/run-phoenix.sh

# Initialize environment
./scripts/consolidated/core/initialize-environment.sh

# Run full verification
./scripts/consolidated/testing/full-verification.sh

# Deploy system
./scripts/consolidated/deployment/deploy.sh
```

### Testing
```bash
# Run all tests
./scripts/consolidated/testing/full-verification.sh

# Test specific components
./scripts/consolidated/testing/verify-services.sh
./scripts/consolidated/testing/verify-apis.sh
./scripts/consolidated/testing/verify-configs.sh
```

### Maintenance
```bash
# System cleanup
./scripts/consolidated/maintenance/cleanup.sh

# Backup data
./scripts/consolidated/maintenance/backup_phoenix_data.sh

# Health checks
./scripts/consolidated/monitoring/health_check_aggregator.sh
```

## Script Categories

| Category | Count | Purpose |
|----------|-------|---------|
| Core | 2 | Essential system operations |
| Deployment | 3 | System deployment and configuration |
| Testing | 7 | Testing and validation |
| Monitoring | 2 | System monitoring and health |
| Maintenance | 3 | System maintenance and data management |
| Utils | 4 | General utilities and integrations |
| Legacy | 3 | Older scripts and migration tools |

## Making Scripts Executable

All scripts are made executable during the consolidation process. If you need to make them executable manually:

```bash
# Make all scripts executable
find scripts/consolidated -name "*.sh" -exec chmod +x {} \;

# Make specific category executable
chmod +x scripts/consolidated/testing/*.sh
```

## Script Dependencies

Most scripts assume:
- Docker and docker-compose are installed
- Working directory is the Phoenix project root
- Environment is properly initialized (via initialize-environment.sh)

## Maintenance Notes

- All scripts maintain backward compatibility with existing references
- Original locations are preserved with symbolic links
- Script consolidation improves discoverability and maintainability
- Regular updates should check both new location and original references