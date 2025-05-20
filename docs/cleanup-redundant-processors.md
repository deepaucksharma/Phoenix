# Cleanup Guide: Removing Redundant Processors

This document provides guidance for cleaning up redundant processors and related components after the consolidation of functionality into the `metric_pipeline` processor.

## Background

As part of our codebase optimization, we have consolidated the functionality of multiple separate processors into a unified `metric_pipeline` processor. This consolidation effort:

1. Combines the features of `priority_tagger`, `adaptive_topk`, and `others_rollup` processors
2. Integrates the `resource_filter` functionality directly into the `metric_pipeline` processor
3. Replaces specific metrics components with the `UnifiedMetricsCollector`

## Files to be Removed

### Processors

The following processor directories and their contents should be removed:

1. `/internal/processor/adaptive_topk/`
   - `config.go`
   - `factory.go`
   - `processor.go`

2. `/internal/processor/others_rollup/`
   - `config.go`
   - `factory.go`
   - `processor.go`

3. `/internal/processor/priority_tagger/`
   - `config.go`
   - `factory.go`
   - `processor.go`
   
4. `/internal/processor/resource_filter/`
   - `config.go`
   - `factory.go`
   - `processor.go`

### Tests

The following test directories should be removed:

1. `/test/processors/adaptive_topk/`
2. `/test/processors/others_rollup/`
3. `/test/processors/priority_tagger/`

### Documentation

The following documentation file should be removed:

1. `/docs/components/processors/others_rollup.md`

### Metrics

The following metrics file should be removed:

1. `/pkg/metrics/pid_metrics.go`

## Automated Cleanup

We have created a set of scripts in `/scripts/cleanup/` to automate the cleanup process:

### Master Cleanup Script

The simplest approach is to use the master cleanup script which orchestrates the entire process:

```bash
cd /home/deepak/Phoenix
./scripts/cleanup/master-cleanup.sh
```

This script performs the following steps:
1. Removes redundant processor files and directories
2. Updates import references in remaining files
3. Cleans up Go module dependencies
4. Runs tests to verify functionality

### Individual Cleanup Scripts

If you prefer more control, you can run the individual scripts separately:

1. **Remove redundant files**:
   ```bash
   ./scripts/cleanup/cleanup-redundant-processors-v2.sh
   ```

2. **Update import references**:
   ```bash
   ./scripts/cleanup/cleanup-processor-imports.sh
   ```

## Manual Steps After Cleanup

After running the automated cleanup scripts, some manual steps may be necessary:

1. **Fix remaining import references**: The import script will flag any files that still contain references to removed processors that couldn't be automatically updated.

2. **Update policy schema**: The schema in `pkg/policy/schema.go` will need to be updated to include the new `metric_pipeline` processor configuration schema.

3. **Update documentation references**: Any documentation that references the removed processors should be updated to reference the `metric_pipeline` processor.

4. **Update configuration examples**: Any examples of `config.yaml` or `policy.yaml` that use the removed processors should be updated to use the new `metric_pipeline` processor.

## Handling Special Cases

### Registry Processor Factory

In the OpenTelemetry Collector model, processors are registered with a central registry. Look for and update:

1. Processor registration in `cmd/sa-omf-otelcol/main.go`:
   ```go
   // OLD
   priority_tagger.NewFactory(),
   adaptive_topk.NewFactory(),
   others_rollup.NewFactory(),
   
   // NEW
   metric_pipeline.NewFactory(),
   ```

### Configuration File Updates

After removing the old processors, update any configuration files that reference them:

```yaml
# OLD configuration - to be updated or removed
processors:
  priority_tagger:
    # ...
  adaptive_topk:
    # ...
  others_rollup:
    # ...

# NEW configuration - to be used
processors:
  metric_pipeline:
    resource_filter:
      # ...
    transformation:
      # ...
```

## Verifying the Cleanup

After running the cleanup process, verify that:

1. All redundant processor code has been removed
2. Import statements have been updated
3. Tests pass successfully
4. The application builds and runs correctly

## Preserving Functionality

The `metric_pipeline` processor preserves all functionality from the removed processors:

1. Priority tagging (from `priority_tagger`)
2. Top-K filtering (from `adaptive_topk`)
3. Rollup aggregation (from `others_rollup`)
4. Resource filtering strategies (from `resource_filter`)

The self-metrics implementation for the `metric_pipeline` processor provides even more detailed insights than the individual processors did.

## Conclusion

This cleanup effort results in a more maintainable codebase by removing redundant components that have been consolidated into the unified `metric_pipeline` processor. The provided scripts make this process as automated as possible, minimizing the risk of errors and ensuring a smooth transition.