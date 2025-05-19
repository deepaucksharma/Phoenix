# others_rollup Processor

The `others_rollup` processor aggregates metrics from low priority processes into
a single synthetic resource named `others`. This reduces metric cardinality when
many low impact processes are present.

## Configuration

```yaml
processors:
  others_rollup:
    enabled: true
    strategy: sum # or "avg"
```

- `enabled`  – toggles whether aggregation is applied.
- `strategy` – aggregation strategy used for numeric values. Supported values are
  `sum` and `avg`.

## Dynamic Configuration

The processor implements the `UpdateableProcessor` interface. The following
configuration parameters can be patched at runtime:

- `enabled`
- `strategy`

Example patch:
```json
{
  "patch_id": "switch-to-avg",
  "target_processor_name": "others_rollup",
  "parameter_path": "strategy",
  "new_value": "avg"
}
```
