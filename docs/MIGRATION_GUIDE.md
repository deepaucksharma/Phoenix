# Phoenix vNext Migration Guide

## Overview
This guide helps migrate from the monolithic Phoenix-vNext structure to the new modular architecture.

## Migration Steps

### 1. Backup Current State
```bash
# Create backup of current configuration
tar -czf phoenix-backup-$(date +%Y%m%d).tar.gz configs/ apps/ docker-compose.yaml
```

### 2. Initialize New Structure
```bash
# Ensure environment is ready
make dev-setup

# Build all modules
make build
```

### 3. Update Environment Variables
The `.env` file remains compatible, but some new variables are available:
- `LOG_LEVEL` - Global log level for all modules
- `ENVIRONMENT` - Deployment environment (development/staging/production)

### 4. Deploy Services

#### Option A: Full Migration
```bash
# Stop old services
docker-compose down

# Deploy new modular services
make deploy
```

#### Option B: Gradual Migration
```bash
# Deploy monitoring first
make deploy-monitoring

# Deploy control plane
make deploy-control

# Deploy core
make deploy-core

# Finally, deploy generators
make deploy-generators
```

### 5. Verify Migration
```bash
# Check service health
make health-check

# View logs
make logs

# Access dashboards
make perf-monitor
```

## Configuration Changes

### Old Structure
```
configs/
├── otel/collectors/main.yaml
├── control/optimization_mode.yaml
└── monitoring/
```

### New Structure
```
modules/
├── phoenix-core/configs/
├── phoenix-control/
│   ├── observer/config/
│   └── actuator/config/
└── phoenix-monitoring/
```

## API Changes

### Control API
- Old: Direct file manipulation
- New: REST API at `http://localhost:8080/api/v1/control`

### Metrics Access
- Old: Single endpoint per pipeline
- New: Structured API with pipeline selection

## Rollback Procedure

If issues occur:
```bash
# Stop new services
make down

# Restore old configuration
tar -xzf phoenix-backup-YYYYMMDD.tar.gz

# Start old services
docker-compose up -d
```

## Common Issues

### Issue: Services not communicating
- Check network configuration in docker-compose.modular.yaml
- Ensure all services are on the `phoenix-net` network

### Issue: Control signals not updating
- Verify volume mount for control-signals
- Check actuator logs: `make control-logs`

### Issue: Metrics not appearing
- Verify prometheus scrape configs
- Check collector health: `curl http://localhost:13133`

## Support
For migration support, please open an issue at:
https://github.com/deepaucksharma/Phoenix/issues