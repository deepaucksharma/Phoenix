# ADR: Consolidate Multiple Processors into a Single Metric Pipeline

## Date

2025-05-20

## Status

Accepted

## Context

The Phoenix monitoring system initially used multiple separate processors for resource filtering and metric transformation:

1. `priority_tagger`: Assigned priority levels to resources based on pattern matching
2. `adaptive_topk`: Selected the top K most important resources based on a metric value
3. `others_rollup`: Aggregated metrics from low-priority resources
4. Various transformation processors: Histogram generation, attribute processing, etc.

This approach had several drawbacks:

1. **Inefficiency**: Each processor required a full iteration over the metrics data, leading to multiple copies and traversals.
2. **Configuration complexity**: Each processor had its own configuration section, making setup and maintenance difficult.
3. **Code duplication**: Similar functionality was implemented multiple times in different processors.
4. **Poor cohesion**: Logically related operations were split across multiple components.
5. **Testing complexity**: Testing the interaction between multiple processors was challenging.

## Decision

We will consolidate these processors into a single `metric_pipeline` processor that performs resource filtering and metric transformation in a unified manner:

1. Create a new `metric_pipeline` processor that combines:
   - Resource filtering (priority-based, top-K, and hybrid approaches)
   - Rollup aggregation for non-priority resources
   - Histogram generation
   - Attribute processing
   
2. Implement a standardized configuration structure that:
   - Organizes related settings logically
   - Provides a single point of configuration
   - Maintains backward compatibility where possible
   
3. Introduce a comprehensive self-metrics system that:
   - Tracks key performance indicators
   - Provides feedback for adaptive controllers
   - Enables detailed troubleshooting and visualization

4. Remove the redundant processors:
   - `priority_tagger`
   - `adaptive_topk`
   - `others_rollup`
   - `resource_filter` (as a standalone processor)

## Consequences

### Positive

1. **Performance improvement**: Single-pass processing reduces data copying and traversals.
2. **Simplified configuration**: Users only need to configure one processor instead of multiple separate ones.
3. **Code reduction**: Elimination of redundant code reduces maintenance burden.
4. **Improved cohesion**: Related functionality is now grouped together logically.
5. **Better observability**: Integrated self-metrics provide comprehensive insights into processor behavior.
6. **Easier testing**: Testing a single processor is simpler than testing interactions between multiple processors.

### Negative

1. **Increased processor complexity**: The unified processor is more complex than any of the individual processors it replaces.
2. **Migration effort**: Existing configurations and code that reference the old processors need to be updated.
3. **Loss of fine-grained control**: Some advanced users might prefer the flexibility of individual processors.

### Mitigations

1. **Well-structured code**: Keep the unified processor well-organized internally with clear separation of concerns.
2. **Migration guides**: Provide documentation to help users update their configurations.
3. **Comprehensive testing**: Ensure thorough test coverage of all features and edge cases.
4. **Self-metrics**: Implement detailed self-metrics to provide visibility into the processor's internal behavior.

## Implementation Notes

The implementation of this consolidation will follow these steps:

1. Create the new `metric_pipeline` processor with all required functionality.
2. Update the configuration schema to support the new unified approach.
3. Implement comprehensive self-metrics for the new processor.
4. Add thorough tests for the new processor.
5. Update documentation to reflect the new approach.
6. Identify and update any code that references the old processors.
7. Remove the redundant processors and associated files.

## References

- [Cleanup Summary Document](/docs/cleanup-summary.md)
- [Metric Pipeline Self-Metrics Documentation](/docs/components/processors/metric_pipeline_self_metrics.md)
- [Cleanup Guide](/docs/cleanup-redundant-processors.md)