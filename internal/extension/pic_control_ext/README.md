# pic_control Extension

`pic_control` provides centralized governance for configuration changes. It
validates incoming `ConfigPatch` objects, enforces policy rules and safety limits
and applies approved patches to target processors.

## Setup

Add the extension to the `extensions` section and reference it from the control
pipeline:

```yaml
extensions:
  pic_control:
    policy_file_path: /etc/sa-omf/policy.yaml
    max_patches_per_minute: 3
    patch_cooldown_seconds: 10
    safe_mode_processor_configs:
      adaptive_topk:
        k_value: 10
```

```yaml
service:
  extensions: [pic_control]
  pipelines:
    control:
      receivers: [prometheus/self]
      processors: [pid_decider]
      exporters: [pic_connector]
```

## policy.yaml Format

The policy file controls self-adaptive behavior and contains these top-level
sections:

- `global_settings`: autonomy level and resource safety limits
- `processors_config`: default configuration for each processor
- `pid_decider_config`: PID controller definitions
- `pic_control_config`: extension options (mirrors the settings above)
- `service`: OpenTelemetry Collector service configuration

The schema is defined in `pkg/policy/schema.go` and validated when the extension
starts.
