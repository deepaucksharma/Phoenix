# Phoenix Documentation

## Overview
This directory contains documentation for the Phoenix v3 Cardinality Optimization System.

## Core Documentation

### System Architecture
- **[CLAUDE.md](../CLAUDE.md)** - Comprehensive AI assistant instructions, architecture details, and operational procedures
- **[README.md](../README.md)** - Main project documentation with quick start guide

### Component Documentation
- **[Scripts Documentation](../scripts/consolidated/README.md)** - Consolidated scripts organization and usage
- **[Prometheus Rules](../configs/monitoring/prometheus/rules/README.md)** - Metrics and alerting rules documentation
- **[Go Common Packages](../packages/go-common/README.md)** - Shared Go packages documentation

## Quick Links

### Configuration
- [OpenTelemetry Collectors](../configs/otel/collectors/)
- [Prometheus Configuration](../configs/monitoring/prometheus/)
- [Control System](../configs/control/)

### Scripts
- [Master Script Manager](../scripts/consolidated/phoenix-scripts.sh) - Unified entry point
- [Initialize Environment](../scripts/consolidated/core/initialize-environment.sh)
- [Run Phoenix](../scripts/consolidated/core/run-phoenix.sh)

### Deployment
- [AWS Deployment](../scripts/consolidated/deployment/deploy-aws.sh)
- [Azure Deployment](../scripts/consolidated/deployment/deploy-azure.sh)
- [Docker Compose](../docker-compose.yaml)

## Key System Information

### Service Ports
- Control Actuator: `8081`
- Anomaly Detector: `8082`
- Benchmark Controller: `8083`
- Main Collector Health: `13133`
- Observer Health: `13134`

### Performance Targets
- Signal preservation: >98%
- Cardinality reduction: 15-40%
- Control loop latency: <100ms
- Memory usage: <512MB baseline

## Historical Documentation
Historical reports and analysis documents were previously stored in an `archive/` directory.
That directory has been removed after consolidation, and all critical information has been
incorporated into the main documentation.
