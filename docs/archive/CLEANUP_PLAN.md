# Phoenix Cleanup Execution Plan

## Analysis Results

### ✅ Services USED in docker-compose.yaml
- `apps/control-actuator-go/` ✅ KEEP
- `apps/anomaly-detector/` ✅ KEEP  
- `services/benchmark/` ✅ KEEP
- `services/generators/synthetic/` ✅ KEEP

### ❌ Services NOT USED in docker-compose.yaml
- `services/analytics/` ❌ REMOVE - Not referenced
- `services/collector/` ❌ REMOVE - Not used (using otelcol-contrib image)
- `services/control-plane/` ❌ REMOVE - Superseded by apps/control-actuator-go
- `services/generators/complex/` ❌ REMOVE - Using synthetic generator
- `services/validator/` ❌ REMOVE - Not referenced

## Cleanup Execution Order

### Phase 1: Archive Directory (SAFE)
```bash
rm -rf archive/
```

### Phase 2: Redundant Documentation Structure (SAFE)
```bash
rm -rf docs/configs/
rm -rf docs/docs/
rm -rf docs/scripts/
```

### Phase 3: Unused Services (SAFE)
```bash
rm -rf services/analytics/
rm -rf services/collector/
rm -rf services/control-plane/
rm -rf services/generators/complex/
rm -rf services/validator/
```

### Phase 4: Duplicate Scripts (With Verification)
```bash
# Verify symbolic links exist first
ls -la tools/scripts/initialize-environment.sh  # Should point to consolidated
ls -la scripts/deploy.sh  # Should point to consolidated
ls -la tests/integration/test_core_functionality.sh  # Should point to consolidated

# Then remove originals
rm scripts/api-test.sh scripts/cleanup.sh scripts/deploy.sh
rm scripts/functional-test.sh scripts/newrelic-integration.sh scripts/validate-system.sh
rm -rf tools/scripts/
rm tests/integration/test_core_functionality.sh
```

### Phase 5: Template/Config Analysis (CAREFUL)
Review and clean configs/templates/ and configs/production/

### Phase 6: Infrastructure Duplicates (REVIEW)
Review infrastructure/docker/compose/ vs root compose files