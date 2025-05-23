# Phoenix Advanced Prometheus Recording Rules

This directory contains Prometheus recording rules that pre-calculate complex metrics for improved performance and advanced analytics.

## Files

- `phoenix_rules.yml` - Basic operational rules and alerts
- `phoenix_advanced_rules.yml` - Advanced analytics and ML-ready features

## Rule Groups

### 1. Cardinality Analysis (`phoenix_cardinality_analysis`)
Pre-calculated metrics for cardinality monitoring and prediction:

- `phoenix:cardinality_zscore` - Statistical anomaly detection using Z-scores
- `phoenix:cardinality_growth_rate` - Rate of cardinality change
- `phoenix:cardinality_explosion_risk` - Risk score (0-100) for cardinality explosion
- `phoenix:cardinality_prediction_1h` - ML-ready prediction for next hour

### 2. Cost Analysis (`phoenix_cost_analysis`)
Financial metrics for cost optimization:

- `phoenix:cost_per_pipeline` - Estimated hourly cost per pipeline in USD
- `phoenix:estimated_monthly_cost_usd` - Total projected monthly cost
- `phoenix:cost_efficiency_score` - Datapoints processed per dollar
- `phoenix:cost_per_million_datapoints` - Unit cost metric
- `phoenix:cardinality_reduction_percentage` - Savings from optimization

### 3. Pipeline Efficiency (`phoenix_pipeline_efficiency`)
Performance and efficiency metrics:

- `phoenix:pipeline_efficiency_percentage` - Success rate of processing
- `phoenix:pipeline_throughput_per_core` - CPU efficiency metric
- `phoenix:memory_efficiency` - Datapoints per MB of memory
- `phoenix:pipeline_queue_saturation` - Queue usage percentage
- `phoenix:processor_effectiveness_score` - Individual processor performance

### 4. ML Features (`phoenix_ml_features`)
Machine learning-ready features:

- `phoenix:hourly_seasonality_component` - Hourly patterns for ML models
- `phoenix:weekly_baseline` - 7-day rolling baseline
- `phoenix:anomaly_score` - Deviation from normal behavior
- `phoenix:trend_strength` - Momentum indicator

### 5. Optimization Impact (`phoenix_optimization_impact`)
Metrics to measure optimization effectiveness:

- `phoenix:optimization_data_loss_percentage` - Data fidelity impact
- `phoenix:optimization_latency_impact` - Processing delay added
- `phoenix:optimization_resource_savings` - Memory/CPU saved

### 6. Health SLI (`phoenix_health_sli`)
Service Level Indicators:

- `phoenix:system_health_score` - Overall health (0-100)
- `phoenix:data_freshness_sli` - Data timeliness indicator
- `phoenix:pipeline_availability_sli` - Uptime percentage
- `phoenix:pipeline_error_rate` - Error ratio

### 7. Capacity Planning (`phoenix_capacity_planning`)
Forward-looking capacity metrics:

- `phoenix:days_until_cardinality_limit` - Runway before hitting limits
- `phoenix:memory_headroom_percentage` - Available memory buffer
- `phoenix:projected_monthly_volume_tb` - Storage planning metric
- `phoenix:required_collector_instances` - Scaling requirements

## Usage Examples

### Dashboard Queries

```promql
# Show cardinality anomalies
phoenix:cardinality_zscore > 2 or phoenix:cardinality_zscore < -2

# Cost optimization opportunities
topk(5, phoenix:cost_per_pipeline)

# Capacity planning alert
phoenix:days_until_cardinality_limit < 7

# Health score visualization
phoenix:system_health_score
```

### Alert Examples

```yaml
- alert: CardinalityAnomalyDetected
  expr: abs(phoenix:cardinality_zscore) > 3
  for: 5m
  annotations:
    summary: "Unusual cardinality pattern detected"

- alert: CostBudgetExceeded
  expr: phoenix:estimated_monthly_cost_usd > 1000
  annotations:
    summary: "Monthly cost projection exceeds budget"

- alert: CapacityPlanningWarning
  expr: phoenix:days_until_cardinality_limit < 14
  annotations:
    summary: "Will reach cardinality limit in {{ $value }} days"
```

## Performance Impact

These recording rules:
- Execute every 30-60 seconds (configurable per group)
- Reduce dashboard query complexity by 80%
- Enable sub-second dashboard loads
- Provide consistent calculations across all uses

## Best Practices

1. **Use Recording Rules When:**
   - Query is used in multiple dashboards
   - Calculation takes > 1 second
   - Result is used in alerts
   - Aggregating high-cardinality data

2. **Naming Convention:**
   - Format: `phoenix:<category>_<metric>`
   - Always prefix with `phoenix:`
   - Use underscores for multi-word names
   - Include units where applicable

3. **Update Frequency:**
   - Critical metrics: 30s
   - Analytics metrics: 60s
   - Capacity planning: 5m

## Monitoring Recording Rules

```promql
# Rule evaluation performance
prometheus_rule_evaluation_duration_seconds{rule_group=~"phoenix.*"}

# Failed evaluations
rate(prometheus_rule_evaluation_failures_total{rule_group=~"phoenix.*"}[5m])

# Number of samples produced
prometheus_rule_group_last_samples{rule_group=~"phoenix.*"}
```

## Maintenance

### Adding New Rules
1. Add to appropriate group or create new group
2. Follow naming convention
3. Document in this README
4. Test in staging first
5. Monitor evaluation performance

### Optimizing Rules
1. Check evaluation duration regularly
2. Combine similar calculations
3. Use appropriate intervals
4. Avoid expensive operations in hot paths

### Debugging
```bash
# Test rule syntax
promtool check rules phoenix_advanced_rules.yml

# Evaluate rule manually
curl -g 'http://prometheus:9090/api/v1/query?query=<rule_expression>'

# Check rule health
curl http://prometheus:9090/api/v1/rules | jq '.data.groups[] | select(.name | contains("phoenix"))'
```