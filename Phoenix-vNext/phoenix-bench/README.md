# Phoenix-vNext 5-Pipeline Benchmarking System

## Overview

Phoenix-vNext is a sophisticated OpenTelemetry-based benchmarking system that implements 5 parallel processing pipelines to evaluate different cardinality optimization strategies. This consolidated version represents the final, streamlined implementation.

## Architecture Components

### Core Services
- **Main Collector** (`otelcol-main`): Implements 5 parallel pipelines (full, opt, ultra, exp, hybrid)
- **Observer Collector** (`otelcol-observer`): Monitors cardinality and generates dynamic control signals
- **Synthetic Metrics Generator**: Creates test data for pipeline comparison
- **Prometheus**: Metrics storage and querying
- **Grafana**: Visualization and dashboards

### 5-Pipeline Strategy
1. **Full Pipeline**: Baseline with minimal processing (250 series target)
2. **Opt Pipeline**: Moderate optimization with filtering (150 series target)  
3. **Ultra Pipeline**: Aggressive optimization (50 series target)
4. **Exp Pipeline**: Experimental Top-K algorithms (100 series target)
5. **Hybrid Pipeline**: Balanced multi-technique approach (125 series target)

## Quick Start

1. **Initialize Environment**:
   ```bash
   # Ensure .env file has your New Relic API keys
   cp .env.template .env
   # Edit .env with your actual API keys
   ```

2. **Start the System**:
   ```bash
   docker compose up -d
   ```

3. **Verify System Health**:
   ```bash
   ./test-phoenix-system.sh
   ```

4. **Access Dashboards**:
   - Grafana: http://localhost:3000 (admin/admin)
   - Prometheus: http://localhost:9090

## Configuration Files

### Essential Configurations
- `configs/collectors/otelcol-main.yaml` - Main 5-pipeline collector
- `configs/collectors/otelcol-observer.yaml` - Control signal generator  
- `configs/control_signals/opt_mode.yaml` - Dynamic control file
- `configs/metrics/synthetic-metrics.yaml` - Test data generator
- `docker-compose.yaml` - Service orchestration

### Schema-Aligned Thresholds
The system uses three optimization modes aligned with the main collector's schema:
- **moderate**: 300 series threshold (optimization_level: 0-25)
- **adaptive**: 375 series threshold (optimization_level: 26-75)  
- **ultra**: 450 series threshold (optimization_level: 76-100)

## Control Signal Mechanism

The observer monitors cardinality metrics and dynamically updates `opt_mode.yaml`:

```yaml
mode: "moderate"              # Current optimization mode
optimization_level: 0         # Fine-grained level (0-100)
ts_count: 142                # Current time series count
thresholds:                  # Schema-aligned thresholds
  moderate: 300.0
  adaptive: 375.0
  ultra: 450.0
```

## Testing and Verification

Use the consolidated test suite to verify system health:

```bash
./test-phoenix-system.sh
```

The test suite validates:
- ✅ Schema coherence between components
- ✅ Component health and connectivity  
- ✅ Control signal propagation
- ✅ Pipeline metrics generation
- ✅ YAML configuration syntax

## Key Features

### Dynamic Optimization
- Real-time cardinality monitoring
- Automatic mode switching based on thresholds
- Schema-aligned control signals

### Comprehensive Instrumentation
- Per-pipeline cardinality tracking
- Quality vs. cost trade-off metrics
- Control loop stability monitoring

### Production-Ready
- Robust error handling
- Comprehensive logging
- Health check endpoints
- Graceful degradation

## Monitoring

### Key Metrics to Watch
- `phoenix_opt_ts_active` - Current cardinality in opt pipeline
- `phoenix_observer_mode` - Current optimization mode
- `phoenix_quality_*` - Data quality scores per pipeline
- `phoenix_system_cpu_time_seconds_total` - Consolidated dashboard metric

### Alerting Recommendations
- Alert on sustained ultra mode (indicates high cardinality)
- Monitor for rapid mode oscillation (control instability)
- Track pipeline effectiveness scores

## Troubleshooting

### Common Issues
1. **Containers Restarting**: Check YAML syntax with `./test-phoenix-system.sh`
2. **No Phoenix Metrics**: Verify synthetic metrics generator is running
3. **Mode Incoherence**: Allow 30-60 seconds for control signal propagation
4. **High Resource Usage**: Monitor pipeline cardinality and adjust thresholds

### Debug Commands
```bash
# Check container status
docker compose ps

# View main collector logs
docker logs phoenix-bench-otelcol-main-1

# Check observer logs  
docker logs phoenix-bench-otelcol-observer-1

# Validate configurations
python3 -c "import yaml; yaml.safe_load(open('configs/collectors/otelcol-main.yaml'))"
```

## Environment Variables

Required for New Relic export:
- `NR_FULL_KEY` - API key for full pipeline data
- `NR_OPT_KEY` - API key for optimized pipeline data  
- `NR_ULTRA_KEY` - API key for ultra-optimized data
- `NR_EXP_KEY` - API key for experimental pipeline
- `NR_HYBRID_KEY` - API key for hybrid pipeline

Optional configuration:
- `THRESHOLD_MODERATE=300` - Moderate optimization threshold
- `THRESHOLD_ADAPTIVE=375` - Adaptive optimization threshold  
- `THRESHOLD_ULTRA=450` - Ultra optimization threshold
- `BENCHMARK_ID=phoenix-vnext` - Benchmark identifier

## Advanced Usage

### Custom Thresholds
Modify thresholds in `.env` file and restart:
```bash
THRESHOLD_MODERATE=250
THRESHOLD_ADAPTIVE=350
THRESHOLD_ULTRA=400
docker compose restart
```

### Pipeline Analysis
Compare pipeline effectiveness in Grafana:
1. Navigate to Phoenix dashboard
2. Compare quality vs. cardinality metrics
3. Analyze cost reduction percentages
4. Monitor mode transition frequency

### Integration Testing
The system supports A/B testing by sending identical data through all 5 pipelines simultaneously, enabling direct comparison of optimization strategies.

## File Structure

```
phoenix-bench/
├── configs/
│   ├── collectors/          # OpenTelemetry collector configs
│   ├── control_signals/     # Dynamic control files  
│   ├── dashboards/          # Grafana dashboards
│   ├── metrics/             # Synthetic data generators
│   ├── monitoring/          # Prometheus/Grafana configs
│   └── processors/          # Reusable processor configs
├── data/                    # Persistent storage
├── docs/                    # Additional documentation
├── docker-compose.yaml     # Service orchestration
├── test-phoenix-system.sh   # Consolidated test suite
└── README_FINAL.md         # This file
```

## Success Metrics

### Technical KPIs
- Cardinality reduction: 40-80% depending on mode
- Data quality retention: 85-100% across pipelines  
- Control loop stability: <3 mode changes per hour
- Pipeline latency: <100ms p95

### Business Value
- Reduced observability costs through intelligent cardinality management
- Maintained data quality for critical monitoring use cases
- Dynamic adaptation to workload changes
- Clear cost vs. quality trade-off visibility

---

This consolidated Phoenix-vNext system provides a production-ready foundation for cardinality optimization research and development.