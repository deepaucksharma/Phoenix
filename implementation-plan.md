# SA-OMF v7.0: Ultra-Detailed Architecture and Agentic Implementation Plan

This document provides a comprehensive implementation plan for the Self-Aware OpenTelemetry Metrics Fabric (SA-OMF) v7.0, breaking down the journey from zero to a complete, production-ready system. The plan is structured into sprints spanning 18 months, with detailed coding tasks, testing strategies, and deployment milestones.

**Project Codename**: Phoenix (Self-optimizing OTel collector)  
**Target Implementation Timeline**: 18 months  
**Repository Structure**: Monorepo with modular packages  

## Table of Contents

1. [Repository Structure](#repository-structure)
2. [Core Components Overview](#core-components-overview)
3. [Implementation Phases](#implementation-phases)
   - [Phase 1: Foundation (Months 0-4)](#phase-1-foundation-months-0-4)
   - [Phase 2: Enhanced Processors (Months 5-9)](#phase-2-enhanced-processors-months-5-9)
   - [Phase 3: Advanced Intelligence (Months 10-14)](#phase-3-advanced-intelligence-months-10-14)
   - [Phase 4: Production Hardening (Months 15-18)](#phase-4-production-hardening-months-15-18)
4. [Detailed Component Implementation](#detailed-component-implementation)
   - [Core Interfaces](#core-interfaces)
   - [pic_control Extension](#pic_control-extension)
   - [Data Pathway Processors](#data-pathway-processors)
   - [Control Pathway Components](#control-pathway-components)
5. [Testing Strategy](#testing-strategy)
6. [Deployment and Operations](#deployment-and-operations)
7. [Appendix: Code Templates](#appendix-code-templates)

## Repository Structure

```
sa-omf/
├── cmd/
│   └── sa-omf-otelcol/             # Main binary entrypoint
├── internal/
│   ├── interfaces/                  # Core interfaces (UpdateableProcessor, etc.)
│   ├── extension/
│   │   └── piccontrolext/           # pic_control implementation
│   ├── connector/
│   │   └── picconnector/            # pic_connector implementation
│   ├── processor/                   # All custom processors
│   │   ├── prioritytagger/
│   │   ├── adaptivepid/             # pid_decider
│   │   ├── adaptivesampler/
│   │   ├── cardinalityguardian/
│   │   ├── reservoirsampler/
│   │   ├── othersrollup/
│   │   ├── contextlearner/
│   │   ├── semanticcorrelator/
│   │   └── multitemporal/
│   ├── control/                     # Control logic helpers
│   │   ├── pid/                     # PID controller implementation
│   │   ├── bayesian/                # Bayesian optimization (Phase 3)
│   │   ├── configpatch/             # ConfigPatch validation & application
│   │   └── safety/                  # Safe mode detection & mitigation
│   └── safetynet/                   # Guard rails implementation
├── pkg/                             # Reusable packages
│   ├── metrics/                     # Self-metrics definitions
│   ├── util/
│   │   ├── hll/                     # HyperLogLog implementation
│   │   ├── reservoir/               # Reservoir sampling algorithms
│   │   ├── topk/                    # Space-saving algorithm
│   │   ├── causality/               # Granger/TE implementations
│   │   └── timeseries/              # Time series forecasting
│   └── policy/                      # Policy schema & validation
├── test/
│   ├── e2e/                         # End-to-end test scenarios
│   ├── benchmark/                   # Performance benchmarks
│   └── chaos/                       # Chaos testing
├── dashboards/                      # Grafana dashboards
├── deploy/
│   ├── kubernetes/                  # K8s manifests
│   └── docker/                      # Dockerfile
└── docs/                            # Documentation
    ├── design/                      # Architecture documentation
    ├── examples/                    # Example configurations
    └── tutorials/                   # User guides
```

## Core Components Overview

Before diving into the implementation plan, let's solidify our understanding of the core components and their interactions:

1. **Dual Pipeline Architecture**:
   - **Data Pathway (Pipeline A)**: Processes host & process metrics
   - **Control Pathway (Pipeline B)**: Self-monitors and adjusts Pipeline A

2. **Key Components**:
   - **pic_control (Extension)**: Central governance layer
   - **UpdateableProcessor (Interface)**: Contract for dynamic configuration
   - **pid_decider (Processor)**: Generates configuration patches
   - **pic_connector (Exporter)**: Connects pid_decider to pic_control

3. **Policy Management**:
   - **policy.yaml**: Source of truth for configs, KPIs, guard-rails
   - Hot-reloaded by pic_control
   - Optionally managed via OpAMP

4. **Self-Monitoring & Explainability**:
   - aemf_* metrics
   - OTLP Traces & Exemplars for decisions

## Implementation Phases

### Phase 1: Foundation (Months 0-4)

**Goal**: Establish the core framework and minimal viable control loop.

#### Sprint 1.1: Core Interfaces & Framework (Weeks 1-3)

**Tasks**:
1. Set up project repository structure, CI/CD pipeline, testing framework
2. Define and implement core interfaces:
   - `UpdateableProcessor`
   - `ConfigPatch`
   - `ConfigStatus`
3. Implement policy.yaml schema and validation

**Deliverables**:
- Repository scaffold
- Core interfaces with tests
- Basic policy schema

#### Sprint 1.2: pic_control Extension (Weeks 4-7)

**Tasks**:
1. Implement basic pic_control extension:
   - Policy file watching
   - ConfigPatch validation
   - UpdateableProcessor registry
   - Safe mode skeleton
2. Create test harness for pic_control

**Deliverables**:
- Functional pic_control extension
- Test harness for manual testing

#### Sprint 1.3: First Smart Processor - priority_tagger (Weeks 8-10)

**Tasks**:
1. Implement priority_tagger processor:
   - Basic rule-based process tagging
   - UpdateableProcessor implementation
   - Self-metrics emission
2. Integrate with pic_control
3. Write unit and integration tests

**Deliverables**:
- Fully functional priority_tagger processor
- Integration tests with pic_control

#### Sprint 1.4: Control Pipeline Basics (Weeks 11-14)

**Tasks**:
1. Implement prometheus/self receiver configuration
2. Create self_metrics_aggregator using metricstransformprocessor
3. Implement basic pid_decider (single PID loop)
4. Implement pic_connector
5. Integration test for basic control loop

**Deliverables**:
- End-to-end control flow for a single parameter
- Basic self-metrics aggregation

#### Sprint 1.5: First Adaptive Processor - adaptive_topk (Weeks 15-17)

**Tasks**:
1. Implement Space-Saving algorithm in pkg/util/topk
2. Create adaptive_topk processor:
   - Top-K selection with dynamic K
   - UpdateableProcessor implementation
   - Coverage score metric
3. Connect to pid_decider via control loop
4. End-to-end testing

**Deliverables**:
- Functional topk processor
- End-to-end test of adaptive K adjustment based on coverage score

### Phase 2: Enhanced Processors (Months 5-9)

**Goal**: Implement cardinality control and statistical sampling to achieve target cardinality reduction.

#### Sprint 2.1: Cardinality Guardian Implementation (Weeks 18-21)

**Tasks**:
1. Implement HyperLogLog algorithm in pkg/util/hll
2. Create cardinality_guardian processor:
   - Per-metric cardinality estimation
   - Dynamic threshold adjustment
   - Mitigation strategies (hash/drop)
   - UpdateableProcessor implementation
3. Add PID loop in pid_decider for cardinality management
4. Integration testing

**Deliverables**:
- Fully functional cardinality_guardian
- Multi-KPI PID control

#### Sprint 2.2: Reservoir Sampler Implementation (Weeks 22-25)

**Tasks**:
1. Implement stratified reservoir sampling algorithms in pkg/util/reservoir
2. Create reservoir_sampler processor:
   - Stratified sampling with adjustable reservoir sizes
   - Statistical guarantees for data representation
   - Self-metrics for sampling coverage
3. Add another PID loop for reservoir size adjustment
4. Integration testing

**Deliverables**:
- Functional reservoir_sampler processor
- Three-way PID control (adaptive_topk, cardinality_guardian, reservoir_sampler)

#### Sprint 2.3: Others Rollup Implementation (Weeks 26-28)

**Tasks**:
1. Implement others_rollup processor:
   - Aggregation of non-priority processes
   - UpdateableProcessor implementation for aggregation function selection
2. End-to-end testing with full Phase 2 processor chain

**Deliverables**:
- Functional others_rollup processor
- Complete data pathway for basic intelligent processing

#### Sprint 2.4: Enhanced Safety & Guard-Rails (Weeks 29-32)

**Tasks**:
1. Improve pic_control safety mechanisms:
   - Resource monitoring enhancements
   - Graduated safe mode levels
   - Targeted mitigation strategies
   - PID integrator reset
2. Implement throttling and patch cooling
3. Stress testing and chaos experiments

**Deliverables**:
- Robust safety guard-rails
- Documented chaos test results

#### Sprint 2.5: Initial Dashboards & Visualizations (Weeks 33-36)

**Tasks**:
1. Create core Grafana dashboards:
   - Autonomy Pulse
   - Coverage vs. Cardinality
   - Decision Stream
   - Processor overview dashboards
2. Deploy test environment with Prometheus + Grafana
3. End-to-end testing of full Phase 2 capabilities

**Deliverables**:
- Core Grafana dashboards
- Test environment deployment manifests
- Phase 2 functional demo

### Phase 3: Advanced Intelligence (Months 10-14)

**Goal**: Implement learning capabilities and fleet management.

#### Sprint 3.1: Process Context Learner (Weeks 37-40)

**Tasks**:
1. Implement process relationship tracking
2. Implement process importance scoring algorithms
3. Create process_context_learner processor:
   - Parent/child relationship tracking
   - PageRank-style importance calculation
   - UpdateableProcessor implementation
4. Integrate learner outputs with priority_tagger
5. Integration testing

**Deliverables**:
- Functional process_context_learner
- Integration with priority_tagger

#### Sprint 3.2: pic_control OpAMP Integration (Weeks 41-45)

**Tasks**:
1. Add OpAMP client to pic_control:
   - Remote policy.yaml management
   - Status reporting
   - Secure channel setup (mTLS)
2. Implement remote policy validation and application
3. Integration testing with OpAMP server

**Deliverables**:
- Functional OpAMP integration
- Remote management capabilities

#### Sprint 3.3: Bayesian Optimization Addition (Weeks 46-49)

**Tasks**:
1. Implement Bayesian optimization algorithms in pkg/util/bayesian
2. Enhance pid_decider with Bayesian fallback:
   - Stall detection
   - Multi-objective optimization
   - Parameter exploration
3. Testing of combined PID + Bayesian approach

**Deliverables**:
- Bayesian optimization capability
- Enhanced multi-objective control

#### Sprint 3.4: Semantic Correlator Foundations (Weeks 50-53)

**Tasks**:
1. Implement Granger causality and Transfer Entropy algorithms in pkg/util/causality
2. Create initial semantic_correlator processor:
   - Basic causality detection
   - Self-metrics for detected relationships
3. Position after reservoir_sampler to minimize CPU impact
4. Initial testing

**Deliverables**:
- Initial semantic_correlator implementation
- Integration with existing processors

#### Sprint 3.5: Fleet-Scale Testing & Dashboards (Weeks 54-57)

**Tasks**:
1. Enhance Grafana dashboards for Phase 3 capabilities
2. Create OpAMP-based fleet management dashboard
3. Set up multi-node test environment
4. Fleet-scale testing

**Deliverables**:
- Enhanced dashboards
- Fleet management capabilities
- Multi-node test results

### Phase 4: Production Hardening (Months 15-18)

**Goal**: Complete advanced intelligence components and prepare for production deployment.

#### Sprint 4.1: Multi-Temporal Adaptive Engine (Weeks 58-61)

**Tasks**:
1. Implement time series forecasting algorithms in pkg/util/timeseries
2. Create multi_temporal_adaptive_engine processor:
   - Multi-horizon forecasting
   - Anomaly detection
   - Seasonality identification
3. Integrate with pid_decider for predictive control
4. Testing and performance optimization

**Deliverables**:
- Functional multi_temporal_adaptive_engine
- Predictive control capabilities

#### Sprint 4.2: Semantic Correlator Completion (Weeks 62-65)

**Tasks**:
1. Complete semantic_correlator processor:
   - Causal relationship visualization
   - Dynamic correlation thresholds
   - Integration with multi_temporal_adaptive_engine
2. Performance optimization
3. Testing with realistic workloads

**Deliverables**:
- Complete semantic_correlator implementation
- Causality visualization

#### Sprint 4.3: Performance Optimization & Benchmarking (Weeks 66-69)

**Tasks**:
1. Comprehensive benchmarking suite:
   - Component-level benchmarks
   - End-to-end benchmarks
   - Resource utilization profiling
2. Optimize hotspots and high-resource components
3. Implement caching and other performance improvements

**Deliverables**:
- Performance optimization report
- Benchmark suite

#### Sprint 4.4: Final Security & Chaos Testing (Weeks 70-73)

**Tasks**:
1. Security review and hardening:
   - OpAMP authentication
   - Policy verification
   - Privilege review
2. Extended chaos testing:
   - Config oscillation
   - Resource starvation
   - Component failure
3. Documentation of security findings and recommendations

**Deliverables**:
- Security review report
- Extended chaos test results

## Detailed Component Implementation

This section provides in-depth guidance for implementing key components.

### Core Interfaces

#### UpdateableProcessor Interface

```go
// File: internal/interfaces/updateable_processor.go

package interfaces

import (
    "context"
    "go.opentelemetry.io/collector/component"
)

// ConfigPatch defines a proposed change to a processor's configuration
type ConfigPatch struct {
    PatchID             string      `mapstructure:"patch_id"`             // Unique ID for this patch attempt
    TargetProcessorName component.ID `mapstructure:"target_processor_name"` // Name of the processor to update
    ParameterPath       string      `mapstructure:"parameter_path"`       // Dot-separated path to the parameter
    NewValue            interface{} `mapstructure:"new_value"`            // The new value for the parameter
    Reason              string      `mapstructure:"reason"`               // Why this patch is proposed
    Severity            string      `mapstructure:"severity"`             // e.g., "normal", "urgent", "safety_override"
    Source              string      `mapstructure:"source"`               // e.g., "pid_decider", "opamp", "manual_api"
    Timestamp           int64       `mapstructure:"timestamp"`            // Unix timestamp when patch was created
    TTLSeconds          int         `mapstructure:"ttl_seconds"`          // Time-to-live in seconds
}

// ConfigStatus provides current operational parameters of an UpdateableProcessor
type ConfigStatus struct {
    Parameters map[string]interface{} `mapstructure:"parameters"` // Current values of tunable parameters
    Enabled    bool                   `mapstructure:"enabled"`    // Whether the processor is currently enabled
}

// UpdateableProcessor defines the interface for processors that can be dynamically reconfigured
type UpdateableProcessor interface {
    component.Component // Embed standard component interface

    // OnConfigPatch applies a configuration change.
    // Returns error if patch cannot be applied.
    OnConfigPatch(ctx context.Context, patch ConfigPatch) error

    // GetConfigStatus returns the current effective configuration.
    GetConfigStatus(ctx context.Context) (ConfigStatus, error)
}
```

#### Policy Schema

```go
// File: pkg/policy/schema.go

package policy

import (
    "github.com/xeipuuv/gojsonschema"
)

// PolicySchema defines the JSONSchema for validating policy.yaml
var PolicySchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["global_settings", "processors_config", "pid_decider_config", "pic_control_config", "service"],
  "properties": {
    "global_settings": {
      "type": "object",
      "required": ["autonomy_level", "collector_cpu_safety_limit_mcores", "collector_rss_safety_limit_mib"],
      "properties": {
        "autonomy_level": {
          "type": "string",
          "enum": ["shadow", "advisory", "active"]
        },
        "collector_cpu_safety_limit_mcores": {
          "type": "integer",
          "minimum": 100,
          "maximum": 2000
        },
        "collector_rss_safety_limit_mib": {
          "type": "integer",
          "minimum": 100,
          "maximum": 1000
        }
      }
    },
    "processors_config": {
      "type": "object",
      "properties": {
        "priority_tagger": { "type": "object" },
        "process_context_learner": { "type": "object" },
        "semantic_correlator": { "type": "object" },
        "multi_temporal_adaptive_engine": { "type": "object" },
        "adaptive_topk": { "type": "object" },
        "cardinality_guardian": { "type": "object" },
        "reservoir_sampler": { "type": "object" },
        "others_rollup": { "type": "object" }
      }
    },
    "pid_decider_config": {
      "type": "object",
      "required": ["controllers"],
      "properties": {
        "controllers": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["name", "enabled", "kpi_metric_name", "kpi_target_value", "kp", "output_config_patches"],
            "properties": {
              "name": { "type": "string" },
              "enabled": { "type": "boolean" },
              "kpi_metric_name": { "type": "string" },
              "kpi_target_value": { "type": "number" },
              "kp": { "type": "number" },
              "ki": { "type": "number" },
              "kd": { "type": "number" },
              "integral_windup_limit": { "type": "number" },
              "hysteresis_percent": { "type": "number" },
              "output_config_patches": {
                "type": "array",
                "items": {
                  "type": "object",
                  "required": ["target_processor_name", "parameter_path", "change_scale_factor"],
                  "properties": {
                    "target_processor_name": { "type": "string" },
                    "parameter_path": { "type": "string" },
                    "change_scale_factor": { "type": "number" },
                    "min_value": { "type": "number" },
                    "max_value": { "type": "number" }
                  }
                }
              }
            }
          }
        }
      }
    },
    "pic_control_config": {
      "type": "object",
      "required": ["policy_file_path", "max_patches_per_minute", "patch_cooldown_seconds", "safe_mode_processor_configs"],
      "properties": {
        "policy_file_path": { "type": "string" },
        "max_patches_per_minute": { 
          "type": "integer",
          "minimum": 1,
          "maximum": 60
        },
        "patch_cooldown_seconds": {
          "type": "integer",
          "minimum": 1,
          "maximum": 600
        },
        "safe_mode_processor_configs": { "type": "object" },
        "opamp_client_config": { "type": "object" }
      }
    },
    "service": { "type": "object" }
  }
}`

// ValidatePolicy validates the provided policy against the schema
func ValidatePolicy(policyContent []byte) error {
    // Implementation details...
}
```

### pic_control Extension

```go
// File: internal/extension/piccontrolext/extension.go

package piccontrolext

import (
    "context"
    "sync"
    "time"
    
    "go.opentelemetry.io/collector/component"
    "go.opentelemetry.io/collector/extension"
    
    "github.com/yourorg/sa-omf/internal/interfaces"
    "github.com/yourorg/sa-omf/pkg/policy"
)

type picControlSettings struct {
    PolicyFilePath       string                 `mapstructure:"policy_file_path"`
    MaxPatchesPerMinute  int                    `mapstructure:"max_patches_per_minute"`
    PatchCooldownSeconds int                    `mapstructure:"patch_cooldown_seconds"`
    SafeModeConfigs      map[string]interface{} `mapstructure:"safe_mode_processor_configs"`
    OpAMPConfig          *OpAMPClientConfig     `mapstructure:"opamp_client_config"`
}

type picControlExtension struct {
    settings        picControlSettings
    host            component.Host
    processors      map[component.ID]interfaces.UpdateableProcessor
    policyWatcher   *policyWatcher
    safetyMonitor   *safetyMonitor
    patchHistory    []interfaces.ConfigPatch
    patchRateLimiter *rateLimiter
    safeMode        bool
    lock            sync.RWMutex
    metricsEmitter  *metricsEmitter
    tracer          *decisionsTracer
}

// Function implementations for extension methods:
// - Start: Initialize, register processors, start file watcher
// - Shutdown: Clean up resources
// - SubmitConfigPatch: Process incoming ConfigPatch
// - validatePatch: Check if patch is valid
// - applyPatch: Apply patch to target processor
// - enterSafeMode: Detect and handle resource issues
// etc.
```

### Data Pathway Processors

For brevity, we'll show just the adaptive_topk implementation as it's a critical component:

```go
// File: internal/processor/adaptiveTopk/processor.go

package adaptiveTopk

import (
    "context"
    "sync"
    
    "go.opentelemetry.io/collector/component"
    "go.opentelemetry.io/collector/processor"
    "go.opentelemetry.io/collector/pdata/pmetric"
    
    "github.com/yourorg/sa-omf/internal/interfaces"
    "github.com/yourorg/sa-omf/pkg/util/topk"
)

type topkSettings struct {
    KValue          int    `mapstructure:"k_value"`
    KMin            int    `mapstructure:"k_min"`
    KMax            int    `mapstructure:"k_max"`
    ResourceMetric  string `mapstructure:"resource_metric"`
    Enabled         bool   `mapstructure:"enabled"`
}

type topkProcessor struct {
    settings  topkSettings
    topkAlgo  *topk.SpaceSaving
    lock      sync.RWMutex
    metrics   *processorMetrics
}

// Start, Shutdown, ConsumeMetrics implementations

// UpdateableProcessor interface implementation
func (p *topkProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
    if patch.ParameterPath != "k_value" {
        return fmt.Errorf("unknown parameter: %s", patch.ParameterPath)
    }
    
    newK, ok := patch.NewValue.(int)
    if !ok {
        return fmt.Errorf("invalid value type for k_value: %T", patch.NewValue)
    }
    
    if newK < p.settings.KMin || newK > p.settings.KMax {
        return fmt.Errorf("k_value out of allowed range [%d, %d]: %d", 
                         p.settings.KMin, p.settings.KMax, newK)
    }
    
    p.lock.Lock()
    defer p.lock.Unlock()
    
    // Apply the new K value
    p.settings.KValue = newK
    p.metrics.ReconfigurationTotal.Add(context.Background(), 1)
    p.metrics.CurrentK.Record(context.Background(), int64(newK))
    
    return nil
}

func (p *topkProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
    p.lock.RLock()
    defer p.lock.RUnlock()
    
    return interfaces.ConfigStatus{
        Parameters: map[string]interface{}{
            "k_value": p.settings.KValue,
        },
        Enabled: p.settings.Enabled,
    }, nil
}

// processorMetrics implements the metrics for this processor
type processorMetrics struct {
    ReconfigurationTotal  metric.Int64Counter
    CurrentK              metric.Int64ValueRecorder
    CoveragePct           metric.Float64ValueRecorder
    // Other metrics
}
```

### Control Pathway Components

#### pid_decider Processor

```go
// File: internal/processor/adaptivepid/processor.go

package adaptivepid

import (
    "context"
    "sync"
    "time"
    
    "go.opentelemetry.io/collector/component"
    "go.opentelemetry.io/collector/processor"
    "go.opentelemetry.io/collector/pdata/pmetric"
    
    "github.com/yourorg/sa-omf/internal/interfaces"
    "github.com/yourorg/sa-omf/internal/control/pid"
)

type controllerConfig struct {
    Name               string                `mapstructure:"name"`
    Enabled            bool                  `mapstructure:"enabled"`
    KPIMetricName      string                `mapstructure:"kpi_metric_name"`
    KPITargetValue     float64               `mapstructure:"kpi_target_value"`
    KP                 float64               `mapstructure:"kp"`
    KI                 float64               `mapstructure:"ki"`
    KD                 float64               `mapstructure:"kd"`
    IntegralWindupLimit float64              `mapstructure:"integral_windup_limit"`
    HysteresisPercent  float64               `mapstructure:"hysteresis_percent"`
    OutputConfigPatches []outputConfigPatch  `mapstructure:"output_config_patches"`
}

type outputConfigPatch struct {
    TargetProcessorName string  `mapstructure:"target_processor_name"`
    ParameterPath       string  `mapstructure:"parameter_path"`
    ChangeScaleFactor   float64 `mapstructure:"change_scale_factor"`
    MinValue            float64 `mapstructure:"min_value"`
    MaxValue            float64 `mapstructure:"max_value"`
}

type pidDeciderProcessor struct {
    controllers        []*controller
    lock               sync.RWMutex
    lastValues         map[string]float64    // Last observed values for each KPI
    lastPatchTime      map[string]time.Time  // Last patch time for each parameter
    metrics            *processorMetrics
}

type controller struct {
    config          controllerConfig
    pidController   *pid.Controller
    lastOutputs     map[string]float64  // Last output for each parameter
}

// ConsumeMetrics processes incoming metrics and generates patches based on PID control
func (p *pidDeciderProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
    p.lock.Lock()
    defer p.lock.Unlock()
    
    // Extract KPI values from incoming metrics
    kpiValues := extractKPIValues(md)
    
    // Process each controller
    for _, ctrl := range p.controllers {
        if !ctrl.config.Enabled {
            continue
        }
        
        // Get current KPI value
        kpiValue, found := kpiValues[ctrl.config.KPIMetricName]
        if !found {
            continue // KPI metric not found in this batch
        }
        
        // Calculate error
        error := ctrl.config.KPITargetValue - kpiValue
        
        // Update PID controller
        output := ctrl.pidController.Compute(error)
        
        // Generate ConfigPatch for each output parameter
        for _, outConfig := range ctrl.config.OutputConfigPatches {
            // Apply scaling factor to the output
            scaledOutput := output * outConfig.ChangeScaleFactor
            
            // Check last output against hysteresis
            lastOutput := ctrl.lastOutputs[outConfig.ParameterPath]
            changePercent := (scaledOutput - lastOutput) / lastOutput * 100
            if abs(changePercent) < ctrl.config.HysteresisPercent {
                continue // Change too small, skip
            }
            
            // Clamp the value
            newValue := lastOutput + scaledOutput
            if newValue < outConfig.MinValue {
                newValue = outConfig.MinValue
            } else if newValue > outConfig.MaxValue {
                newValue = outConfig.MaxValue
            }
            
            // Generate patch
            patch := interfaces.ConfigPatch{
                PatchID:             generateUUID(),
                TargetProcessorName: component.MustNewIDFromString(outConfig.TargetProcessorName),
                ParameterPath:       outConfig.ParameterPath,
                NewValue:            newValue,
                Reason:              generateReason(ctrl.config.Name, error, output),
                Severity:            "normal",
                Source:              "pid_decider",
                Timestamp:           time.Now().Unix(),
                TTLSeconds:          300, // 5 minute TTL
            }
            
            // Emit as metric with attributes
            emitPatchAsMetric(ctx, p.metrics, patch)
            
            // Update last output value
            ctrl.lastOutputs[outConfig.ParameterPath] = newValue
        }
    }
    
    return nil
}
```

#### pic_connector Exporter

```go
// File: internal/connector/picconnector/exporter.go

package picconnector

import (
    "context"
    
    "go.opentelemetry.io/collector/component"
    "go.opentelemetry.io/collector/exporter"
    "go.opentelemetry.io/collector/pdata/pmetric"
    
    "github.com/yourorg/sa-omf/internal/extension/piccontrolext"
    "github.com/yourorg/sa-omf/internal/interfaces"
)

type picConnectorExporter struct {
    picControl piccontrolext.PicControl
}

func (e *picConnectorExporter) Start(ctx context.Context, host component.Host) error {
    // Retrieve pic_control extension
    extensions := host.GetExtensions()
    for id, ext := range extensions {
        if id.String() == "pic_control" {
            if pc, ok := ext.(piccontrolext.PicControl); ok {
                e.picControl = pc
                return nil
            }
        }
    }
    return fmt.Errorf("pic_control extension not found")
}

func (e *picConnectorExporter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
    if e.picControl == nil {
        return fmt.Errorf("pic_control not initialized")
    }
    
    // Extract ConfigPatch objects from metrics
    patches := extractConfigPatches(md)
    
    // Submit each patch to pic_control
    for _, patch := range patches {
        err := e.picControl.SubmitConfigPatch(ctx, patch)
        if err != nil {
            // Log error but continue with other patches
            e.logger.Error("Failed to submit ConfigPatch", 
                          zap.String("patch_id", patch.PatchID),
                          zap.Error(err))
        }
    }
    
    return nil
}

// extractConfigPatches extracts ConfigPatch objects from OTLP metrics
func extractConfigPatches(md pmetric.Metrics) []interfaces.ConfigPatch {
    var patches []interfaces.ConfigPatch
    
    // Iterate through metrics looking for aemf_ctrl_proposed_patch
    for i := 0; i < md.ResourceMetrics().Len(); i++ {
        rm := md.ResourceMetrics().At(i)
        for j := 0; j < rm.ScopeMetrics().Len(); j++ {
            sm := rm.ScopeMetrics().At(j)
            for k := 0; k < sm.Metrics().Len(); k++ {
                metric := sm.Metrics().At(k)
                
                if metric.Name() != "aemf_ctrl_proposed_patch" {
                    continue
                }
                
                // Handle different metric types appropriately
                switch metric.Type() {
                case pmetric.MetricTypeGauge:
                    for l := 0; l < metric.Gauge().DataPoints().Len(); l++ {
                        dp := metric.Gauge().DataPoints().At(l)
                        patch := configPatchFromDataPoint(dp)
                        if patch != nil {
                            patches = append(patches, *patch)
                        }
                    }
                }
            }
        }
    }
    
    return patches
}

// configPatchFromDataPoint creates a ConfigPatch from metric data point attributes
func configPatchFromDataPoint(dp pmetric.NumberDataPoint) *interfaces.ConfigPatch {
    // Extract attributes to build ConfigPatch
    // Implementation details...
}
```

## Testing Strategy

A comprehensive testing strategy is essential for a complex system like SA-OMF:

### Unit Testing

1. **Component-Level Tests**:
   - Interface implementations
   - PID controller logic
   - Utility algorithms (HLL, Space-Saving, etc.)
   - Policy validation

2. **Mock-Based Integration Tests**:
   - pic_control with mocked UpdateableProcessor
   - pid_decider with mocked metrics
   - End-to-end control flow with mocks

### Integration Testing

1. **Processor Chain Tests**:
   - Verify metrics flow through processor chain
   - Test correct application of transformations
   - Validate self-metrics emission

2. **Control Loop Tests**:
   - Inject KPI deviations and verify patches
   - Test feedback mechanisms
   - Verify end-to-end adaptation

### Performance Testing

1. **Component Benchmarks**:
   - Individual processor performance
   - Control loop latency
   - Resource utilization profiling

2. **Scalability Tests**:
   - High cardinality handling
   - CPU/memory scaling with load
   - Recovery from resource exhaustion

### Chaos Testing

1. **Failure Injection**:
   - Process restart recovery
   - Network partition handling
   - Resource constraints

2. **Edge Cases**:
   - Configuration oscillation
   - Conflicting PID controllers
   - Extreme cardinality spikes

## Deployment and Operations

### Kubernetes Deployment

1. **DaemonSet Configuration**:
   ```yaml
   # File: deploy/kubernetes/daemonset.yaml
   apiVersion: apps/v1
   kind: DaemonSet
   metadata:
     name: sa-omf-collector
     namespace: monitoring
   spec:
     selector:
       matchLabels:
         app: sa-omf-collector
     template:
       metadata:
         labels:
           app: sa-omf-collector
       spec:
         containers:
         - name: sa-omf-collector
           image: yourorg/sa-omf-otelcol:v1.0.0
           resources:
             limits:
               cpu: 350m
               memory: 350Mi
             requests:
               cpu: 250m
               memory: 256Mi
           volumeMounts:
           - name: config
             mountPath: /etc/sa-omf/config.yaml
             subPath: config.yaml
           - name: policy
             mountPath: /etc/sa-omf/policy.yaml
             subPath: policy.yaml
           ports:
           - containerPort: 8888  # metrics
           - containerPort: 13133 # health
           livenessProbe:
             httpGet:
               path: /health
               port: 13133
             initialDelaySeconds: 15
             periodSeconds: 30
           readinessProbe:
             httpGet:
               path: /health
               port: 13133
             initialDelaySeconds: 5
             periodSeconds: 10
         volumes:
         - name: config
           configMap:
             name: sa-omf-config
         - name: policy
           configMap:
             name: sa-omf-policy
   ```

2. **ConfigMap for policy.yaml**:
   ```yaml
   # File: deploy/kubernetes/configmap-policy.yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: sa-omf-policy
     namespace: monitoring
   data:
     policy.yaml: |
       global_settings:
         autonomy_level: shadow  # Start in shadow mode
         collector_cpu_safety_limit_mcores: 300
         collector_rss_safety_limit_mib: 300
       
       processors_config:
         priority_tagger:
           enabled: true
           # Additional configuration...
         
         adaptive_topk:
           enabled: true
           k_value: 30
           k_min: 10
           k_max: 60
           # Additional configuration...
         
         # Other processors (initially disabled)
         process_context_learner:
           enabled: false
         # ...
       
       pid_decider_config:
         controllers:
         - name: coverage_controller
           enabled: true
           kpi_metric_name: aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m
           kpi_target_value: 0.90
           kp: 30
           ki: 5
           kd: 0
           integral_windup_limit: 60
           hysteresis_percent: 0.03
           output_config_patches:
           - target_processor_name: adaptive_topk
             parameter_path: k_value
             change_scale_factor: -20.0
             min_value: 10
             max_value: 60
         
         # Additional controllers (initially disabled)
         # ...
       
       pic_control_config:
         policy_file_path: /etc/sa-omf/policy.yaml
         max_patches_per_minute: 3
         patch_cooldown_seconds: 10
         # Safe mode configurations...
         
       service:
         extensions: [pic_control, health_check]
         pipelines:
           metrics:
             receivers: [hostmetrics]
             processors: [priority_tagger, adaptive_topk]
             exporters: [prometheusremotewrite]
           control:
             receivers: [prometheus/self]
             processors: [self_metrics_aggregator, pid_decider]
             exporters: [pic_connector]
   ```

### Operational Considerations

1. **Observability**: All components emit detailed metrics under the aemf_* prefix, accessible via /metrics endpoint for scraping by Prometheus.

2. **Resource Requirements**: The collector has specified resource limits, with safety thresholds below these limits to allow for self-regulation.

3. **Deployment Strategy**: Use shadow mode initially, then advisory mode, and finally active mode once the system proves stable.

4. **Update Strategy**: Rolling updates with careful validation of policy changes. OpAMP provides fleet-wide management capability.

5. **Backup and Recovery**: The collector can function with a default safe configuration if policy.yaml becomes corrupted.

## Appendix: Code Templates

### Processor Factory Template

```go
// Template for creating a new processor factory

package yourprocessor

import (
    "context"
    "go.opentelemetry.io/collector/component"
    "go.opentelemetry.io/collector/processor"
)

const (
    typeStr = "your_processor"
)

// NewFactory creates a factory for your processor
func NewFactory() processor.Factory {
    return processor.NewFactory(
        typeStr,
        createDefaultConfig,
        processor.WithMetrics(createMetricsProcessor, component.StabilityLevelDevelopment),
    )
}

func createDefaultConfig() component.Config {
    return &Config{
        // Default configuration
    }
}

func createMetricsProcessor(
    ctx context.Context,
    set processor.CreateSettings,
    cfg component.Config,
    nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
    pCfg := cfg.(*Config)
    return newProcessor(pCfg, set, nextConsumer)
}
```

### UpdateableProcessor Implementation Template

```go
// Template for implementing the UpdateableProcessor interface

package yourprocessor

import (
    "context"
    "sync"
    
    "github.com/yourorg/sa-omf/internal/interfaces"
)

// Ensure your processor implements UpdateableProcessor
var _ interfaces.UpdateableProcessor = (*yourProcessor)(nil)

// OnConfigPatch implements UpdateableProcessor
func (p *yourProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
    if !p.isEnabled {
        return fmt.Errorf("processor is disabled")
    }
    
    // Lock for thread safety
    p.lock.Lock()
    defer p.lock.Unlock()
    
    // Handle specific parameters
    switch patch.ParameterPath {
    case "param1":
        // Type assertion
        newValue, ok := patch.NewValue.(int)
        if !ok {
            return fmt.Errorf("invalid type for param1: %T", patch.NewValue)
        }
        
        // Validation
        if newValue < p.minValue || newValue > p.maxValue {
            return fmt.Errorf("param1 out of range [%d, %d]: %d", 
                             p.minValue, p.maxValue, newValue)
        }
        
        // Apply change
        p.config.Param1 = newValue
        
        // Update internal state if needed
        p.updateInternalState()
        
        // Emit metric
        p.metrics.ReconfigurationTotal.Add(ctx, 1)
        p.metrics.CurrentParam1.Record(ctx, int64(newValue))
        
        return nil
        
    // Handle other parameters...
        
    default:
        return fmt.Errorf("unknown parameter: %s", patch.ParameterPath)
    }
}

// GetConfigStatus implements UpdateableProcessor
func (p *yourProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
    p.lock.RLock()
    defer p.lock.RUnlock()
    
    return interfaces.ConfigStatus{
        Parameters: map[string]interface{}{
            "param1": p.config.Param1,
            "param2": p.config.Param2,
            // Other parameters...
        },
        Enabled: p.isEnabled,
    }, nil
}
```

This implementation plan provides a comprehensive roadmap for bringing the SA-OMF v7.0 architecture to life. By following the phase-based approach, focusing on core functionality first, and incrementally adding advanced capabilities, development teams can deliver a robust, production-ready system while mitigating risks and providing early value.
