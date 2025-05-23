# Phoenix Metric Standards and Best Practices

## Overview
This document defines the metric naming conventions, labeling standards, and cardinality management practices for the Phoenix-vNext platform.

## Metric Naming Conventions

### Format
```
<namespace>_<subsystem>_<name>_<unit>
```

### Examples
- `phoenix_pipeline_processed_total` (counter)
- `phoenix_pipeline_latency_seconds` (histogram)
- `phoenix_cardinality_estimate_count` (gauge)
- `phoenix_optimization_mode_info` (info metric)

### Rules
1. Use lowercase with underscores
2. Be descriptive but concise
3. Include unit as suffix when applicable
4. Use standard units (seconds, bytes, ratio)

### Metric Types

#### Counters
- Suffix with `_total`
- Always increasing
- Reset only on restart
```
phoenix_pipeline_processed_total
phoenix_errors_total
phoenix_requests_total
```

#### Gauges
- Current value metrics
- Can go up and down
```
phoenix_queue_length
phoenix_active_connections
phoenix_memory_usage_bytes
```

#### Histograms
- Include unit in name
- Use for latency, sizes
```
phoenix_request_duration_seconds
phoenix_batch_size_items
phoenix_response_size_bytes
```

#### Info Metrics
- Configuration/metadata
- Value always 1
- Information in labels
```
phoenix_build_info{version="1.2.3",commit="abc123"}
phoenix_optimization_mode_info{mode="balanced"}
```

## Label Standards

### Good Labels
✅ **Bounded cardinality**
- `service`: Fixed set of service names
- `pipeline`: `full_fidelity`, `optimized`, `experimental_topk`
- `status`: `success`, `error`, `timeout`
- `region`: `us-east`, `us-west`, `eu-central`
- `environment`: `dev`, `staging`, `prod`
- `error_type`: Enumerated error categories

### Bad Labels
❌ **Unbounded cardinality**
- `user_id`: Unique per user
- `request_id`: Unique per request
- `ip_address`: Too many values
- `timestamp`: Continuous values
- `session_id`: Unique identifiers
- `full_path`: URL paths with IDs

### Label Naming
1. Use lowercase with underscores
2. Be consistent across metrics
3. Keep labels short but descriptive
4. Avoid redundancy with metric name

## Cardinality Management

### Targets
| Pipeline | Target Cardinality | Hard Limit |
|----------|-------------------|------------|
| Full Fidelity | < 30,000 | 50,000 |
| Optimized | < 20,000 | 30,000 |
| Experimental | < 10,000 | 15,000 |

### Calculation
```
Cardinality = Metric Names × Label Combinations
```

Example:
- Metric: `http_requests_total`
- Labels: `method` (5 values) × `status` (10 values) × `service` (20 values)
- Cardinality: 1 × 5 × 10 × 20 = 1,000 time series

### Cardinality Budgets

#### Per Service Budget
```yaml
service_budgets:
  api_gateway: 5000
  user_service: 3000
  payment_service: 4000
  notification_service: 2000
  background_jobs: 3000
```

#### Monitoring Cardinality
```promql
# Current cardinality by service
count by (service) (
  group by (__name__, service) ({service!=""})
)

# Top 10 metrics by cardinality
topk(10, 
  count by (__name__) (
    group by (__name__, {}) ({})
  )
)
```

## Implementation Guidelines

### 1. Pre-deployment Validation
```bash
# Test metric cardinality in staging
./scripts/validate-metrics.sh staging

# Estimate production cardinality
./scripts/estimate-cardinality.sh --env=prod --scale-factor=10
```

### 2. Metric Registration
```yaml
# metrics_registry.yaml
metrics:
  - name: phoenix_api_requests_total
    type: counter
    labels:
      - method: ["GET", "POST", "PUT", "DELETE"]
      - status: ["2xx", "3xx", "4xx", "5xx"]
      - service: bounded_list
    max_cardinality: 500
```

### 3. Instrumentation Code

#### Good Example
```go
requestCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "phoenix_api_requests_total",
        Help: "Total API requests",
    },
    []string{"method", "status_class", "service"},
)

// Usage
requestCounter.WithLabelValues(
    r.Method,
    fmt.Sprintf("%dxx", statusCode/100),
    serviceName,
).Inc()
```

#### Bad Example
```go
// DON'T DO THIS
requestCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "api_requests",
    },
    []string{"user_id", "path", "ip", "timestamp"}, // Unbounded labels!
)
```

## Optimization Strategies

### 1. Label Aggregation
```yaml
processors:
  metricstransform:
    transforms:
      - include: http_requests_total
        match_type: strict
        action: update
        operations:
          - action: aggregate_labels
            label_set: [service, method, status_class]
            aggregation_type: sum
```

### 2. Sampling High-Cardinality Metrics
```yaml
processors:
  probabilistic_sampler:
    sampling_percentage: 10
    attribute_source: traceID
```

### 3. Drop Unnecessary Metrics
```yaml
processors:
  filter:
    metrics:
      exclude:
        match_type: regexp
        metric_names:
          - ".*_debug.*"
          - ".*_test.*"
          - "go_.*"  # Runtime metrics in prod
```

## Monitoring and Alerting

### Cardinality Alerts
```yaml
groups:
  - name: cardinality
    rules:
      - alert: HighMetricCardinality
        expr: |
          count by (job) (
            count by (__name__, job)({__name__!=""})
          ) > 1000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High cardinality in {{ $labels.job }}"
          
      - alert: CardinalityGrowthRate
        expr: |
          rate(prometheus_tsdb_symbol_table_size_bytes[1h]) > 1000000
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "Rapid cardinality growth detected"
```

## Review Process

### Metric Review Checklist
- [ ] Follows naming convention
- [ ] Labels are bounded
- [ ] Cardinality calculated and within budget
- [ ] Has appropriate help text
- [ ] Documented in registry
- [ ] Tested in staging
- [ ] Dashboards updated
- [ ] Alerts configured

### Quarterly Review
1. Analyze cardinality trends
2. Identify optimization opportunities
3. Update budgets based on growth
4. Archive unused metrics
5. Document lessons learned

## Tools and Scripts

### Cardinality Analysis
```bash
# Analyze current cardinality
promtool tsdb analyze /prometheus

# Find high-cardinality metrics
curl -s localhost:9090/api/v1/label/__name__/values | \
  jq -r '.data[]' | \
  xargs -I {} curl -s "localhost:9090/api/v1/query?query=count(count+by(__name__)({__name__%3D~'{}',__name__%3D~'.%2B'}))" | \
  jq -r '.data.result[] | "\(.value[1]) \(.metric.__name__)"' | \
  sort -rn | head -20
```

### Metric Validation
```go
// validate_metrics.go
func ValidateMetric(name string, labels []string) error {
    if !metricNameRegex.MatchString(name) {
        return fmt.Errorf("invalid metric name: %s", name)
    }
    
    for _, label := range labels {
        if unboundedLabels[label] {
            return fmt.Errorf("unbounded label: %s", label)
        }
    }
    
    estimatedCardinality := CalculateCardinality(name, labels)
    if estimatedCardinality > MaxCardinality {
        return fmt.Errorf("exceeds cardinality limit: %d", estimatedCardinality)
    }
    
    return nil
}
```

## References
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/reference/specification/metrics/semantic_conventions/)
- [Cardinality Management Guide](https://www.robustperception.io/cardinality-is-key)
- [Metric Design Patterns](https://sre.google/workbook/implementing-slos/)