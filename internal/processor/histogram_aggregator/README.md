# Histogram Aggregator Processor

The Histogram Aggregator processor optimizes histograms for OTLP export to systems like New Relic. It performs two main functions:

1. **Bucket Reduction**: Limits the number of buckets in a histogram to a configurable maximum to reduce cardinality and improve performance
2. **Custom Bucketing**: Allows specification of custom bucket boundaries for important metrics to ensure optimal visualization

## Configuration

```yaml
processors:
  histogram_aggregator:
    enabled: true
    max_buckets: 10  # Maximum number of buckets to preserve
    target_processors: ["java", "nginx"]  # Target only specific processes (optional)
    custom_boundaries:  # Custom bucket boundaries for specific metrics
      process.memory.usage:
        - 10000000  # 10MB
        - 50000000  # 50MB  
        - 100000000 # 100MB
        - 500000000 # 500MB
        - 1000000000 # 1GB
      process.cpu.time:
        - 0.1
        - 0.5
        - 1.0
        - 5.0
        - 10.0
```

## When to Use

Use this processor when:

1. You need to optimize histogram metrics for export to OTLP-based systems 
2. Histogram data has too many buckets, causing cardinality issues
3. You want to standardize histogram boundaries across your metrics for better visualization

## Integration with Phoenix SA-OMF

This processor optimizes histogram data specifically for OTLP export to systems like New Relic, ensuring:

1. Cardinality is kept under control
2. Histogram visualization is optimized
3. System resources are conserved by reducing the size of exported data

For process metrics in particular, this provides more efficient representation of memory usage, CPU time, and other histogram-based metrics.