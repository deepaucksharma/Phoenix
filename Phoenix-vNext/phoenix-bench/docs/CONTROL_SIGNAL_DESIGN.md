# Phoenix-vNext Control Signal Design

## Overview

This document describes the design and implementation of the control signal system in Phoenix-vNext. Control signals are used to manage optimization modes in the collector pipeline, allowing dynamic adjustment of collection behavior based on observed metrics cardinality.

## Key Components

1. **otelcol-observer**: Monitors metrics from the main collector and generates control signals based on cardinality thresholds.
2. **otelcol-main**: Consumes control signals and adjusts its pipeline configuration accordingly.
3. **opt_mode.yaml**: The control file that contains the current mode and related configuration.
4. **check-coherence.sh**: Verifies synchronization between observed and applied modes.
5. **update-control-file.sh**: Manual override of the control file.
6. **override-control-mode.sh**: Simplified interface for manual mode switching.

## Mode Definitions

The system supports three optimization modes that are defined in `otelcol-main.yaml`:

- **moderate**: Default operation mode with full data collection (active when cardinality < threshold_moderate)
- **adaptive**: Intermediate mode with selective data reduction (active when threshold_moderate < cardinality <= threshold_adaptive)
- **ultra**: Maximum optimization with aggressive data reduction (active when cardinality > threshold_adaptive)

## Control File Schema

The `opt_mode.yaml` control file uses the following schema fields that are validated by the main collector:

```yaml
# Required fields (validated by otelcol-main)
mode: "moderate" | "adaptive" | "ultra"  # Current optimization mode
last_updated: "2025-05-22T10:00:00Z"     # ISO-8601 timestamp
config_version: 12345                    # Integer version number
correlation_id: "observer-1621234567"    # Unique identifier for the control change

# Optional fields (used for context/debugging)
reason: "ts_opt=400, thresholds: moderate=300.0, caution=350.0, warning=400.0, ultra=450.0"
optimization_level: 75                   # 0-100 scale of optimization intensity
ts_count: 400                            # Current timeseries count that triggered this mode

# Threshold configuration
thresholds:
  moderate: 300.0
  caution: 350.0  # Maps to "adaptive" mode when exceeded
  warning: 400.0  # Also maps to "adaptive" mode when exceeded
  ultra: 450.0    # Maps to "ultra" mode when exceeded

# State transition information
state:
  previous_mode: "moderate"
  transition_timestamp: "2025-05-22T09:55:00Z"
  transition_duration_seconds: 0
  stability_period_seconds: 300  # Don't change mode again for 5 minutes
```

## Control Flow

1. The observer scrapes cardinality metrics from the main collector via its Prometheus endpoint.
2. The `transform/control_file_generator` processor analyzes the metrics and determines the appropriate mode.
3. The observer writes the control file via the file exporter.
4. The main collector detects the file change and updates its pipeline configuration.
5. The `check-coherence.sh` script verifies that the observer's determined mode matches what the main collector is applying.

## Monitoring

The observer exposes a `phoenix_observer_mode` metric with a label `mode` that contains the current mode value. This metric can be queried to verify the current mode setting:

```
curl -s http://localhost:8891/metrics | grep phoenix_observer_mode
```

## Service Dependencies

- The synthetic-metrics-collector service provides additional metrics for testing and development.
- The observer depends on the main collector for scraping metrics.
- The main collector depends on the observer for control signals.

## Environment Variables

- `THRESHOLD_MODERATE`: Default 300.0, controls the threshold for moderate mode
- `THRESHOLD_ADAPTIVE`: Default 375.0, controls the threshold for adaptive mode
- `THRESHOLD_ULTRA`: Default 450.0, controls the threshold for ultra mode
- `OBSERVER_INSTANCE_ID`: Instance identifier for the observer, used in correlation IDs
- `CONTROL_SIGNAL_WRITE_PATH`: Path where the observer writes the control file
- `CONTROL_SIGNAL_PATH`: Path where the main collector reads the control file
