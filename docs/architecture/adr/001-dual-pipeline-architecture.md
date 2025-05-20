# ADR-001: Dual-Pipeline Architecture for Self-Adjusting Metrics Fabric

Date: 2025-05-20

## Context

The SA-OMF (Phoenix) system requires a mechanism for the OpenTelemetry Collector to dynamically configure itself based on the characteristics of the telemetry it is processing. We need a way for the collector to:

1. Observe its own performance and output metrics
2. Make decisions about configuration changes
3. Apply these changes to its processing pipeline

This raises several architectural questions:

- How should components communicate for self-adjustment?
- How do we prevent circular dependencies between monitoring and processing?
- How can we maintain standard OTel compliance while enabling self-adaptation?
- How can we provide robust guard-rails and safety mechanisms?

## Decision

We will implement a dual-pipeline architecture within a single OpenTelemetry Collector instance:

1. **Data Pathway (Pipeline A)**: Processes host & process metrics through intelligent, dynamically configurable processors. These processors implement a common `UpdateableProcessor` interface.

2. **Control Pathway (Pipeline B)**: Monitors the collector's self-metrics and makes decisions about configuration adjustments. This pathway consists of:
   - A standard `prometheus/self` receiver that scrapes the collector's own metrics endpoint
   - A `metricstransform/self_aggregator` processor to prepare metrics for decision-making
   - A custom `pid_decider` processor that implements PID control loops for each KPI
   - A custom `pic_connector` exporter that converts metric-encoded patches to internal API calls

3. **pic_control Extension**: A central governance layer that:
   - Loads and watches the policy.yaml file
   - Receives, validates, and applies configuration patches to processors
   - Implements guard-rails, rate limiting, and safe mode
   - Generates detailed decision traces for auditability

4. **ConfigPatch**: A standardized data structure for expressing configuration changes, encoded as OTLP metrics for transport from the `pid_decider` to the `pic_control` extension.

## Consequences

### Positive

- **Pure OTel Native**: All components are standard OTel types (receivers, processors, exporters, extensions) with well-defined interfaces.
- **Clean Separation**: Control and data flows are clearly separated, preventing circular dependencies.
- **Explainable Decisions**: All configuration changes are visible as metrics and traces, providing complete auditability.
- **Centralized Governance**: The `pic_control` extension provides a single point for policy enforcement and safety mechanisms.
- **Extensible**: Additional processors implementing the `UpdateableProcessor` interface can be easily added to support new use cases.

### Negative

- **Complexity**: The dual-pipeline architecture adds complexity compared to a simpler, static configuration approach.
- **Potential for Thrashing**: Without careful tuning, the control loops could oscillate or make too-frequent changes.
- **Resource Overhead**: The control pathway consumes additional CPU and memory resources.
- **Startup Sequence**: Care must be taken to ensure that components initialize in the correct order.

### Mitigations

- Implement a graduated approach to autonomy with configurable modes: "shadow" (log only), "advisory" (suggest changes), and "active" (apply changes).
- Enforce a minimum cooldown period between configuration changes to prevent thrashing.
- Implement resource usage guard-rails in the `pic_control` extension to prevent runaway resource consumption.
- Design a "safe mode" that can be automatically triggered if the system detects abnormal behavior.
- Provide extensive metrics and Grafana dashboards to monitor the system's behavior.

## Alternative Approaches Considered

1. **External Control Loop**: Using a separate process to monitor and adjust the collector. Rejected due to complexity of deployment and slower reaction time.

2. **Sidecar Pattern**: Using a small controller sidecar in Kubernetes to manage the collector. Rejected due to platform-specific nature and increased deployment complexity.

3. **Central Control Plane**: Using a central service to manage multiple collectors. While valuable for fleet management, this moves critical adjustment logic away from the edge where it's most needed.

4. **Static Configuration with Overrides**: Using a static baseline with occasional manual overrides. Rejected due to inability to adapt to changing workloads in real-time.

## References

- OpenTelemetry Collector Architecture: https://opentelemetry.io/docs/collector/about/
- PID Controller Theory: https://en.wikipedia.org/wiki/PID_controller
- Service Mesh Auto-Configuration: Various Istio and Linkerd papers

