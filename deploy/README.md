# SA-OMF Deployment Resources

This directory contains all deployment-related resources for the Self-Aware OpenTelemetry Metrics Fabric.

## Directory Structure

- **docker/**: Docker-related files
  - **Dockerfile**: Main Dockerfile for building the collector image
- **kubernetes/**: Kubernetes deployment resources
  - **prometheus-operator-resources.yaml**: Resources for deploying with Prometheus Operator
- **compose/**: Docker Compose configurations
  - **docker-compose.yml**: Base Docker Compose configuration
  - **bare/**: Minimalist deployment with just the collector
  - **prometheus/**: Deployment with Prometheus for monitoring
  - **full/**: Complete deployment with all monitoring components

## Deployment Options

### Docker

To build and run the collector using Docker:

```bash
# Build the image
make docker

# Or manually
docker build -t sa-omf-otelcol:latest -f deploy/docker/Dockerfile .

# Run the container
docker run -p 8888:8888 -v $PWD/configs/default:/etc/sa-omf sa-omf-otelcol:latest --config=/etc/sa-omf/config.yaml
```

### Docker Compose

For a more complete environment with monitoring:

```bash
# Basic deployment
cd deploy/compose/bare && docker-compose up -d

# With Prometheus
cd deploy/compose/prometheus && docker-compose up -d

# Full stack with Grafana
cd deploy/compose/full && docker-compose up -d
```

### Kubernetes

Kubernetes deployment is supported with the following resources:

```bash
# Apply the Prometheus Operator resources
kubectl apply -f deploy/kubernetes/prometheus-operator-resources.yaml

# Deploy using Helm (if available)
helm install sa-omf ./deploy/kubernetes/helm
```

## Configuration

All deployments use the configurations from the `configs` directory. See the [configuration README](../configs/README.md) for more information.
