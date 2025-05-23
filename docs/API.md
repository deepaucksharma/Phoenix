# Phoenix-vNext API Documentation

## Overview

This document provides comprehensive API documentation for all Phoenix-vNext services. All APIs use REST over HTTP with JSON payloads unless otherwise specified.

## Table of Contents

- [Control Actuator API](#control-actuator-api)
- [Anomaly Detector API](#anomaly-detector-api)
- [Benchmark Controller API](#benchmark-controller-api)
- [OpenTelemetry Collector APIs](#opentelemetry-collector-apis)
- [Prometheus Query API](#prometheus-query-api)

---

## Control Actuator API

**Base URL**: `http://localhost:8081`

The Control Actuator provides real-time metrics about the PID control loop and system state.

### GET /metrics

Returns current control loop state and metrics.

**Response**: `200 OK`

```json
{
  "current_mode": "balanced",
  "transition_count": 3,
  "stability_score": 0.92,
  "integral_error": 125.5,
  "last_error": -523.0,
  "uptime_seconds": 3600
}
```

**Fields**:
- `current_mode`: Current optimization mode (conservative/balanced/aggressive)
- `transition_count`: Total number of mode transitions since startup
- `stability_score`: Control loop stability (0-1, higher is more stable)
- `integral_error`: Accumulated error for PID control
- `last_error`: Most recent error value
- `uptime_seconds`: Time since last mode change

### POST /anomaly (Internal)

Webhook endpoint for anomaly detector notifications.

**Request Body**:
```json
{
  "anomaly_type": "cardinality_explosion",
  "severity": "critical",
  "timestamp": "2024-05-23T10:30:00Z",
  "recommended_action": "switch_to_aggressive"
}
```

**Response**: `200 OK`
```json
{
  "status": "acknowledged",
  "action_taken": "mode_change_scheduled"
}
```

---

## Anomaly Detector API

**Base URL**: `http://localhost:8082`

The Anomaly Detector monitors metrics and identifies anomalous patterns.

### GET /health

Health check endpoint.

**Response**: `200 OK`
```text
OK
```

### GET /alerts

Returns all detected anomalies and alerts.

**Response**: `200 OK`
```json
[
  {
    "id": "cardinality-1234567890",
    "anomaly": {
      "detector_name": "statistical_zscore",
      "metric_name": "phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate",
      "timestamp": "2024-05-23T10:30:00Z",
      "value": 35000,
      "expected": 20000,
      "severity": "high",
      "confidence": 0.95,
      "labels": {
        "pipeline": "optimized"
      },
      "description": "Value 35000 is 4.2 standard deviations from mean 20000"
    },
    "created_at": "2024-05-23T10:30:05Z",
    "status": "active",
    "action_taken": "Notified control loop to switch to aggressive mode"
  }
]
```

**Query Parameters**:
- `status`: Filter by alert status (active/resolved/acknowledged)
- `severity`: Filter by severity (low/medium/high/critical)
- `detector`: Filter by detector name
- `since`: ISO8601 timestamp to filter recent alerts

**Example**: `/alerts?status=active&severity=high&since=2024-05-23T00:00:00Z`

### Alert Object Schema

```typescript
interface Alert {
  id: string;
  anomaly: {
    detector_name: string;
    metric_name: string;
    timestamp: string;
    value: number;
    expected: number;
    severity: "low" | "medium" | "high" | "critical";
    confidence: number; // 0-1
    labels: Record<string, string>;
    description: string;
  };
  created_at: string;
  status: "active" | "resolved" | "acknowledged";
  action_taken?: string;
}
```

---

## Benchmark Controller API

**Base URL**: `http://localhost:8083`

The Benchmark Controller runs performance validation scenarios.

### GET /health

Health check endpoint.

**Response**: `200 OK`
```text
OK
```

### GET /benchmark/scenarios

Lists all available benchmark scenarios.

**Response**: `200 OK`
```json
[
  {
    "name": "baseline_steady_state",
    "description": "Validate system behavior under steady state load",
    "duration": "10m"
  },
  {
    "name": "cardinality_spike",
    "description": "Test control loop response to sudden cardinality increase",
    "duration": "15m"
  },
  {
    "name": "gradual_growth",
    "description": "Validate smooth transitions during gradual load increase",
    "duration": "20m"
  },
  {
    "name": "wave_pattern",
    "description": "Test hysteresis under oscillating load",
    "duration": "30m"
  }
]
```

### POST /benchmark/run

Starts a benchmark scenario.

**Request Body**:
```json
{
  "scenario": "baseline_steady_state"
}
```

**Response**: `202 Accepted`
```json
{
  "status": "started",
  "scenario": "baseline_steady_state",
  "estimated_completion": "2024-05-23T10:40:00Z"
}
```

**Error Response**: `400 Bad Request`
```json
{
  "error": "scenario not found",
  "available_scenarios": ["baseline_steady_state", "cardinality_spike", "gradual_growth", "wave_pattern"]
}
```

### GET /benchmark/results

Returns benchmark results.

**Response**: `200 OK`
```json
[
  {
    "scenario": "baseline_steady_state",
    "start_time": "2024-05-23T10:00:00Z",
    "end_time": "2024-05-23T10:10:00Z",
    "metrics": {
      "signal_preservation": 0.98,
      "cardinality_reduction": 15.2,
      "cpu_usage": 45.3,
      "memory_usage": 412.5,
      "pipeline_latency_p99": 42.5,
      "control_stability_score": 0.95
    },
    "control_behavior": [
      {
        "timestamp": "2024-05-23T10:05:00Z",
        "from_mode": "conservative",
        "to_mode": "balanced",
        "reason": "cardinality_threshold"
      }
    ],
    "resource_usage": {
      "avg_cpu_percent": 45.3,
      "max_cpu_percent": 62.1,
      "avg_memory_mb": 412.5,
      "max_memory_mb": 523.8,
      "p99_latency_ms": 42.5
    },
    "passed": true,
    "failure_reasons": []
  }
]
```

**Query Parameters**:
- `scenario`: Filter results by scenario name
- `passed`: Filter by pass/fail status (true/false)
- `limit`: Maximum number of results to return

---

## OpenTelemetry Collector APIs

### Main Collector

**Base URL**: `http://localhost:13133`

#### GET /

Health check and component status.

**Response**: `200 OK`
```json
{
  "status": "ready",
  "version": "0.91.0",
  "uptime": 3600,
  "components": {
    "receivers": ["otlp"],
    "processors": ["memory_limiter", "batch", "resource"],
    "exporters": ["prometheus/full", "prometheus/optimized", "prometheus/experimental"]
  }
}
```

### Metrics Endpoints

Each pipeline exports metrics to a dedicated Prometheus endpoint:

#### GET http://localhost:8888/metrics
Full fidelity pipeline metrics (Prometheus format)

#### GET http://localhost:8889/metrics
Optimized pipeline metrics (Prometheus format)

#### GET http://localhost:8890/metrics
Experimental TopK pipeline metrics (Prometheus format)

### Debug Endpoints

#### GET http://localhost:1777/debug/pprof/
pprof profiling endpoints for performance analysis

Available profiles:
- `/debug/pprof/heap`: Memory allocations
- `/debug/pprof/goroutine`: All goroutines
- `/debug/pprof/cpu`: CPU profile (requires `?seconds=30` parameter)
- `/debug/pprof/trace`: Execution trace

#### GET http://localhost:55679/debug/tracez
zpages trace information for debugging OTLP processing

---

## Prometheus Query API

**Base URL**: `http://localhost:9090`

Phoenix uses Prometheus for metrics storage and querying.

### Common Queries

#### Get current cardinality
```
GET /api/v1/query?query=phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate
```

#### Get signal preservation score
```
GET /api/v1/query?query=phoenix:signal_preservation_score
```

#### Get control stability over time
```
GET /api/v1/query_range?query=phoenix:control_stability_score&start=2024-05-23T00:00:00Z&end=2024-05-23T12:00:00Z&step=60s
```

### Recording Rules

Phoenix provides pre-computed metrics via recording rules:

| Rule | Description | Unit |
|------|-------------|------|
| `phoenix:signal_preservation_score` | Data fidelity across pipelines | ratio (0-1) |
| `phoenix:cardinality_efficiency_ratio` | Cardinality reduction effectiveness | ratio (0-1) |
| `phoenix:resource_efficiency_score` | Cost per datapoint metric | score |
| `phoenix:control_stability_score` | Control loop stability | ratio (0-1) |
| `phoenix:cardinality_growth_rate` | Rate of cardinality change | series/sec |
| `phoenix:cardinality_explosion_risk` | Risk of cardinality explosion | score (0-10) |

---

## Error Responses

All APIs use consistent error response format:

```json
{
  "error": "error_type",
  "message": "Human readable error message",
  "details": {
    "field": "additional context"
  },
  "timestamp": "2024-05-23T10:30:00Z"
}
```

Common HTTP status codes:
- `200 OK`: Success
- `202 Accepted`: Async operation started
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

---

## Rate Limiting

APIs implement basic rate limiting:
- Default: 100 requests per minute per IP
- Burst: 20 requests
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

---

## Authentication

Currently, all APIs are unauthenticated for local development. Production deployments should implement:
- API key authentication
- mTLS for service-to-service communication
- RBAC for different user roles

---

## WebSocket Support (Future)

Planned WebSocket endpoints for real-time updates:
- `/ws/alerts`: Real-time anomaly alerts
- `/ws/metrics`: Streaming metrics updates
- `/ws/control`: Control loop state changes

---

## SDK Examples

### Python
```python
import requests

# Get control state
response = requests.get('http://localhost:8081/metrics')
control_state = response.json()
print(f"Current mode: {control_state['current_mode']}")

# Run benchmark
response = requests.post('http://localhost:8083/benchmark/run', 
                        json={'scenario': 'baseline_steady_state'})
print(f"Benchmark started: {response.json()['status']}")
```

### Go
```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type ControlMetrics struct {
    CurrentMode     string  `json:"current_mode"`
    StabilityScore  float64 `json:"stability_score"`
}

func main() {
    resp, _ := http.Get("http://localhost:8081/metrics")
    var metrics ControlMetrics
    json.NewDecoder(resp.Body).Decode(&metrics)
    fmt.Printf("Mode: %s, Stability: %.2f\n", 
               metrics.CurrentMode, metrics.StabilityScore)
}
```

### curl
```bash
# Get anomaly alerts
curl -s http://localhost:8082/alerts | jq '.[] | {severity, metric: .anomaly.metric_name}'

# Check benchmark results
curl -s http://localhost:8083/benchmark/results | jq '.[0] | {scenario, passed, metrics}'

# Query Prometheus
curl -s "http://localhost:9090/api/v1/query?query=up" | jq '.data.result'
```

---

## API Versioning

APIs follow semantic versioning:
- Current version: v1 (implicit, no version in URL)
- Future versions will use URL prefix: `/v2/metrics`
- Deprecation policy: 6 months notice before removing endpoints

---

## OpenAPI Specification

OpenAPI/Swagger specifications are available at:
- Control Actuator: `http://localhost:8081/openapi.json` (planned)
- Anomaly Detector: `http://localhost:8082/openapi.json` (planned)
- Benchmark Controller: `http://localhost:8083/openapi.json` (planned)
