# Phoenix Platform Architecture Overview

## Introduction

Phoenix is an automated observability platform that optimizes process metrics collection through intelligent OpenTelemetry pipelines. The platform reduces telemetry costs by 50-80% while maintaining 100% visibility for critical processes.

## Architecture Diagram

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Web Dashboard  │────▶│   API Gateway   │────▶│ Experiment API  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                          │
                                ┌─────────────────────────┴───────────────┐
                                │                                         │
                        ┌───────▼────────┐                      ┌────────▼────────┐
                        │  Config Gen    │                      │ Experiment Ctrl │
                        └───────┬────────┘                      └────────┬────────┘
                                │                                         │
                        ┌───────▼────────┐                      ┌────────▼────────┐
                        │   Git Repo     │◀─────────────────────│  Kubernetes API │
                        └───────┬────────┘                      └─────────────────┘
                                │                                         │
                        ┌───────▼────────┐     ┌─────────────────────────┘
                        │    ArgoCD      │     │
                        └───────┬────────┘     │
                                │              │
                        ┌───────▼──────────────▼──┐
                        │   OTel Collectors       │
                        │  (Baseline & Candidate) │
                        └───────┬─────────────────┘
                                │
                        ┌───────▼────────┐
                        │  New Relic &   │
                        │  Prometheus    │
                        └────────────────┘
```

## Core Components

### 1. Control Plane

#### API Gateway
- **Technology**: Go with Chi router and gRPC-gateway
- **Purpose**: External REST/WebSocket interface for dashboard
- **Key Features**:
  - JWT authentication
  - WebSocket support for real-time updates
  - Request validation and rate limiting

#### Experiment API
- **Technology**: Go with gRPC
- **Purpose**: Core business logic for experiment management
- **Key Features**:
  - Experiment lifecycle management
  - Pipeline validation
  - Integration with Config Generator

#### Configuration Generator
- **Technology**: Go
- **Purpose**: Transforms visual pipeline designs into OTel configurations
- **Key Features**:
  - YAML generation for OTel collectors
  - Kubernetes manifest generation
  - Git integration for version control

#### Experiment Controller
- **Technology**: Kubernetes controller (Go)
- **Purpose**: Manages experiment deployment and lifecycle
- **Key Features**:
  - Watches PhoenixExperiment CRDs
  - Coordinates with Pipeline Operator
  - Handles experiment state transitions

### 2. Data Plane

#### OTel Collectors
- **Technology**: OpenTelemetry Collector (contrib distribution)
- **Purpose**: Process metrics collection and optimization
- **Deployment**: Kubernetes DaemonSet
- **Key Features**:
  - Dual deployment for A/B testing
  - Multiple processor chains
  - Dual export (Prometheus + New Relic)

#### Pipeline Templates
Pre-validated configurations:
- `process-baseline-v1`: No optimization (control)
- `process-priority-filter-v1`: Priority-based filtering
- `process-topk-v1`: Top CPU/memory consumers only
- `process-aggregated-v1`: Aggregate common applications

### 3. User Interface

#### Web Dashboard
- **Technology**: React 18 with TypeScript
- **Purpose**: Visual pipeline builder and experiment management
- **Key Features**:
  - Drag-and-drop pipeline builder (React Flow)
  - Real-time experiment monitoring
  - Cost analysis dashboards
  - Pipeline template library

### 4. Operators

#### Pipeline Operator
- **Purpose**: Manages OTel collector deployments
- **CRD**: PhoenixProcessPipeline
- **Key Features**:
  - DaemonSet management
  - ConfigMap generation
  - Rolling updates

#### LoadSim Operator
- **Purpose**: Manages process simulation jobs
- **CRD**: LoadSimulationJob
- **Key Features**:
  - Job scheduling
  - Load profile management
  - Cleanup automation

### 5. Observability Stack

#### Prometheus
- **Purpose**: Metrics storage and querying
- **Key Metrics**:
  - Collector performance metrics
  - Pipeline cardinality metrics
  - Experiment comparison data

#### Grafana
- **Purpose**: Visualization and dashboards
- **Key Dashboards**:
  - Pipeline Performance
  - A/B Experiment Comparison
  - Cost Analysis
  - System Health

## Data Flow

### 1. Experiment Creation
1. User designs pipeline in dashboard
2. Dashboard sends request to API Gateway
3. API validates and stores in PostgreSQL
4. Config Generator creates OTel configs
5. Git PR created with configurations
6. ArgoCD deploys to Kubernetes

### 2. Metrics Collection
1. Host processes generate metrics
2. OTel Collector scrapes via hostmetrics receiver
3. Processors optimize based on pipeline config
4. Metrics exported to:
   - Prometheus (local analysis)
   - New Relic (production monitoring)

### 3. A/B Testing
1. Two collectors deployed on same host
2. Both process identical input metrics
3. Different optimization strategies applied
4. Results compared in real-time
5. Winner promoted after analysis

## Security Architecture

### Authentication & Authorization
- JWT-based authentication for API
- RBAC in Kubernetes
- Service account isolation

### Network Security
- TLS for all external communication
- mTLS between internal services
- Network policies for pod isolation

### Secret Management
- External Secrets Operator integration
- Kubernetes secrets for sensitive data
- No hardcoded credentials

## Scalability Considerations

### Performance Targets
- 100+ concurrent experiments
- 1000+ nodes per experiment
- 500+ processes per node
- 3.5M+ unique time series

### Scaling Strategies
- Horizontal pod autoscaling for API
- Prometheus federation for metrics
- PostgreSQL read replicas
- CDN for dashboard assets

## Deployment Architecture

### Kubernetes Resources
- **Namespaces**: `phoenix-system`, `phoenix-experiments`
- **Storage**: PVCs for Prometheus, PostgreSQL
- **Ingress**: NGINX with TLS termination
- **Service Mesh**: Optional Istio integration

### GitOps Workflow
1. All configs in Git
2. ArgoCD syncs changes
3. Automated rollback on failure
4. Audit trail via Git history

## Monitoring & Alerting

### Key Metrics
- API latency and error rates
- Collector resource usage
- Pipeline cardinality reduction
- Experiment success rate

### Alerting Rules
- Collector failures
- High cardinality detection
- API degradation
- Storage capacity warnings