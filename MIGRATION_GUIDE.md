# Phoenix-vNext Streamlining Migration Guide

## Overview

Phoenix-vNext has been streamlined to remove redundancies and consolidate configurations. This guide helps you migrate from the previous structure.

## Key Changes

### 1. Configuration Consolidation

**Prometheus Configuration**
- Old: Multiple configs in `monitoring/prometheus/`, `config/defaults/`, etc.
- New: Single config at `configs/monitoring/prometheus/prometheus.yaml`

**Recording Rules**
- Old: Multiple rule files (`phoenix_rules.yml`, `phoenix_core_rules.yml`, etc.)
- New: Single consolidated file `configs/monitoring/prometheus/rules/phoenix_rules.yml`
- Includes all operational alerts, performance metrics, and advanced analytics

**Grafana Dashboards**
- Old: Scattered across `monitoring/grafana/` and `config/defaults/`
- New: All dashboards in `configs/monitoring/grafana/dashboards/`

### 2. Service Consolidation

**Benchmark Service**
- Location: `services/benchmark/`
- Removed duplicate `apps/benchmark-controller/`

**Control Actuator**
- Primary: `apps/control-actuator-go/` (Go implementation)
- Removed: Bash implementation variants

### 3. Kubernetes Structure

- Kept: `k8s/` with Kustomize structure
- Archived: `infrastructure/kubernetes/` and alternate structures

### 4. Docker Compose

- Main file: `docker-compose.yaml`
- Dev overrides: `docker-compose.override.yml`
- Removed: `docker-compose.dev.yml`, modular variants

## Migration Steps

### Step 1: Backup Current State
```bash
# Create backup
tar -czf phoenix-backup-$(date +%Y%m%d).tar.gz \
  configs/ data/ .env docker-compose.yaml
```

### Step 2: Update Environment Variables
No changes required - existing `.env` file remains compatible.

### Step 3: Update Docker Compose Commands

**Old:**
```bash
docker-compose -f docker-compose.yaml -f docker-compose.dev.yml up
```

**New:**
```bash
# Production
docker-compose up -d

# Development (auto-loads override file)
docker-compose up -d

# With generators
docker-compose --profile generators up -d
```

### Step 4: Update Configuration References

If you have custom scripts referencing old paths, update them:

```bash
# Old paths
/monitoring/prometheus/prometheus.yaml
/config/defaults/monitoring/grafana/dashboards/

# New paths
/configs/monitoring/prometheus/prometheus.yaml
/configs/monitoring/grafana/dashboards/
```

### Step 5: Restart Services
```bash
# Stop all services
docker-compose down

# Start with new structure
docker-compose up -d

# Verify health
docker-compose ps
curl http://localhost:9090/-/healthy  # Prometheus
curl http://localhost:3000/api/health # Grafana
```

## Validation Checklist

- [ ] All services running: `docker-compose ps`
- [ ] Prometheus scraping metrics: http://localhost:9090/targets
- [ ] Grafana dashboards loading: http://localhost:3000
- [ ] Control loop updating: `docker-compose logs control-actuator-go`
- [ ] No missing config errors in logs

## Rollback Plan

If issues occur:
```bash
# Stop services
docker-compose down

# Restore backup
tar -xzf phoenix-backup-[date].tar.gz

# Use archived configs if needed
cp -r archive/monitoring/prometheus/* configs/monitoring/prometheus/

# Restart
docker-compose up -d
```

## Benefits of Streamlining

1. **Simpler Maintenance**: Single source of truth for each component
2. **Faster Startup**: No redundant config loading
3. **Clearer Dependencies**: Explicit service relationships
4. **Easier Debugging**: Consistent file locations
5. **Better Performance**: Consolidated recording rules

## Support

For issues or questions:
- Check logs: `docker-compose logs [service-name]`
- Review runbooks: `runbooks/troubleshooting/common-issues.md`
- File issues: GitHub Issues with `streamlining` tag