# Phoenix Monorepo Modularity Review

## Executive Summary

This document reviews the current Phoenix monorepo structure against architectural best practices and provides actionable recommendations for improving modularity, code reuse, and maintainability while supporting the 3-pipeline cardinality optimization architecture.

## Current State Analysis

### Directory Structure Overview

```
phoenix/
├── apps/                    # Go microservices (inconsistent usage)
├── services/               # Mixed services (Go, Node.js, scripts)
├── packages/               # Underutilized shared packages
├── configs/                # Technology-grouped configurations
├── infrastructure/         # Cloud deployment resources
├── scripts/                # Operational utilities
└── tools/                  # Development utilities
```

### Key Findings

#### 1. **Inconsistent Service Organization**

**Current Issues:**
- Go services arbitrarily split between `apps/` and `services/`
- No clear criteria for placement (e.g., `control-actuator-go` in `apps/`, `benchmark` in `services/`)
- Mixed implementation languages within same directories

**Impact:**
- Confusion for developers
- Inconsistent build processes
- Difficult to establish conventions

#### 2. **Limited Code Reuse**

**Current Issues:**
- Each Go service has independent `go.mod` with duplicated dependencies
- Common patterns reimplemented in each service:
  ```go
  // Repeated in every service:
  func getEnv(key, defaultValue string) string { ... }
  func NewPrometheusClient() { ... }
  func handleHealth(w http.ResponseWriter, r *http.Request) { ... }
  ```
- No shared types for cross-service communication

**Impact:**
- Increased maintenance burden
- Inconsistent implementations
- Higher risk of bugs

#### 3. **Tight Coupling Through Files**

**Current Issues:**
- Control signal via shared YAML file creates implicit coupling
- Services directly mount configuration volumes
- No versioning or validation of shared data structures

**Example:**
```yaml
# configs/control/optimization_mode.yaml
# Written by control-actuator, read by otel-collector
# No schema validation or version compatibility checks
optimization_mode: balanced
last_updated: 2024-01-01T00:00:00Z
```

**Impact:**
- Deployment dependencies
- Version compatibility issues
- Difficult to test in isolation

#### 4. **Underutilized Monorepo Tooling**

**Current Issues:**
- Turborepo configured but benefits limited due to lack of shared packages
- npm workspaces defined but no JavaScript/TypeScript packages to share
- No dependency graph between services

**Turbo.json Analysis:**
```json
{
  "pipeline": {
    "build": {
      "dependsOn": ["^build"],  // Only external deps
      "outputs": ["dist/**"]
    }
    // No service-to-service dependencies defined
  }
}
```

## Architectural Recommendations

### 1. **Reorganize Service Structure**

**Proposed Structure:**
```
phoenix/
├── apps/                          # Configuration-driven applications
│   ├── otel-collector-main/      # Main 3-pipeline collector
│   ├── otel-collector-observer/  # Monitoring collector
│   ├── prometheus/               # Metrics storage
│   └── grafana/                  # Dashboards
│
├── services/                      # Business logic services (all Go)
│   ├── control-actuator/         # PID controller
│   ├── anomaly-detector/         # Anomaly detection
│   ├── benchmark-controller/     # Performance testing
│   └── metric-analyzer/          # Analytics service
│
├── packages/                      # Shared libraries
│   ├── go-common/                # Go shared library
│   │   ├── observability/        # Metrics, traces, logs
│   │   ├── config/              # Configuration management
│   │   ├── health/              # Health check standards
│   │   ├── control/             # Control signal types
│   │   └── testing/             # Test utilities
│   │
│   ├── contracts/                # Service contracts
│   │   ├── api/                 # OpenAPI specs
│   │   ├── proto/               # gRPC definitions
│   │   └── events/              # Event schemas
│   │
│   └── pipeline-configs/         # OTEL pipeline definitions
│       ├── processors/          # Shared processor configs
│       ├── exporters/           # Shared exporter configs
│       └── pipelines/           # Pipeline templates
```

### 2. **Implement Shared Go Package**

**Create `packages/go-common/go.mod`:**
```go
module github.com/phoenix-vnext/phoenix/packages/go-common

go 1.21

require (
    github.com/prometheus/client_golang v1.17.0
    github.com/prometheus/common v0.45.0
    go.opentelemetry.io/otel v1.21.0
    gopkg.in/yaml.v3 v3.0.1
)
```

**Shared Types (`packages/go-common/control/types.go`):**
```go
package control

import "time"

type OptimizationMode string

const (
    ModeConservative OptimizationMode = "conservative"
    ModeBalanced     OptimizationMode = "balanced"
    ModeAggressive   OptimizationMode = "aggressive"
)

type ControlSignal struct {
    Mode           OptimizationMode `json:"mode" yaml:"mode"`
    Version        string          `json:"version" yaml:"version"`
    Timestamp      time.Time       `json:"timestamp" yaml:"timestamp"`
    CorrelationID  string          `json:"correlation_id" yaml:"correlation_id"`
    Source         string          `json:"source" yaml:"source"`
    Reason         string          `json:"reason,omitempty" yaml:"reason,omitempty"`
}

type CardinalityMetrics struct {
    Pipeline      string    `json:"pipeline"`
    TimeSeriesCount float64 `json:"time_series_count"`
    Timestamp     time.Time `json:"timestamp"`
}
```

**Shared Observability (`packages/go-common/observability/metrics.go`):**
```go
package observability

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsServer struct {
    registry *prometheus.Registry
    port     string
}

func NewMetricsServer(serviceName string, port string) *MetricsServer {
    registry := prometheus.NewRegistry()
    
    // Register standard metrics
    registry.MustRegister(
        prometheus.NewBuildInfoCollector(),
        prometheus.NewGoCollector(),
    )
    
    return &MetricsServer{
        registry: registry,
        port:     port,
    }
}

// Standardized metrics for all services
func (m *MetricsServer) RegisterServiceMetrics(serviceName string) {
    // Request duration
    requestDuration := prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "phoenix_service_request_duration_seconds",
            Help: "Request duration in seconds",
            ConstLabels: prometheus.Labels{"service": serviceName},
        },
        []string{"method", "endpoint", "status"},
    )
    m.registry.MustRegister(requestDuration)
}
```

### 3. **Implement Service Mesh Pattern**

**Service Discovery via Environment:**
```go
// packages/go-common/discovery/client.go
package discovery

import (
    "fmt"
    "os"
)

type ServiceEndpoints struct {
    ControlActuator  string
    AnomalyDetector  string
    Observer         string
    Prometheus       string
}

func LoadEndpoints() *ServiceEndpoints {
    return &ServiceEndpoints{
        ControlActuator: getEndpoint("CONTROL_ACTUATOR", "http://control-actuator:8081"),
        AnomalyDetector: getEndpoint("ANOMALY_DETECTOR", "http://anomaly-detector:8082"),
        Observer:        getEndpoint("OBSERVER", "http://otel-observer:9888"),
        Prometheus:      getEndpoint("PROMETHEUS", "http://prometheus:9090"),
    }
}
```

### 4. **Standardize Service Communication**

**Option A: HTTP with Contracts**
```go
// packages/contracts/http/control.go
package http

import "github.com/phoenix-vnext/phoenix/packages/go-common/control"

type UpdateModeRequest struct {
    Mode   control.OptimizationMode `json:"mode"`
    Reason string                   `json:"reason"`
}

type UpdateModeResponse struct {
    Success       bool   `json:"success"`
    PreviousMode  string `json:"previous_mode"`
    CurrentMode   string `json:"current_mode"`
    CorrelationID string `json:"correlation_id"`
}
```

**Option B: gRPC with Proto**
```protobuf
// packages/contracts/proto/control.proto
syntax = "proto3";
package phoenix.control.v1;

service ControlService {
    rpc GetCurrentMode(GetModeRequest) returns (GetModeResponse);
    rpc UpdateMode(UpdateModeRequest) returns (UpdateModeResponse);
    rpc StreamModeChanges(StreamRequest) returns (stream ModeChangeEvent);
}

message UpdateModeRequest {
    enum Mode {
        CONSERVATIVE = 0;
        BALANCED = 1;
        AGGRESSIVE = 2;
    }
    Mode mode = 1;
    string reason = 2;
    string correlation_id = 3;
}
```

### 5. **Improve Build Configuration**

**Enhanced turbo.json:**
```json
{
  "$schema": "https://turbo.build/schema.json",
  "pipeline": {
    // Shared package builds first
    "@phoenix/go-common#build": {
      "outputs": ["bin/**", "pkg/**"],
      "cache": true
    },
    
    // Services depend on common package
    "services/control-actuator#build": {
      "dependsOn": ["@phoenix/go-common#build"],
      "outputs": ["bin/**"],
      "env": ["GOOS", "GOARCH"]
    },
    
    // Integration tests depend on all services
    "test:integration": {
      "dependsOn": [
        "services/control-actuator#build",
        "services/anomaly-detector#build",
        "services/benchmark-controller#build"
      ],
      "cache": false
    },
    
    // Docker builds
    "docker:build": {
      "dependsOn": ["build"],
      "outputs": [],
      "cache": true
    }
  }
}
```

### 6. **Configuration Management**

**Centralized Configuration Schema:**
```yaml
# packages/configs/schemas/control-config.schema.yaml
$schema: http://json-schema.org/draft-07/schema#
type: object
required:
  - optimization_mode
  - version
  - timestamp
properties:
  optimization_mode:
    type: string
    enum: [conservative, balanced, aggressive]
  version:
    type: string
    pattern: "^\\d+\\.\\d+\\.\\d+$"
  timestamp:
    type: string
    format: date-time
```

**Configuration Service:**
```go
// services/config-manager/main.go
package main

import (
    "github.com/phoenix-vnext/phoenix/packages/go-common/config"
)

type ConfigManager struct {
    validator *config.SchemaValidator
    store     config.Store
}

func (cm *ConfigManager) UpdateControlMode(signal control.ControlSignal) error {
    // Validate against schema
    if err := cm.validator.Validate("control-config", signal); err != nil {
        return fmt.Errorf("invalid control signal: %w", err)
    }
    
    // Store with versioning
    return cm.store.Put("control/mode", signal, config.WithVersion(signal.Version))
}
```

## Implementation Roadmap

### Phase 1: Foundation (Week 1-2)
1. Create `packages/go-common` with basic shared types
2. Migrate helper functions to shared package
3. Standardize health check endpoints
4. Set up shared testing utilities

### Phase 2: Service Refactoring (Week 3-4)
1. Update all Go services to use shared package
2. Implement standardized metrics collection
3. Replace file-based control with API calls
4. Add service discovery pattern

### Phase 3: Communication Layer (Week 5-6)
1. Define gRPC/HTTP contracts
2. Implement service-to-service communication
3. Add distributed tracing
4. Implement circuit breakers

### Phase 4: Build Optimization (Week 7-8)
1. Update turbo.json with proper dependencies
2. Implement parallel builds
3. Add integration test pipeline
4. Optimize Docker builds with shared layers

## Expected Benefits

1. **Reduced Code Duplication**: ~40% less code through shared packages
2. **Improved Maintainability**: Single source of truth for common functionality
3. **Better Testing**: Shared test utilities and mocks
4. **Faster Builds**: Proper caching and parallel execution
5. **Easier Onboarding**: Clear structure and conventions
6. **Enhanced Reliability**: Standardized patterns reduce bugs
7. **Scalability**: Services can be deployed independently
8. **Version Compatibility**: Explicit contracts with versioning

## Metrics for Success

- **Code Reuse**: >60% of common functionality in shared packages
- **Build Time**: 50% reduction through proper caching
- **Test Coverage**: >80% across all services
- **Service Isolation**: Zero file-based dependencies
- **API Contracts**: 100% of inter-service communication documented
- **Deployment Independence**: Any service can be deployed without others

## Conclusion

The current Phoenix monorepo structure has served its purpose but needs evolution to support the growing complexity of the 3-pipeline architecture. By implementing these recommendations, the system will become more maintainable, scalable, and aligned with microservices best practices while maintaining the benefits of a monorepo structure.