# Phoenix-vNext: Complete Deployment & Operations Guide

## 🚀 Quick Start (5 Minutes to Running System)

### Prerequisites
- Docker & Docker Compose
- Python 3.9+
- 8GB+ RAM (for optimal performance)
- Ports 3000, 4317-4320, 8888-8896, 9090, 9999 available

### 1. Launch the Complete System
```bash
# Clone and enter directory
cd phoenix-bench

# Start all services
docker compose up -d

# Verify system health (wait ~30 seconds for startup)
./test-phoenix-system.sh
```

### 2. Generate High-Cardinality Load
```bash
# Generate realistic high-cardinality metrics
python3 generate-high-cardinality-metrics.py --interval 3 --duration 60
```

### 3. Start Dynamic Optimization
```bash
# Start cardinality observer for automatic optimization
python3 phoenix-cardinality-observer.py --interval 20 &
```

### 4. Access Monitoring
- **Grafana Dashboard**: http://localhost:3000 (anonymous access enabled)
- **Prometheus**: http://localhost:9090
- **Pipeline Metrics**: 
  - Full: http://localhost:8888/metrics
  - Ultra: http://localhost:8890/metrics (optimized)

---

## 📊 Expected Results

After running the system for 2-3 minutes, you should see:

```
📊 Pipeline Cardinality Comparison:
   Full Pipeline:  6,000+ Phoenix metrics (100% baseline)
   Ultra Pipeline: 1,500+ Phoenix metrics (75%+ reduction)

💰 Cardinality Reduction: 75%+
🎯 Cost Savings Potential: 75%+
```

---

## 🏗️ Architecture Overview

### Core Services
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ High-Cardinality│    │   5-Pipeline     │    │   Cardinality   │
│   Generator     │───▶│  Main Collector  │◀──▶│    Observer     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │                          │
                              ▼                          ▼
                       ┌─────────────┐            ┌─────────────┐
                       │ Prometheus  │            │ Control     │
                       │ (Storage)   │            │ Signals     │
                       └─────────────┘            └─────────────┘
                              │
                              ▼
                       ┌─────────────┐
                       │   Grafana   │
                       │(Dashboard)  │
                       └─────────────┘
```

### Pipeline Differentiation
- **Full Pipeline (8888)**: No optimization - baseline cardinality
- **Opt Pipeline (8889)**: Moderate filtering - excludes debug metrics
- **Ultra Pipeline (8890)**: Aggressive filtering - core metrics only
- **Exp Pipeline (8895)**: Experimental algorithms - top-K simulation
- **Hybrid Pipeline (8896)**: Balanced approach - multiple techniques

---

## 🎯 Optimization Strategies

### 1. Filter-Based Optimization
```yaml
# Moderate: Remove debug and low-value metrics
filter/moderate:
  exclude:
    - "^system\.disk\..*debug.*$"
    - "^system\.filesystem\..*inodes.*$"
    - "^process\..*\.file_descriptor.*$"

# Ultra: Keep only core system metrics  
filter/ultra_aggressive:
  include:
    - "^(system_(cpu|memory)_.*|process_(cpu|memory)_.*)$"
```

### 2. Dynamic Threshold Control
```python
# Automatic mode switching based on cardinality
thresholds = {
    "moderate": 300.0,    # 0-25% optimization
    "adaptive": 375.0,    # 26-75% optimization  
    "ultra": 450.0        # 76-100% optimization
}
```

### 3. Real-time Adaptation
- Monitor cardinality every 20-30 seconds
- Auto-switch modes when thresholds exceeded
- Update control file for immediate optimization
- Preserve critical metrics while reducing noise

---

## 🔧 Advanced Configuration

### Environment Variables (.env file)
```bash
# New Relic Integration (optional)
NR_FULL_KEY=your_nr_api_key_for_full_pipeline
NR_OPT_KEY=your_nr_api_key_for_opt_pipeline
NR_ULTRA_KEY=your_nr_api_key_for_ultra_pipeline

# Custom Thresholds
THRESHOLD_MODERATE=300
THRESHOLD_ADAPTIVE=375
THRESHOLD_ULTRA=450

# Benchmark Configuration
BENCHMARK_ID=phoenix-vnext
DEPLOYMENT_ENV=production
```

### Custom Optimization Thresholds
```bash
# Start observer with custom thresholds
python3 phoenix-cardinality-observer.py \
  --moderate-threshold 250 \
  --adaptive-threshold 350 \
  --ultra-threshold 400 \
  --interval 15
```

### High-Load Testing
```bash
# Generate extreme cardinality load
python3 generate-high-cardinality-metrics.py \
  --interval 1 \
  --duration 300  # 5 minutes of high load
```

---

## 📈 Monitoring & Alerting

### Key Metrics to Watch
```prometheus
# Current cardinality per pipeline
count(count by (__name__)({__name__=~"phoenix_.*", job="phoenix-metrics-generator"}))

# Optimization effectiveness
(1 - (ultra_pipeline_count / full_pipeline_count)) * 100

# Control signal changes
phoenix_observer_mode

# System resource usage
rate(phoenix_system_cpu_usage_percent[5m])
```

### Grafana Dashboard Queries
```prometheus
# Pipeline Comparison
count(count by (__name__)({__name__=~"phoenix_.*", pipeline_id="full"}))
count(count by (__name__)({__name__=~"phoenix_.*", pipeline_id="ultra"}))

# Cost Reduction Calculation
(1 - (count(count by (__name__)({__name__=~"phoenix_.*", pipeline_id="ultra"})) / 
      count(count by (__name__)({__name__=~"phoenix_.*", pipeline_id="full"})))) * 100
```

---

## 🚨 Troubleshooting

### Common Issues

#### 1. Services Not Starting
```bash
# Check container status
docker compose ps

# View logs
docker logs phoenix-bench-otelcol-main-1
docker logs phoenix-bench-otelcol-observer-1

# Restart specific service
docker compose restart otelcol-main
```

#### 2. No Metrics Appearing
```bash
# Verify metrics generation
curl -s http://localhost:8888/metrics | grep "phoenix" | wc -l

# Check if generator is working
python3 generate-high-cardinality-metrics.py --interval 5 --duration 15

# Verify OTLP endpoint
curl -s http://localhost:4318/v1/metrics || echo "OTLP not accessible"
```

#### 3. Control Signals Not Working
```bash
# Check control file
cat configs/control_signals/opt_mode.yaml

# Manually trigger control signal
python3 -c "
import yaml
from datetime import datetime, timezone
control = {
    'mode': 'ultra',
    'last_updated': datetime.now(timezone.utc).isoformat(),
    'ts_count': 1000,
    'optimization_level': 100
}
with open('configs/control_signals/opt_mode.yaml', 'w') as f:
    yaml.dump(control, f)
print('Control signal updated')
"
```

#### 4. Port Conflicts
```bash
# Check port usage
lsof -i :8888
lsof -i :9090
lsof -i :3000

# Use alternative ports in docker-compose.yaml if needed
```

---

## 🎮 Demo Scenarios

### Scenario 1: Basic Optimization Demo
```bash
# 1. Start system
docker compose up -d && sleep 30

# 2. Generate baseline load
python3 generate-high-cardinality-metrics.py --duration 30

# 3. Check results
echo "Full: $(curl -s http://localhost:8888/metrics | grep phoenix | wc -l)"
echo "Ultra: $(curl -s http://localhost:8890/metrics | grep phoenix | wc -l)"
```

### Scenario 2: Dynamic Threshold Testing
```bash
# 1. Start with low threshold
python3 phoenix-cardinality-observer.py --moderate-threshold 100 --interval 10 &

# 2. Generate increasing load
for i in {1..5}; do
  python3 generate-high-cardinality-metrics.py --duration 20
  sleep 10
done

# 3. Observe automatic mode switching
tail -f configs/control_signals/opt_mode.yaml
```

### Scenario 3: Production Simulation
```bash
# 1. Run complete demo script
python3 run-complete-demo.py

# 2. Monitor in Grafana
# Open http://localhost:3000 and import phoenix-dashboard.json

# 3. Observe real-time optimization
```

---

## 📚 API Reference

### Metrics Endpoints
- `GET /metrics` - Pipeline-specific metrics
- `GET /-/healthy` - Health check
- `GET /debug/pprof/*` - Performance profiling

### Control File Schema
```yaml
mode: "moderate|adaptive|ultra"           # Required
last_updated: "2025-05-22T10:00:00Z"     # Required (ISO-8601)
config_version: 12345                    # Required (Integer)
correlation_id: "observer-1621234567"    # Required (String)
optimization_level: 75                   # 0-100 scale
ts_count: 400                           # Current time series count
thresholds:                             # Threshold configuration
  moderate: 300.0
  adaptive: 375.0
  ultra: 450.0
```

---

## 🏆 Success Metrics

### Performance Targets
- ✅ **>70% Cardinality Reduction** in ultra pipeline
- ✅ **<30s Response Time** for mode switching
- ✅ **99%+ Uptime** for all core services
- ✅ **<5% CPU Overhead** for optimization processing

### Business KPIs
- **Cost Reduction**: 70%+ storage/processing savings
- **Quality Preservation**: Critical metrics retained
- **Automation Level**: 100% hands-off operation
- **Scalability**: Handles 10K+ unique time series

---

## 🚀 Production Deployment

### Recommended Hardware
- **CPU**: 4+ cores
- **RAM**: 8GB+ (16GB for high-load)
- **Storage**: 100GB+ SSD
- **Network**: 1Gbps+ for metric ingestion

### Security Considerations
- Use secrets management for API keys
- Enable TLS for production endpoints
- Implement proper authentication for Grafana
- Monitor for anomalous cardinality patterns

### Scaling Guidelines
- Scale collector replicas based on metric volume
- Use load balancer for metric ingestion
- Implement metric retention policies
- Consider multi-region deployment for HA

---

**🎊 Phoenix-vNext: Ready for Production Cardinality Optimization! 🚀**