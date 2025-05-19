# Configuration Reference

This document provides a comprehensive reference for configuring the Phoenix system.

## Configuration Files

Phoenix uses two main configuration files:

1. **config.yaml**: Standard OpenTelemetry Collector configuration
2. **policy.yaml**: Self-adaptive behavior configuration

These files are located in the `configs/[environment]/` directory, with environment-specific versions:
- `configs/default/`: Standard baseline configuration
- `configs/development/`: Development configuration with verbose logging
- `configs/production/`: Production configuration optimized for stability
- `configs/testing/`: Configuration optimized for tests

## config.yaml Reference

The `config.yaml` file follows the standard OpenTelemetry Collector configuration format:

```yaml
extensions:
  pic_control:
    policy_file_path: /etc/sa-omf/policy.yaml
    max_patches_per_minute: 3
    patch_cooldown_seconds: 10
    safe_mode_processor_configs:
      adaptive_topk:
        k_value: 10

receivers:
  hostmetrics:
    collection_interval: 10s
    scrapers:
      cpu:
      memory:
      load:
      filesystem:
      network:
      paging:
      process:
        include:
          match_type: regexp
          processes: [".*"]

processors:
  priority_tagger:
    enabled: true
    rules:
      - match: "process.command_line=~/.*java.*/"
        priority: high
      - match: "process.name=~/nginx|httpd/"
        priority: high
      - match: "process.name=~/mysql|postgres|mongodb/"
        priority: critical
      - match: "process.command_line=~/.*python.*/"
        priority: medium
      - match: ".*"
        priority: low
  
  pid_decider:
    controllers:
      - name: coverage_controller
        enabled: true
        kpi_metric_name: aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m
        kpi_target_value: 0.90
        kp: 30
        ki: 5
        kd: 0
        hysteresis_percent: 3
        output_config_patches:
          - target_processor_name: adaptive_topk
            parameter_path: k_value
            change_scale_factor: -20.0
            min_value: 10
            max_value: 60

exporters:
  logging:
    verbosity: detailed
  pic_connector: {}

service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger]
      exporters: [logging]
    
    control:
      receivers: [hostmetrics]
      processors: [pid_decider]
      exporters: [pic_connector, logging]
  
  extensions: [pic_control]
```

## policy.yaml Reference

The `policy.yaml` file defines the self-adaptive behavior of the system:

```yaml
global_settings:
  autonomy_level: shadow  # Start in shadow mode for safety
  collector_cpu_safety_limit_mcores: 400
  collector_rss_safety_limit_mib: 350

processors_config:
  priority_tagger:
    enabled: true
    rules:
      - match: "process.command_line=~/.*java.*/"
        priority: high
      - match: "process.name=~/nginx|httpd/"
        priority: high
      - match: "process.name=~/mysql|postgres|mongodb/"
        priority: critical
      - match: "process.command_line=~/.*python.*/"
        priority: medium
      - match: ".*"
        priority: low
  
  adaptive_topk:
    enabled: true
    k_value: 30
    k_min: 10
    k_max: 60
  
  cardinality_guardian:
    enabled: false
    max_unique: 1000
  
  reservoir_sampler:
    enabled: false
    reservoir_size: 100
  
  others_rollup:
    enabled: false

pid_decider_config:
  controllers:
    - name: coverage_controller
      enabled: true
      kpi_metric_name: aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m
      kpi_target_value: 0.90
      kp: 30
      ki: 5
      kd: 0
      hysteresis_percent: 3
      output_config_patches:
        - target_processor_name: adaptive_topk
          parameter_path: k_value
          change_scale_factor: -20.0
          min_value: 10
          max_value: 60

pic_control_config:
  policy_file_path: /etc/sa-omf/policy.yaml
  max_patches_per_minute: 3
  patch_cooldown_seconds: 10
  safe_mode_processor_configs:
    adaptive_topk:
      k_value: 10
    cardinality_guardian:
      max_unique: 100

service:
  extensions: [pic_control, health_check]
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger, adaptive_topk]
      exporters: [prometheusremotewrite]
    control:
      receivers: [prometheus/self]
      processors: [pid_decider]
      exporters: [pic_connector]
```

## Configuration Options

### Global Settings

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `autonomy_level` | string | Level of autonomy (shadow, advisory, active) | `"shadow"` |
| `collector_cpu_safety_limit_mcores` | int | CPU usage limit in millicores | 400 |
| `collector_rss_safety_limit_mib` | int | Memory usage limit in MiB | 350 |

### Processor Configuration

Each processor has its own configuration section:

#### priority_tagger

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `enabled` | boolean | Enable the processor | `true` |
| `rules` | array | Priority tagging rules | - |

#### adaptive_topk

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `enabled` | boolean | Enable the processor | `true` |
| `k_value` | int | Initial k value | 30 |
| `k_min` | int | Minimum allowed k value | 10 |
| `k_max` | int | Maximum allowed k value | 60 |

### PID Controller Configuration

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `name` | string | Controller name | - |
| `enabled` | boolean | Enable the controller | `true` |
| `kpi_metric_name` | string | Metric name to monitor | - |
| `kpi_target_value` | float | Target value for KPI | - |
| `kp` | float | Proportional term | - |
| `ki` | float | Integral term | - |
| `kd` | float | Derivative term | - |
| `hysteresis_percent` | float | Hysteresis threshold | 3 |

### Extension Configuration

#### pic_control

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `policy_file_path` | string | Path to policy file | - |
| `max_patches_per_minute` | int | Rate limit for patches | 3 |
| `patch_cooldown_seconds` | int | Cooldown between patches | 10 |
| `safe_mode_processor_configs` | object | Safe mode configurations | - |

## Environment Variables

The following environment variables can be used to override configuration:

| Environment Variable | Description |
|----------------------|-------------|
| `SA_OMF_CONFIG_PATH` | Path to config file |
| `SA_OMF_POLICY_PATH` | Path to policy file |
| `SA_OMF_LOG_LEVEL` | Log level (debug, info, warn, error) |
| `SA_OMF_AUTONOMY_LEVEL` | Override autonomy level |

## Configuration Loading

Configuration files are loaded from these locations in order:

1. Path specified via command line flags (`--config=...`)
2. Environment variable location (`SA_OMF_CONFIG_PATH`)
3. Default locations:
   - `/etc/sa-omf/config.yaml`
   - `./configs/default/config.yaml`