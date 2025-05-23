# Phoenix-vNext Modular Architecture

## Overview
This document outlines the refactored micro-module architecture for Phoenix-vNext, transforming the monolithic structure into well-defined, loosely coupled modules with clear responsibilities and interfaces.

## Module Structure

### 1. phoenix-core (Metrics Collection Core)
**Purpose**: Core OpenTelemetry collector functionality with multi-pipeline processing

**Components**:
- OTLP receiver configuration
- Pipeline processors (cardinality reduction, aggregation, TopK)
- Prometheus exporters
- Dynamic configuration reader

**Interfaces**:
- Input: OTLP metrics on port 4318
- Output: Prometheus metrics endpoints (ports 8888-8891)
- Config: Reads optimization_mode.yaml for dynamic control

**Directory Structure**:
```
modules/phoenix-core/
├── configs/
│   ├── pipelines/
│   │   ├── full_fidelity.yaml
│   │   ├── optimized.yaml
│   │   └── experimental_topk.yaml
│   └── processors/
├── docker/
│   └── Dockerfile
├── api/
│   └── metrics.proto
└── README.md
```

### 2. phoenix-control (Control Plane)
**Purpose**: Adaptive control system for dynamic optimization

**Components**:
- Observer collector (KPI monitoring)
- Control actuator (PID-like controller)
- Control signal generator

**Interfaces**:
- Input: Scrapes phoenix-core metrics
- Output: Control signals via optimization_mode.yaml
- API: REST endpoint for manual control override

**Directory Structure**:
```
modules/phoenix-control/
├── observer/
│   ├── config/
│   └── Dockerfile
├── actuator/
│   ├── src/
│   ├── scripts/
│   └── Dockerfile
├── api/
│   └── control.proto
└── README.md
```

### 3. phoenix-generators (Load Generation Suite)
**Purpose**: Synthetic workload generation for testing and validation

**Components**:
- Process metrics generator (Go)
- Complex metrics generator (Bash/Python)
- Load profiles and scenarios

**Interfaces**:
- Output: OTLP metrics to phoenix-core
- Config: Load profiles via YAML/JSON
- API: REST endpoint for dynamic load control

**Directory Structure**:
```
modules/phoenix-generators/
├── synthetic/
│   ├── cmd/
│   ├── internal/
│   └── Dockerfile
├── complex/
│   ├── scripts/
│   └── Dockerfile
├── profiles/
│   └── scenarios/
└── README.md
```

### 4. phoenix-validation (Benchmarking & Validation)
**Purpose**: Performance validation and cost analysis

**Components**:
- Benchmark controller
- Performance analyzers
- Cost calculators

**Interfaces**:
- Input: Prometheus queries
- Output: Benchmark results (JSON/SQLite)
- API: REST endpoint for benchmark triggers

**Directory Structure**:
```
modules/phoenix-validation/
├── benchmark/
│   ├── cmd/
│   ├── internal/
│   └── Dockerfile
├── analyzers/
├── storage/
│   └── schemas/
└── README.md
```

### 5. phoenix-monitoring (Monitoring Stack)
**Purpose**: Metrics storage and visualization

**Components**:
- Prometheus configuration
- Grafana dashboards
- Alert rules

**Interfaces**:
- Input: Prometheus scrape endpoints
- Output: Grafana UI, Prometheus query API
- Config: Dashboard and alert definitions

**Directory Structure**:
```
modules/phoenix-monitoring/
├── prometheus/
│   ├── config/
│   └── rules/
├── grafana/
│   ├── dashboards/
│   └── provisioning/
├── alerts/
└── README.md
```

### 6. phoenix-contracts (Shared Contracts & APIs)
**Purpose**: Common interfaces and data contracts

**Components**:
- Protocol buffer definitions
- JSON schemas
- API specifications
- Shared utilities

**Directory Structure**:
```
modules/phoenix-contracts/
├── proto/
│   ├── metrics.proto
│   ├── control.proto
│   └── benchmark.proto
├── schemas/
│   ├── control_signal.json
│   └── benchmark_result.json
├── openapi/
└── README.md
```

## Inter-Module Communication

### Communication Patterns
1. **Metrics Flow**: generators → core → monitoring
2. **Control Loop**: core → control → core (via config file)
3. **Validation**: validation → monitoring → validation

### API Contracts

#### Metrics Pipeline API
```yaml
endpoints:
  ingestion:
    protocol: OTLP/gRPC
    port: 4318
  export:
    full_fidelity: http://localhost:8888/metrics
    optimized: http://localhost:8889/metrics
    experimental: http://localhost:8890/metrics
    observatory: http://localhost:8891/metrics
```

#### Control Plane API
```yaml
endpoints:
  observer:
    metrics: http://localhost:9888/metrics
  actuator:
    control: http://localhost:8080/api/v1/control
    status: http://localhost:8080/api/v1/status
```

## Deployment Architecture

### Container Orchestration
Each module will have its own Docker container(s) with well-defined:
- Resource limits
- Health checks
- Network policies
- Volume mounts

### Configuration Management
- Environment-specific configs in `environments/`
- Secrets management via Docker secrets or external vault
- Feature flags for gradual rollout

## Migration Plan

### Phase 1: Directory Restructuring
1. Create module directories
2. Move existing code to appropriate modules
3. Update import paths

### Phase 2: Interface Definition
1. Define protocol buffers
2. Create OpenAPI specs
3. Implement API gateways

### Phase 3: Decoupling
1. Replace direct dependencies with API calls
2. Implement service discovery
3. Add circuit breakers

### Phase 4: Independent Deployment
1. Create module-specific CI/CD pipelines
2. Implement versioning strategy
3. Enable independent scaling

## Benefits

1. **Modularity**: Clear separation of concerns
2. **Scalability**: Independent scaling of components
3. **Maintainability**: Easier to understand and modify
4. **Testability**: Isolated unit and integration testing
5. **Flexibility**: Easy to replace or upgrade individual modules
6. **Reusability**: Modules can be used in other projects