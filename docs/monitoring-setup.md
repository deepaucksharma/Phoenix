# Phoenix-vNext Monitoring Setup Guide

## Overview

The Phoenix-vNext system includes a comprehensive monitoring stack with 6 specialized Grafana dashboards, Prometheus alerting rules, and complete observability for the 3-pipeline cardinality optimization system.

## 📊 Dashboard Suite

### 1. Phoenix v3 - Mission Control
**File**: `phoenix-mission-control.json`  
**Purpose**: Single-screen NOC board showing overall stack health, optimization effectiveness, and key performance indicators.

**Key Panels**:
- Cost reduction gauge (Optimised vs Full pipeline)
- Active optimization profile status
- Time series count by pipeline
- Top CPU/Memory consuming processes
- Collector health status grid
- Resource usage monitoring

### 2. Phoenix v3 - Pipeline Efficiency & Cost
**File**: `phoenix-pipeline-efficiency.json`  
**Purpose**: Deep-dive into cost analysis and optimization effectiveness.

**Key Panels**:
- Estimated daily data points by pipeline
- 7-day trend analysis for optimized pipeline
- Cost savings percentage tracking
- Optimization profile change timeline
- Profile overlay on time series counts

### 3. Phoenix v3 - Process Hotspots Deep-Dive
**File**: `phoenix-process-hotspots.json`  
**Purpose**: Detailed analysis of resource-consuming processes.

**Key Panels**:
- Process CPU usage heatmap by executable and host
- P95 memory trends by priority tier
- Memory leak detection (30-minute RSS delta)
- Top 10 CPU and memory consuming processes
- Template variables for pipeline view selection

### 4. Phoenix v3 - Adaptive Control Loop Analysis
**File**: `phoenix-adaptive-control-loop.json`  
**Purpose**: Analysis of the PID-like control system behavior.

**Key Panels**:
- Current optimization profile from control file
- Optimized pipeline TS count with threshold lines
- PID error calculation visualization
- Control profile transition history
- Hysteresis window and stability analysis

### 5. Phoenix v3 - OTel Collector Operations & pprof
**File**: `phoenix-collector-ops.json`  
**Purpose**: Operational metrics and deep diagnostics for collectors.

**Key Panels**:
- Heap RSS and GC pause duration
- Container CPU and memory usage
- Exporter queue depth monitoring
- Batch processor performance metrics
- pprof flamegraph links for diagnostics
- Collector health status indicators

### 6. Legacy Dashboard
**File**: `phoenix-v3-ultra-overview.json`  
**Purpose**: Existing overview dashboard (maintained for compatibility)

## 🔧 Configuration Files

### Grafana Configuration
- **`dashboards_provider.yaml`**: Configures Grafana to discover dashboard JSON files
- **`grafana-datasource.yaml`**: Prometheus datasource configuration
- **`grafana-dashboards.yaml`**: Legacy dashboard provider (replaced by dashboards_provider.yaml)

### Prometheus Configuration
- **`prometheus.yaml`**: Main Prometheus configuration with scrape jobs
- **`rules/phoenix_rules.yml`**: Recording rules and alerting rules

## 📈 Metrics Architecture

### Scrape Jobs
1. **`otelcol-main-telemetry`** (port 8888): Full pipeline + collector self-metrics
2. **`otelcol-main-opt-output`** (port 8889): Optimized pipeline metrics
3. **`otelcol-main-exp-output`** (port 8890): Experimental pipeline metrics
4. **`otelcol-observer-metrics`** (port 9888): Observer KPIs and control metrics
5. **`control-loop-actuator-metrics`**: Control script metrics (when available)

### Recording Rules
- **`phoenix:cost_reduction_ratio`**: Primary cost savings KPI
- **`phoenix:cost_reduction_ratio_direct`**: Alternative calculation method
- **`phoenix:control_active_profile_code`**: Current optimization profile
- **`phoenix:optimised_pipeline_effectiveness`**: Rollup effectiveness measurement

### Alerting Rules
- **`PhoenixOptimizationDrift`**: Cost reduction below 40% threshold
- **`PhoenixControlLoopFileStale`**: Control file not updating
- **`OtelcolMainHighCPU/Memory`**: Resource usage alerts
- **`PhoenixPipelineDown`**: Pipeline endpoint unavailability
- **`PhoenixHighCardinalitySpike`**: Cardinality threshold breaches

## 🚀 Quick Start

### 1. Initialize System
```bash
cd /home/deepak/phoenix-vnext
./scripts/initialize-environment.sh
```

### 2. Start Monitoring Stack
```bash
docker-compose up -d prometheus grafana
```

### 3. Access Dashboards
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090

### 4. Load Dashboards
Dashboards are automatically provisioned via:
```
configs/monitoring/grafana/dashboards/
├── phoenix-mission-control.json
├── phoenix-pipeline-efficiency.json
├── phoenix-process-hotspots.json
├── phoenix-adaptive-control-loop.json
└── phoenix-collector-ops.json
```

## 🔍 Dashboard Navigation

### Mission Control (Primary)
- **Use Case**: Wall display, NOC monitoring, executive overview
- **Refresh**: 15 seconds
- **Time Range**: Last 1 hour

### Pipeline Efficiency (Analysis)
- **Use Case**: Cost analysis, optimization trending, profile evaluation
- **Refresh**: 30 seconds  
- **Time Range**: Last 6 hours

### Process Hotspots (Troubleshooting)
- **Use Case**: Performance debugging, resource leak detection
- **Template Variables**: Pipeline view selection, host filtering
- **Time Range**: Last 1 hour

### Control Loop (Advanced)
- **Use Case**: Control system debugging, PID tuning, stability analysis
- **Refresh**: 30 seconds
- **Time Range**: Last 3 hours

### Collector Ops (Operations)
- **Use Case**: Collector performance, deep diagnostics, operational health
- **Template Variables**: Collector instance selection
- **Special Features**: Direct pprof and zpages links

## 🎯 Key Performance Indicators

### Primary KPIs
1. **Cost Reduction Ratio**: Target >70%, Alert <40%
2. **Active Time Series Counts**: Monitor cardinality across pipelines
3. **Optimization Profile**: Conservative/Balanced/Aggressive transitions
4. **Resource Usage**: CPU <90%, Memory <85% of limits

### Secondary KPIs
1. **Top Process Rankings**: CPU and Memory consumers
2. **Rollup Effectiveness**: Processes in "Others" category
3. **Control Loop Health**: File update frequency, version changes
4. **Pipeline Effectiveness**: Experimental vs Optimized reductions

## 🚨 Alert Thresholds

### Critical Alerts
- Pipeline endpoints down (1 minute)
- Memory usage >85% of limits (10 minutes)
- Control file stale (20 minutes total)

### Warning Alerts  
- Cost reduction <40% (10 minutes)
- CPU usage >90% of limits (10 minutes)
- High cardinality spikes (5 minutes)

## 🛠️ Troubleshooting

### Dashboard Not Loading
1. Check Grafana logs: `docker-compose logs grafana`
2. Verify dashboard provider config: `configs/monitoring/grafana/dashboards_provider.yaml`
3. Ensure dashboard files are mounted: `/var/lib/grafana/dashboards`

### Missing Metrics
1. Check Prometheus targets: http://localhost:9090/targets
2. Verify collector endpoints are accessible
3. Review prometheus.yaml scrape configuration

### Control Loop Issues
1. Monitor control file updates: `watch cat configs/control/optimization_mode.yaml`
2. Check actuator script logs: `docker-compose logs control-loop-actuator`
3. Verify observer metrics: http://localhost:9888/metrics

## 📚 Additional Resources

- **pprof Endpoints**: 
  - Main Collector: http://localhost:1777/debug/pprof/
  - Observer: http://localhost:1778/debug/pprof/
- **Zpages**: 
  - Main: http://localhost:55679/debug/
  - Observer: http://localhost:55680/debug/
- **Health Checks**:
  - Main: http://localhost:13133
  - Observer: http://localhost:13134