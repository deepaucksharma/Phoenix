# Phoenix Monorepo Structure

## Overview
Phoenix is organized as a monorepo with clear sub-projects for better code organization, dependency management, and independent versioning.

## Directory Structure

```
phoenix/
├── packages/               # Shared libraries and utilities
│   ├── contracts/         # API contracts, protobuf definitions, schemas
│   ├── common/            # Shared utilities and helpers
│   └── config/            # Shared configuration utilities
│
├── services/              # Microservices
│   ├── collector/         # Core OTEL collector service
│   ├── control-plane/     # Control and optimization services
│   │   ├── observer/      # Metrics observer
│   │   └── actuator/      # Control actuator
│   ├── generators/        # Load generation services
│   │   ├── synthetic/     # Synthetic metrics generator
│   │   └── complex/       # Complex metrics generator
│   └── validator/         # Validation and benchmarking service
│
├── infrastructure/        # Infrastructure and deployment
│   ├── docker/           # Docker configurations
│   ├── kubernetes/       # K8s manifests (future)
│   └── terraform/        # Infrastructure as code (future)
│
├── monitoring/           # Monitoring stack configurations
│   ├── prometheus/       # Prometheus configs and rules
│   ├── grafana/         # Grafana dashboards and provisioning
│   └── alerts/          # Alert definitions
│
├── tools/               # Development and build tools
│   ├── scripts/         # Utility scripts
│   ├── cli/             # CLI tools
│   └── benchmarks/      # Performance benchmarks
│
├── docs/                # Documentation
│   ├── architecture/    # Architecture documentation
│   ├── api/            # API documentation
│   ├── guides/         # User and developer guides
│   └── rfcs/           # Design proposals
│
├── config/             # Root configuration files
│   ├── environments/   # Environment-specific configs
│   └── defaults/       # Default configurations
│
└── tests/              # Integration and E2E tests
    ├── integration/    # Integration test suites
    ├── e2e/           # End-to-end tests
    └── load/          # Load testing scenarios
```

## Sub-Project Organization

### 1. Packages (Shared Libraries)
```
packages/
├── contracts/
│   ├── proto/          # Protocol buffer definitions
│   ├── openapi/        # OpenAPI specifications
│   ├── schemas/        # JSON schemas
│   └── package.json    # Package metadata
│
├── common/
│   ├── src/
│   │   ├── logger/     # Shared logging
│   │   ├── metrics/    # Metrics utilities
│   │   └── errors/     # Error handling
│   ├── tests/
│   └── package.json
│
└── config/
    ├── src/
    │   ├── loader/     # Config loading utilities
    │   └── validator/  # Config validation
    └── package.json
```

### 2. Services (Microservices)
Each service follows a standard structure:
```
services/<service-name>/
├── cmd/                # Entry points
├── internal/           # Private packages
├── api/               # API definitions
├── config/            # Service configs
├── tests/             # Unit tests
├── Dockerfile         # Container definition
├── Makefile          # Build tasks
└── README.md         # Service documentation
```

### 3. Infrastructure
```
infrastructure/
├── docker/
│   ├── compose/
│   │   ├── base.yaml
│   │   ├── dev.yaml
│   │   └── prod.yaml
│   └── images/        # Base images
│
├── kubernetes/
│   ├── base/          # Base manifests
│   ├── overlays/      # Environment overlays
│   └── charts/        # Helm charts
│
└── terraform/
    ├── modules/       # Reusable modules
    └── environments/  # Environment configs
```

## Monorepo Management

### Workspace Configuration
Root `package.json` for workspace management:
```json
{
  "name": "phoenix",
  "private": true,
  "workspaces": [
    "packages/*",
    "services/*",
    "tools/*"
  ],
  "scripts": {
    "build": "turbo run build",
    "test": "turbo run test",
    "lint": "turbo run lint"
  }
}
```

### Dependency Management
- Shared dependencies at root level
- Service-specific dependencies in service directories
- Version consistency enforced by tooling

### Build System
- Turborepo for efficient monorepo builds
- Bazel for polyglot builds (future)
- Make for simple tasks

## Benefits

1. **Code Sharing**: Easy sharing of common code via packages
2. **Atomic Changes**: Related changes across services in single commit
3. **Consistent Tooling**: Shared build, test, and lint configurations
4. **Clear Boundaries**: Well-defined service and package boundaries
5. **Independent Deployment**: Services can still be deployed independently