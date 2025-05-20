# priority_tagger Processor

The `priority_tagger` processor adds priority attributes to metrics resources based on configured rules matching the `process.name` attribute.

## Configuration

```yaml
processors:
  priority_tagger:
    enabled: true
    rules:
      - match: "nginx.*"
        priority: "high"
      - match: ".*mysql.*"
        priority: "critical"
      - match: "background.*"
        priority: "low"
```

## Description

The `priority_tagger` processor examines each resource's `process.name` attribute and applies rules to tag it with an appropriate priority level. This priority can later be used by other processors, exporters, or for querying and filtering.

The processor implements the `UpdateableProcessor` interface, which allows dynamic reconfiguration at runtime through configuration patches.

## Rules

Each rule consists of:

- `match`: A regex pattern to match against process.name
- `priority`: Priority value to assign (typically "critical", "high", "medium", "low")

Rules are evaluated in order, and processing stops at the first match.

## Dynamic Configuration

The processor supports the following configuration patches:

- `enabled`: Boolean to enable/disable the processor
- `rules`: Array of rules to replace the current ruleset

Example patch:
```json
{
  "patch_id": "increase-nginx-priority",
  "target_processor_name": "priority_tagger",
  "parameter_path": "rules",
  "new_value": [
    {
      "match": "nginx.*",
      "priority": "critical"
    },
    {
      "match": ".*mysql.*",
      "priority": "critical"
    },
    {
      "match": "background.*",
      "priority": "low"
    }
  ]
}
```
