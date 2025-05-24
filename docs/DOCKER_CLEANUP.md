# Docker Infrastructure Cleanup

## Overview
This document explains the cleanup of legacy Docker Compose files and service directories completed on 2025-05-24.

## What Was Removed

### 1. Legacy Docker Compose Files
**Location**: `infrastructure/docker/compose/`
- `base.yaml` - Unused base configuration
- `dev.yaml` - Development overrides with incorrect service names
- `docker-compose.modular.yaml` - Unused modular configuration
- `docker-compose.yaml` - Outdated version conflicting with root

**Issues**:
- Referenced non-existent service paths
- Used old service names (collector, observer, actuator)
- Incorrect relative paths (../../../)
- Missing current services (anomaly-detector, benchmark-controller)

### 2. Legacy Service Directories
**Removed**:
- `services/collector/` - Replaced by OpenTelemetry Collector images
- `services/control-plane/actuator/` - Replaced by `apps/control-actuator-go`
- `services/control-plane/observer/` - Replaced by OpenTelemetry Collector config
- `services/generators/complex/` - Scripts moved to `scripts/consolidated/`

## Current Structure

### Canonical Docker Compose
The **only** Docker Compose file is now at the root: `docker-compose.yaml`

### Service Locations
- **Go Applications**: `apps/` directory
  - `apps/control-actuator-go` - Control actuator service
  - `apps/anomaly-detector` - Anomaly detection service
- **Services**: `services/` directory  
  - `services/benchmark` - Benchmark controller
  - `services/generators/synthetic` - Synthetic metrics generator

### Docker Commands
All Docker operations should use the root `docker-compose.yaml`:
```bash
# Start all services
docker-compose up -d

# Build specific service
docker-compose build control-actuator-go

# View logs
docker-compose logs -f otelcol-main
```

## Benefits of Cleanup

1. **Clarity**: Single source of truth for Docker configuration
2. **Consistency**: All references now point to correct service implementations
3. **Maintainability**: No conflicting or outdated configurations
4. **Developer Experience**: Clear path for running the system

## Migration Notes

If you were using the old infrastructure files:
- Use `docker-compose.yaml` at the root
- Service names have changed:
  - `collector` → `otelcol-main`
  - `observer` → `otelcol-observer`
  - `actuator` → `control-actuator-go`
- All services now use the Go implementations in `apps/`