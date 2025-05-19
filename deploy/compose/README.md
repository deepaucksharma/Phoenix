# Docker Compose Configuration

This directory contains Docker Compose configurations for different deployment scenarios.

## Available Configurations

1. **Root level docker-compose.yml**
   - Main development configuration with multiple services:
     - `dev`: Development environment with source code mounted
     - `collector-default`: Collector with default configuration
     - `collector-development`: Collector with development configuration
     - `prometheus`: Prometheus for metrics collection
     - `grafana`: Grafana for dashboards
   - Use for local development with monitoring: `docker-compose up -d`

2. **deploy/compose/docker-compose.yml**
   - Simplified configuration for basic collector execution
   - Configurable via environment variables:
     - `TAG`: Docker image tag (default: latest)
     - `VERSION`, `COMMIT`, `BUILD_DATE`: Build arguments
     - `PORT_HTTP`: HTTP port (default: 8888)
     - `PORT_HEALTH`: Health check port (default: 13133)
     - `CONFIG_ENV`: Configuration environment (default: default)
   - Use for simple deployment: `docker-compose -f deploy/compose/docker-compose.yml up -d`

3. **deploy/compose/bare/docker-compose.yaml**
   - Minimal configuration with just the collector
   - Use when you need only the collector: `docker-compose -f deploy/compose/bare/docker-compose.yaml up -d`

4. **deploy/compose/prometheus/docker-compose.yaml**
   - Collector with Prometheus
   - Use for collecting metrics: `docker-compose -f deploy/compose/prometheus/docker-compose.yaml up -d`

5. **deploy/compose/full/docker-compose.yaml**
   - Complete setup with collector, Prometheus, and Grafana
   - Use for full monitoring stack: `docker-compose -f deploy/compose/full/docker-compose.yaml up -d`

## Usage Instructions

Choose the appropriate docker-compose file based on your needs:

```bash
# Basic collector only
docker-compose -f deploy/compose/bare/docker-compose.yaml up -d

# Collector with Prometheus
docker-compose -f deploy/compose/prometheus/docker-compose.yaml up -d

# Full stack with collector, Prometheus, and Grafana
docker-compose -f deploy/compose/full/docker-compose.yaml up -d

# Development environment with all services
docker-compose up -d
```

## Environment Variables

You can customize the deployment by setting environment variables:

```bash
# Example: Use production config with a specific tag
CONFIG_ENV=production TAG=v1.0.0 docker-compose -f deploy/compose/docker-compose.yml up -d
```