# Phoenix Core Functional Requirements

## 🎯 Primary Purpose
Phoenix must automatically optimize OpenTelemetry metric cardinality to reduce costs while preserving signal quality.

## 🔑 Core Functional Requirements

### 1. **Multi-Pipeline Processing**
- **Full Fidelity Pipeline**: Baseline with all metrics
- **Optimized Pipeline**: Moderate cardinality reduction
- **Experimental TopK Pipeline**: Aggressive optimization
- **Cardinality Observatory**: Explosion detection

### 2. **Adaptive Control System**
- Monitor cardinality in real-time
- Switch between optimization modes automatically
- Prevent oscillation with hysteresis
- Handle cardinality explosions

### 3. **Performance Validation**
- Continuous benchmarking of pipelines
- Measure signal preservation
- Track cost reduction
- Detect performance degradation

### 4. **Observability**
- Pipeline metrics exposed to Prometheus
- Grafana dashboards for monitoring
- Recording rules for analysis
- Alerting on anomalies

## ❌ Current Gaps

### Critical Missing Components:
1. **Benchmark Controller** - No validation of pipeline effectiveness
2. **Recording Rules** - Missing key metrics for decision making
3. **Hysteresis Logic** - Control loop can oscillate
4. **Explosion Detection** - No protection against cardinality bombs
5. **Integration Tests** - No end-to-end validation

### Functional Issues:
1. Control loop lacks stability mechanisms
2. No measurement of signal quality loss
3. Missing cost tracking metrics
4. No validation of optimization effectiveness

## ✅ What Must Work

For Phoenix to deliver its core value:
1. **Pipelines must process metrics correctly**
2. **Control system must switch modes reliably**
3. **System must prevent cardinality explosions**
4. **Monitoring must show optimization impact**
5. **Validation must prove cost reduction**