# Phoenix-vNext Pipeline Analysis

## Overview
The Phoenix-vNext system operates three distinct metric processing pipelines with varying levels of cardinality optimization, plus a cardinality observatory pipeline for monitoring. All pipelines share a common intake stage before diverging into specialized processing chains.

## Common Intake Pipeline (lines 310-319)
All metrics flow through this initial processing stage:

### Processors Applied (in order):
1. **memory_limiter** - Prevents OOM by limiting memory usage to ${OTELCOL_MAIN_MEMORY_LIMIT_MIB_QUARTER} (default 512 MiB)
2. **resourcedetection/common** - Detects environment and system metadata (hostname, OS)
3. **transform/common_enrichment_and_profile_tagger** - Adds:
   - `benchmark.id` from environment
   - `deployment.environment` from environment  
   - `phoenix.optimisation_profile` from control file (conservative/balanced/aggressive)
   - `phoenix.control.correlation_id` from control file
4. **cumulativetodelta** - Converts cumulative metrics (process.cpu.time) to delta values
5. **transform/priority_classification_engine** - Classifies processes by priority:
   - **critical**: Process names containing "critical"
   - **high**: java_app, python_api, node_gateway
   - **medium**: nginx, postgres, data_pipeline
   - **low**: Everything else
6. **transform/global_initial_attribute_stripping** - Removes `process.parent_pid` attribute

## Pipeline 1: Full Fidelity (lines 321-329)

### Purpose
Maintains complete metric fidelity with minimal processing - serves as baseline for comparison.

### Processors Applied:
1. **memory_limiter** - Secondary memory protection
2. **resource/tag_pipeline_full** - Adds `phoenix.pipeline.strategy=full_fidelity`
3. **transform/full_attribute_management** - Removes only `process.pid` to reduce cardinality slightly
4. **transform/cardinality_counter_full** - Creates `phoenix_full_output_ts_active` metric for cardinality tracking
5. **batch/final_export_batcher** - Batches metrics (8192 metrics/batch, 10s timeout)

### Key Characteristics:
- Retains all processes and metrics
- Minimal attribute removal (only PID)
- Highest cardinality output
- Complete process visibility

## Pipeline 2: Optimised (lines 331-342)

### Purpose
Moderate cardinality reduction through selective filtering and rollup of lower-priority processes.

### Processors Applied:
1. **memory_limiter** - Memory protection
2. **resource/tag_pipeline_optimised** - Adds `phoenix.pipeline.strategy=optimised`
3. **filter/optimised_selection** - Keeps only:
   - All critical priority processes
   - High priority processes with valid data points
   - Medium priority postgres and nginx processes
4. **transform/optimised_rollup_prep** - Rolls up other medium priority processes:
   - Renames to `{priority}_others_optimised`
   - Adds `rollup.process.count=1` marker
5. **groupbyattrs/optimised_rollup** - Groups metrics by:
   - host.name, service.name, process.executable.name
   - phoenix.pipeline.strategy, phoenix.optimisation_profile, phoenix.priority
6. **transform/optimised_attribute_cleanup** - Removes attributes based on priority:
   - Removes `process.command_line` for non-critical
   - Removes `process.owner` for low priority
   - Removes `container.id` for all
7. **transform/cardinality_counter_optimised** - Creates cardinality tracking metric
8. **batch/final_export_batcher** - Batches for export

### Key Characteristics:
- Selective process filtering by priority
- Rollup of medium-priority non-essential processes
- Attribute reduction based on priority tiers
- ~40-60% cardinality reduction expected

## Pipeline 3: Experimental TopK (lines 344-355)

### Purpose
Aggressive cardinality reduction focusing only on most critical processes.

### Processors Applied:
1. **memory_limiter** - Memory protection
2. **resource/tag_pipeline_experimental** - Adds `phoenix.pipeline.strategy=experimental_topk`
3. **filter/experimental_aggressive_selection** - Keeps only:
   - Critical priority processes
   - High priority java_critical and postgres_primary processes
4. **transform/experimental_rollup_marker** - Aggressive rollup:
   - All non-critical renamed to `phoenix.others.experimental_aggressive`
   - Adds `rollup.process.count=1` marker
5. **groupbyattrs/experimental_rollup** - Groups by minimal dimensions:
   - host.name, service.name, process.executable.name
   - phoenix.pipeline.strategy, phoenix.optimisation_profile
6. **transform/experimental_attribute_cleanup** - Keeps only essential attributes:
   - host.name, service.name, process.executable.name
   - phoenix.pipeline.strategy, phoenix.optimisation_profile
   - phoenix.priority, benchmark.id
7. **transform/cardinality_counter_experimental** - Creates cardinality tracking metric
8. **batch/final_export_batcher** - Batches for export

### Key Characteristics:
- Most aggressive filtering (critical + select high priority only)
- Maximum attribute reduction
- All other processes rolled into single metric
- ~70-90% cardinality reduction expected

## Cardinality Observatory Pipeline (lines 357-365)

### Purpose
Monitors cardinality explosion risks and generates alerts for auto-remediation.

### Processors Applied:
1. **memory_limiter** - Memory protection
2. **transform/cardinality_analysis** - Analyzes cardinality risks:
   - Sets `cardinality.growth.rate` based on priority (low/moderate/high)
   - Calculates `cardinality.explosion.risk` (0.2-0.9 scale)
   - Higher risk for critical Java processes (0.9)
3. **filter/cardinality_explosion_alert** - Filters metrics with explosion risk > 0.8
4. **transform/cardinality_alert_enrichment** - Enriches alerts:
   - Sets `alert.type=cardinality_explosion`
   - Sets severity (critical >0.9, warning >0.8)
   - Enables `auto_remediate` for critical alerts
5. **batch/final_export_batcher** - Batches for export

## Key Metrics for Dashboard Functionality

### Pipeline Performance Metrics:
- `phoenix_full_output_ts_active` - Active time series count for full pipeline
- `phoenix_optimised_output_ts_active` - Active time series count for optimised pipeline  
- `phoenix_experimental_output_ts_active` - Active time series count for experimental pipeline

### Process Metrics (per pipeline):
- `process.cpu.time` - CPU usage (converted to delta)
- `process.memory.usage` - Memory consumption
- `process.disk.io` - Disk I/O operations
- `process.threads` - Thread count
- `process.open_file_descriptors` - File descriptor usage

### Control System Metrics:
- `phoenix.optimisation_profile` - Current optimization mode (conservative/balanced/aggressive)
- `phoenix.control.correlation_id` - Control loop correlation ID
- `phoenix.priority` - Process priority classification

### Cardinality Observatory Metrics:
- `cardinality.explosion.risk` - Risk score (0.2-0.9)
- `cardinality.growth.rate` - Growth rate classification
- `alert.type` - Alert classification
- `alert.severity` - Alert severity level
- `auto_remediate` - Auto-remediation flag

### Resource Attributes Available:
- `host.name` - Hostname
- `service.name` - Service identifier
- `process.executable.name` - Process name (or rollup name)
- `benchmark.id` - Benchmark run identifier
- `deployment.environment` - Deployment environment
- `phoenix.pipeline.strategy` - Pipeline type
- `rollup.process.count` - Number of processes rolled up (for aggregated metrics)

## Pipeline Differences Summary

| Aspect | Full Fidelity | Optimised | Experimental TopK |
|--------|--------------|-----------|-------------------|
| Process Selection | All processes | Critical + High + Select Medium | Critical + Select High only |
| Attribute Retention | All except PID | Priority-based removal | Minimal essential only |
| Rollup Strategy | None | Medium priority others | All non-critical |
| Cardinality Reduction | ~5-10% | ~40-60% | ~70-90% |
| Use Case | Baseline/debugging | Production balanced | High-scale environments |