# Phoenix Test Scenarios

This document provides an overview of the integration and end-to-end test scenarios implemented for the Phoenix / SA-OMF project. These tests verify core functionality, resilience, and behavior of the system under various conditions.

## Test Categories

### Dual-Pipeline Wiring Tests

These tests verify the interaction between the data and control pipelines.

| ID | Scenario | Purpose | Location |
|----|----------|---------|----------|
| PIPE-05 | Control Pipeline Missing Self-Metrics | Verify PID decider and pic_connector handle missing self-metrics gracefully | `test/e2e/pipelines/missing_self_metrics_test.go` ✅ |
| PIPE-06 | Dual Pipeline Interaction | Verify data pipeline and control pipeline properly interact and influence each other | `test/e2e/pipelines/dual_pipeline_interaction_test.go` ✅ |
| PIPE-07 | Multiple PID Decider Instances | Verify independent PID decider instances work correctly without interference | `test/e2e/pipelines/multiple_pid_deciders_test.go` |

### Policy & Config-Patch Governance Tests

These tests verify the behavior of the policy management and configuration patch systems.

| ID | Scenario | Purpose | Location |
|----|----------|---------|----------|
| POL-05 | Patch with Invalid Parameter Value Type | Verify pic_control correctly validates and rejects patches with wrong types | `test/e2e/policy/invalid_parameter_type_test.go` ✅ |
| POL-06 | Patch Targeting Non-UpdateableProcessor | Verify graceful handling of patches targeting non-updateable processors | `test/e2e/policy/non_updateable_target_test.go` ✅ |
| POL-07 | Rate Limiting of Configuration Patches | Verify pic_control correctly applies rate limiting to patches | `test/e2e/policy/rate_limit_test.go` ✅ |
| POL-08 | Policy with Conflicting PID Output Targets | Verify conflict resolution for multiple controllers targeting same parameter | `test/e2e/policy/conflicting_targets_test.go` |
| POL-09 | Safe Mode Policy Application | Verify pic_control applies safe mode configurations correctly | `test/e2e/policy/safe_mode_test.go` |
| POL-10 | Policy File Syntax Error | Verify pic_control handles corrupt or invalid policy files | `test/e2e/policy/policy_syntax_error_test.go` |

### PID Controller Behavior Tests

These tests verify the behavior of the PID controllers and adaptive decision making.

| ID | Scenario | Purpose | Location |
|----|----------|---------|----------|
| PID-05 | KPI Metric Flapping/Unstable | Verify PID output is damped for noisy input and hysteresis prevents thrashing | `test/e2e/pid/metric_flapping_test.go` |
| PID-06 | Controller Disabled/Enabled via Policy | Verify controllers can be dynamically disabled and re-enabled | `test/e2e/pid/controller_toggle_test.go` |
| PID-07 | PID Output Parameter Out of Bounds | Verify PID output is clamped to defined min/max bounds | `test/e2e/pid/output_parameter_bounds_test.go` ✅ |
| PID-08 | PID Controller Windup Protection | Verify PID controller prevents integral windup when output is saturated | `test/e2e/pid/windup_protection_test.go` ✅ |
| PID-09 | Stall Detection & Bayesian Fallback | Verify system detects stalled controllers and tries alternative approaches | `test/e2e/pid/stall_detection_test.go` |

### Component-Level Integration Tests

These tests verify specific component behaviors and interactions.

| ID | Scenario | Purpose | Location |
|----|----------|---------|----------|
| COMP-PRC-03 | Cardinality Guardian with High Cardinality | Verify cardinality guardian correctly mitigates high cardinality | `test/e2e/components/cardinality_guardian_test.go` |
| COMP-PRC-04 | Reservoir Sampler with Varying Input | Verify reservoir sampler dynamically adjusts sample size | `test/e2e/components/reservoir_sampler_test.go` |
| COMP-PRC-05 | Others Rollup with Priority Tagging | Verify others_rollup correctly aggregates low priority metrics | `test/e2e/components/others_rollup_test.go` |
| COMP-PRC-06 | Processor Pipeline Sequence | Verify all processors work correctly in sequence | `test/e2e/components/processor_sequence_test.go` |

### Safety Tests

These tests verify the behavior of the safety mechanisms.

| ID | Scenario | Purpose | Location |
|----|----------|---------|----------|
| SAFETY-03 | Override Safety Thresholds | Verify temporary override of safety thresholds for urgent patches | `test/e2e/safety/override_thresholds_test.go` ✅ |

### Distributed Tests

These tests verify the behavior of distributed deployment features.

| ID | Scenario | Purpose | Location |
|----|----------|---------|----------|
| DISTR-02 | Distributed Coordination | Verify coordination of configuration changes across multiple nodes | `test/e2e/distributed/coordinator_test.go` ✅ |

### Resilience & Chaos Tests

These tests verify system behavior under failure conditions.

| ID | Scenario | Purpose | Location |
|----|----------|---------|----------|
| CHAOS-POLICY | Corrupt Policy File During Hot-Reload | Verify pic_control handles corrupt policy files gracefully | `test/e2e/chaos/corrupt_policy_test.go` ✅ |
| CHAOS-PATCH | Flood of Conflicting Patches | Verify rate limiting and hysteresis prevent system thrashing | `test/e2e/chaos/patch_flood_test.go` |
| CHAOS-DISK | Disk Full | Verify system remains operational with full disk | `test/e2e/chaos/disk_full_test.go` |

### Advanced Processor Tests

These tests verify advanced processing capabilities.

| ID | Scenario | Purpose | Location |
|----|----------|---------|----------|
| ADV-PCL-01 | Process Context Learner Hierarchy | Verify context learner correctly identifies process hierarchies | `test/e2e/advanced/process_context_learner_test.go` |
| ADV-PCL-02 | Context-Based Priority Elevation | Verify priority tagging uses learned context | `test/e2e/advanced/context_priority_test.go` |
| ADV-MTAE-01 | Multi-Temporal Adaptive Engine Forecasting | Verify forecasting enables proactive adjustments | `test/e2e/advanced/mtae_forecast_test.go` |

## Test Implementation Summary

### Completed Tests (✅)

We have successfully implemented the following tests:

1. **Policy Tests**:
   - POL-05: Verifies rejection of patches with incorrect parameter types
   - POL-06: Verifies rejection of patches targeting non-UpdateableProcessor components
   - POL-07: Verifies rate limiting and cooldown for configuration patches

2. **PID Controller Tests**:
   - PID-07: Verifies proper clamping of output values to min/max bounds
   - PID-08: Verifies integral windup protection functionality

3. **Pipeline Tests**:
   - PIPE-05: Verifies graceful handling of missing self-metrics
   - PIPE-06: Verifies dual pipeline interaction and feedback loops

4. **Safety Tests**:
   - SAFETY-03: Verifies temporary override of safety thresholds for urgent patches

5. **Distributed Tests**:
   - DISTR-02: Verifies coordination of configuration changes across multiple nodes

6. **Chaos Tests**:
   - CHAOS-POLICY: Verifies resilience to corrupt policy files during hot reload

### Test Coverage

The implemented tests provide coverage for key functionality:
- Policy validation and error handling
- PID controller behavior and protection mechanisms
- Interaction between data and control pipelines
- Safety overrides for urgent situations
- Distributed coordination
- Resilience to corrupt configurations

## Running Tests

### Running All Tests

```bash
make test
```

### Running Specific Categories

```bash
make test CATEGORY=policy
make test CATEGORY=pid
make test CATEGORY=safety
make test CATEGORY=distributed
```

### Running Specific Scenarios

```bash
go test -v ./test/e2e/policy/invalid_parameter_type_test.go
go test -v ./test/e2e/pid/windup_protection_test.go
go test -v ./test/e2e/safety/override_thresholds_test.go
```

## Test Utilities

The following utilities are available for test implementation:

- `testutils.MockUpdateableProcessor`: Mock implementation of the UpdateableProcessor interface
- `testutils.MockHost`: Mock implementation of the component.Host interface
- `testutils.MockMetricsProvider`: Mock metrics provider for safety monitor testing
- `metrics.MetricsCollector`: Utility for collecting and analyzing metrics
- Internal processor test helpers for direct testing without the full pipeline

## Future Enhancements

Future enhancements to the test suite could include:

1. **Performance Testing**: Add benchmarks to track processor performance.
2. **Load Testing**: Verify behavior under high metric volume conditions.
3. **Cross-Component Interaction**: Add more tests for interactions between different components.
4. **Edge Case Testing**: Increase coverage of boundary and error conditions.
5. **Regression Testing**: Create a regression test suite for any bugs discovered.

## Adding New Tests

When adding a new test:

1. Add the test file to the appropriate category directory under `test/e2e/`
2. Include a comment with the scenario ID (e.g., `// Scenario: POL-05`)
3. Update this document with the new test information
4. Add any necessary test utilities to `test/testutils/`
5. Mark implemented tests with ✅ in this document