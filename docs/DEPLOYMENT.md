# Phoenix-vNext Deployment and Operations Guide

## Prerequisites

### System Requirements

- **Operating System**: Linux, macOS, or Windows with WSL2
- **Docker**: Docker Desktop 4.0+ with Compose V2
- **Memory**: Minimum 8GB RAM, 16GB recommended
- **CPU**: 4+ cores recommended for optimal performance
- **Storage**: 10GB free space for data volumes and images

### Network Requirements

The following ports must be available:

| Port | Service | Description |
|------|---------|-------------|
| 3000 | Grafana | Dashboard UI |
| 4318 | Main Collector | OTLP/HTTP ingest |
| 8888-8890 | Main Collector | Pipeline Prometheus endpoints |
| 9090 | Prometheus | Metrics storage UI |
| 9888 | Observer | Observer metrics endpoint |
| 13133-13134 | Health Checks | Collector health endpoints |
| 1777-1778 | pprof | Profiling endpoints |
| 55679-55680 | zpages | Collector internal state |

## Quick Start Deployment

### 1. Environment Setup

```bash
# Clone the repository
git clone <repository-url>
cd phoenix-vnext

# Initialize environment and create required files
./scripts/initialize-environment.sh

# Review and customize .env file
nano .env
```

### 2. Configuration Review

Key environment variables to verify:

```bash
# Control thresholds
TARGET_OPTIMIZED_PIPELINE_TS_COUNT=20000
THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=15000
THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=25000

# Resource limits
OTELCOL_MAIN_MEMORY_LIMIT_MIB=1024
OTELCOL_OBSERVER_MEMORY_LIMIT_MIB=256

# Synthetic load configuration
SYNTHETIC_PROCESS_COUNT_PER_HOST=250
SYNTHETIC_HOST_COUNT=3
SYNTHETIC_METRIC_EMIT_INTERVAL_S=15

# Export configuration (optional)
NEW_RELIC_LICENSE_KEY_FULL=""
NEW_RELIC_LICENSE_KEY_OPTIMISED=""
NEW_RELIC_LICENSE_KEY_EXPERIMENTAL=""
```

### 3. Service Startup

```bash
# Start all services
docker-compose up -d

# Verify service health
docker-compose ps

# Check service logs
docker-compose logs -f otelcol-main
docker-compose logs -f otelcol-observer
docker-compose logs -f control-loop-actuator
```

### 4. Access Monitoring

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Health Status**: http://localhost:13133 (main), http://localhost:13134 (observer)

## Production Deployment

### Environment Configuration

Create production-specific environment files:

```bash
# Production environment
cp .env .env.production

# Staging environment  
cp .env .env.staging
```

Key production settings:

```bash
# Resource limits for production
OTELCOL_MAIN_MEMORY_LIMIT_MIB=4096
OTELCOL_MAIN_GOMAXPROCS=4
OTELCOL_OBSERVER_MEMORY_LIMIT_MIB=512

# Production control thresholds
TARGET_OPTIMIZED_PIPELINE_TS_COUNT=50000
THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=40000
THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=60000

# Production synthetic load (reduce or disable)
SYNTHETIC_PROCESS_COUNT_PER_HOST=100
SYNTHETIC_HOST_COUNT=1

# Production monitoring retention
PROMETHEUS_RETENTION_TIME=30d
GRAFANA_ADMIN_PASSWORD=<secure-password>

# Export destinations
NEW_RELIC_OTLP_ENDPOINT=https://otlp.nr-data.net:4318/v1/metrics
NEW_RELIC_LICENSE_KEY_FULL=<production-key>
# ... additional keys for each pipeline
```

### Docker Compose Override

Create `docker-compose.prod.yml`:

```yaml
version: "3.9"

services:
  otelcol-main:
    deploy:
      resources:
        limits: { cpus: '4.0', memory: '4G' }
        reservations: { cpus: '2.0', memory: '2G' }
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"

  otelcol-observer:
    deploy:
      resources:
        limits: { cpus: '1.0', memory: '512M' }
        reservations: { cpus: '0.5', memory: '256M' }

  prometheus:
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=30d'
      - '--storage.tsdb.retention.size=50GB'
      - '--web.enable-lifecycle'
      - '--web.enable-admin-api'
    deploy:
      resources:
        limits: { cpus: '2.0', memory: '4G' }
        reservations: { cpus: '1.0', memory: '2G' }

  # Disable synthetic generators in production
  synthetic-metrics-generator:
    profiles: ["dev"]
    
  stress-ng-cpu-heavy:
    profiles: ["dev"]
    
  stress-ng-io-heavy:
    profiles: ["dev"]
```

Deploy with production configuration:

```bash
docker-compose -f docker-compose.yaml -f docker-compose.prod.yml --env-file .env.production up -d
```

## Kubernetes Deployment

### Namespace and Resources

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: phoenix-vnext
  labels:
    app.kubernetes.io/name: phoenix-vnext
---
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: phoenix-config
  namespace: phoenix-vnext
data:
  main-collector-config.yaml: |
    # Include main.yaml content here
  observer-collector-config.yaml: |
    # Include observer.yaml content here
```

### Main Collector Deployment

```yaml
# main-collector.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otelcol-main
  namespace: phoenix-vnext
spec:
  replicas: 1
  selector:
    matchLabels:
      app: otelcol-main
  template:
    metadata:
      labels:
        app: otelcol-main
    spec:
      containers:
      - name: otelcol
        image: otel/opentelemetry-collector-contrib:0.103.1
        args: ["--config=/etc/otelcol/config.yaml"]
        env:
        - name: GOMAXPROCS
          value: "4"
        - name: GOMEMLIMIT
          value: "4GiB"
        resources:
          requests:
            cpu: 2000m
            memory: 2Gi
          limits:
            cpu: 4000m
            memory: 4Gi
        ports:
        - containerPort: 4318
          name: otlp-http
        - containerPort: 8888
          name: metrics-full
        - containerPort: 8889
          name: metrics-opt
        - containerPort: 8890
          name: metrics-exp
        - containerPort: 13133
          name: health
        volumeMounts:
        - name: config
          mountPath: /etc/otelcol
        - name: control-signals
          mountPath: /etc/otelcol/control
        - name: hostfs-proc
          mountPath: /hostfs/proc
          readOnly: true
        - name: hostfs-sys
          mountPath: /hostfs/sys
          readOnly: true
        livenessProbe:
          httpGet:
            path: /
            port: 13133
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /
            port: 13133
          initialDelaySeconds: 10
          periodSeconds: 10
      volumes:
      - name: config
        configMap:
          name: phoenix-config
          items:
          - key: main-collector-config.yaml
            path: config.yaml
      - name: control-signals
        configMap:
          name: control-signals
      - name: hostfs-proc
        hostPath:
          path: /proc
      - name: hostfs-sys
        hostPath:
          path: /sys
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
```

### Services and Ingress

```yaml
# services.yaml
apiVersion: v1
kind: Service
metadata:
  name: otelcol-main
  namespace: phoenix-vnext
spec:
  selector:
    app: otelcol-main
  ports:
  - name: otlp-http
    port: 4318
    targetPort: 4318
  - name: metrics-full
    port: 8888
    targetPort: 8888
  - name: metrics-opt
    port: 8889
    targetPort: 8889
  - name: metrics-exp
    port: 8890
    targetPort: 8890
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
  namespace: phoenix-vnext
spec:
  selector:
    app: grafana
  ports:
  - name: http
    port: 3000
    targetPort: 3000
  type: LoadBalancer
```

## Operations Guide

### Service Management

#### Starting Services

```bash
# Start all services
docker-compose up -d

# Start specific services
docker-compose up -d otelcol-main prometheus grafana

# Start with custom compose file
docker-compose -f docker-compose.yaml -f docker-compose.prod.yml up -d
```

#### Stopping Services

```bash
# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v

# Force stop and cleanup
docker-compose down --remove-orphans
```

#### Service Health Monitoring

```bash
# Check service status
docker-compose ps

# View service logs
docker-compose logs -f otelcol-main
docker-compose logs -f otelcol-observer
docker-compose logs --tail=100 control-loop-actuator

# Check resource usage
docker stats

# Health check endpoints
curl http://localhost:13133  # Main collector
curl http://localhost:13134  # Observer collector
```

### Configuration Management

#### Dynamic Configuration Updates

```bash
# Update control signals
echo "current_mode: aggressive" > configs/control/optimization_mode.yaml

# Reload Prometheus configuration
curl -X POST http://localhost:9090/-/reload

# Check configuration status
curl http://localhost:13133  # Collector health includes config status
```

#### Environment Variable Updates

```bash
# Update .env file
nano .env

# Restart services to apply changes
docker-compose restart otelcol-main otelcol-observer

# Or use environment-specific override
docker-compose --env-file .env.production restart
```

### Data Management

#### Volume Management

```bash
# List volumes
docker volume ls | grep phoenix

# Backup data volumes
docker run --rm -v phoenix-vnext_prometheus_data:/data -v $(pwd):/backup alpine tar czf /backup/prometheus_backup.tar.gz -C /data .

# Restore data volumes
docker run --rm -v phoenix-vnext_prometheus_data:/data -v $(pwd):/backup alpine tar xzf /backup/prometheus_backup.tar.gz -C /data

# Clean up old data
docker-compose down
docker volume rm phoenix-vnext_prometheus_data phoenix-vnext_grafana_data
```

#### Log Management

```bash
# Configure log rotation in docker-compose.yaml
services:
  otelcol-main:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"

# View and clean logs
docker system prune -f
docker container prune -f
```

### Performance Tuning

#### Resource Optimization

Monitor and adjust based on system performance:

```bash
# Memory usage monitoring
docker stats --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"

# Collector-specific metrics
curl http://localhost:8888/metrics | grep otelcol_process

# Adjust limits in .env
OTELCOL_MAIN_MEMORY_LIMIT_MIB=2048
OTELCOL_MAIN_GOMAXPROCS=2
```

#### Pipeline Tuning

```bash
# Monitor cardinality metrics
curl http://localhost:9090/api/v1/query?query=phoenix_pipeline_output_cardinality_estimate

# Adjust thresholds based on observed patterns
TARGET_OPTIMIZED_PIPELINE_TS_COUNT=30000
THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=20000
THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=40000
```

### Backup and Recovery

#### Configuration Backup

```bash
# Create backup directory
mkdir -p backups/$(date +%Y%m%d_%H%M%S)

# Backup configuration files
cp -r configs/ backups/$(date +%Y%m%d_%H%M%S)/
cp .env backups/$(date +%Y%m%d_%H%M%S)/
cp docker-compose.yaml backups/$(date +%Y%m%d_%H%M%S)/
```

#### Data Recovery

```bash
# Stop services
docker-compose down

# Restore data volumes
docker run --rm -v phoenix-vnext_prometheus_data:/data -v $(pwd)/backups/latest:/backup alpine tar xzf /backup/prometheus_backup.tar.gz -C /data

# Restart services
docker-compose up -d
```

### Security Operations

#### Access Control

```bash
# Change Grafana admin password
docker-compose exec grafana grafana-cli admin reset-admin-password <new-password>

# Update New Relic API keys
nano .env  # Update NEW_RELIC_LICENSE_KEY_* variables
docker-compose restart otelcol-main
```

#### Network Security

```bash
# Restrict external access (production)
# Edit docker-compose.yaml to remove external port mappings
# Use reverse proxy for external access

# Example nginx configuration for Grafana
server {
    listen 80;
    server_name phoenix-dashboard.company.com;
    
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Monitoring and Alerting

#### Key Metrics to Monitor

```bash
# Service availability
up{job="otelcol-main"}
up{job="otelcol-observer"}
up{job="prometheus"}

# Pipeline cardinality
phoenix_pipeline_output_cardinality_estimate

# System resources
otelcol_process_memory_rss
otelcol_process_cpu_seconds_total

# Control system health
phoenix_control_profile_switches_total
phoenix_control_last_switch_timestamp
```

#### Alerting Rules

Create `configs/monitoring/prometheus/rules/alerts.yml`:

```yaml
groups:
- name: phoenix.alerts
  rules:
  - alert: CollectorDown
    expr: up{job="otelcol-main"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "OpenTelemetry Collector is down"

  - alert: HighCardinality
    expr: phoenix_pipeline_output_cardinality_estimate > 50000
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High cardinality detected: {{ $value }} time series"

  - alert: ControlSystemStuck
    expr: increase(phoenix_control_profile_switches_total[1h]) == 0 and phoenix_pipeline_output_cardinality_estimate > 25000
    for: 15m
    labels:
      severity: warning
    annotations:
      summary: "Control system may be stuck in high cardinality state"
```