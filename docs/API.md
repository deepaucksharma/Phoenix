# Phoenix-vNext API Documentation

## Overview

Phoenix-vNext exposes several HTTP endpoints for monitoring, health checks, debugging, and data ingestion. This document provides comprehensive API reference for all exposed endpoints.

## Base URLs

| Service | Base URL | Description |
|---------|----------|-------------|
| Main Collector | http://localhost:4318 | OTLP data ingestion |
| Main Collector Metrics | http://localhost:8888-8890 | Pipeline-specific metrics |
| Observer Collector | http://localhost:9888 | Observer metrics |
| Prometheus | http://localhost:9090 | Metrics storage and querying |
| Grafana | http://localhost:3000 | Visualization dashboards |
| Health Checks | http://localhost:13133-13134 | Service health status |

## Data Ingestion APIs

### OTLP HTTP Endpoint

**Endpoint:** `POST /v1/metrics`  
**Base URL:** `http://localhost:4318`  
**Content-Type:** `application/json` or `application/x-protobuf`

Accepts OpenTelemetry metrics in OTLP format for processing through all three pipelines.

#### Request Format

```json
{
  "resourceMetrics": [
    {
      "resource": {
        "attributes": [
          {
            "key": "service.name",
            "value": {"stringValue": "synthetic-app"}
          },
          {
            "key": "host.name", 
            "value": {"stringValue": "host-001"}
          }
        ]
      },
      "scopeMetrics": [
        {
          "scope": {
            "name": "process-metrics",
            "version": "1.0.0"
          },
          "metrics": [
            {
              "name": "process.cpu.time",
              "description": "Process CPU time",
              "unit": "s",
              "sum": {
                "dataPoints": [
                  {
                    "timeUnixNano": "1640995200000000000",
                    "asDouble": 1.234,
                    "attributes": [
                      {
                        "key": "process.executable.name",
                        "value": {"stringValue": "nginx"}
                      }
                    ]
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ]
}
```

#### Response Codes

| Code | Description |
|------|-------------|
| 200 | Success - metrics accepted |
| 400 | Bad Request - invalid payload |
| 429 | Too Many Requests - rate limited |
| 500 | Internal Server Error |

#### Example Usage

```bash
# Send metrics via curl
curl -X POST http://localhost:4318/v1/metrics \
  -H "Content-Type: application/json" \
  -d @sample-metrics.json

# Send with compression
curl -X POST http://localhost:4318/v1/metrics \
  -H "Content-Type: application/json" \
  -H "Content-Encoding: gzip" \
  --data-binary @sample-metrics.json.gz
```

## Metrics Endpoints

### Pipeline-Specific Metrics

Each pipeline exposes its processed metrics via dedicated Prometheus endpoints:

#### Full Fidelity Pipeline

**Endpoint:** `GET /metrics`  
**Base URL:** `http://localhost:8888`

Returns metrics from the full fidelity pipeline with minimal processing applied.

```bash
curl http://localhost:8888/metrics
```

#### Optimized Pipeline

**Endpoint:** `GET /metrics`  
**Base URL:** `http://localhost:8889`

Returns metrics from the optimized pipeline with selective filtering and aggregation.

```bash
curl http://localhost:8889/metrics
```

#### Experimental TopK Pipeline

**Endpoint:** `GET /metrics`  
**Base URL:** `http://localhost:8890`

Returns metrics from the experimental pipeline with TopK sampling applied.

```bash
curl http://localhost:8890/metrics
```

### Observer Metrics

**Endpoint:** `GET /metrics`  
**Base URL:** `http://localhost:9888`

Returns aggregated cardinality estimates and control system metrics.

```bash
curl http://localhost:9888/metrics
```

#### Key Observer Metrics

| Metric Name | Description | Labels |
|-------------|-------------|---------|
| `phoenix_pipeline_output_cardinality_estimate` | Estimated time series count per pipeline | `pipeline` |
| `phoenix_control_profile_switches_total` | Total profile switches | `from_profile`, `to_profile` |
| `phoenix_control_last_switch_timestamp` | Timestamp of last profile switch | |
| `phoenix_control_current_profile` | Current optimization profile (0=conservative, 1=balanced, 2=aggressive) | |

## Health Check APIs

### Main Collector Health

**Endpoint:** `GET /`  
**Base URL:** `http://localhost:13133`

Returns health status of the main OpenTelemetry collector.

```bash
curl http://localhost:13133
```

**Response:**
```json
{
  "status": "Server available",
  "upSince": "2024-01-15T10:30:00Z",
  "uptime": "2h30m15s"
}
```

### Observer Health

**Endpoint:** `GET /`  
**Base URL:** `http://localhost:13134`

Returns health status of the observer collector.

```bash
curl http://localhost:13134
```

## Debugging APIs

### pprof Profiling

**Base URLs:**
- Main Collector: `http://localhost:1777`
- Observer: `http://localhost:1778`

#### Available Endpoints

| Endpoint | Description |
|----------|-------------|
| `/debug/pprof/` | Profile index |
| `/debug/pprof/heap` | Memory heap profile |
| `/debug/pprof/goroutine` | Goroutine profile |
| `/debug/pprof/profile` | CPU profile |
| `/debug/pprof/trace` | Execution trace |

#### Usage Examples

```bash
# Get heap profile
curl http://localhost:1777/debug/pprof/heap > heap.prof

# Analyze with go tool
go tool pprof heap.prof

# Get 30-second CPU profile
curl "http://localhost:1777/debug/pprof/profile?seconds=30" > cpu.prof

# Get goroutine profile
curl http://localhost:1777/debug/pprof/goroutine?debug=1
```

### zpages Internal State

**Base URLs:**
- Main Collector: `http://localhost:55679`
- Observer: `http://localhost:55680`

#### Available Pages

| Endpoint | Description |
|----------|-------------|
| `/debug/servicez` | Service status overview |
| `/debug/pipelinez` | Pipeline status and metrics |
| `/debug/extensionz` | Extension status |
| `/debug/receiverz` | Receiver status and stats |
| `/debug/processorz` | Processor status and stats |
| `/debug/exporterz` | Exporter status and stats |
| `/debug/configz` | Current configuration |

#### Usage Examples

```bash
# View pipeline status
curl http://localhost:55679/debug/pipelinez | jq '.'

# Check receiver statistics
curl http://localhost:55679/debug/receiverz

# View current configuration
curl http://localhost:55679/debug/configz
```

## Prometheus Query API

### Base Endpoints

**Base URL:** `http://localhost:9090`

#### Query Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/query` | GET/POST | Instant queries |
| `/api/v1/query_range` | GET/POST | Range queries |
| `/api/v1/targets` | GET | Scrape targets |
| `/api/v1/rules` | GET | Recording/alerting rules |
| `/api/v1/alerts` | GET | Active alerts |

#### Key Queries

##### Cardinality Monitoring

```bash
# Current cardinality per pipeline
curl "http://localhost:9090/api/v1/query?query=phoenix_pipeline_output_cardinality_estimate"

# Cardinality trend over time
curl "http://localhost:9090/api/v1/query_range?query=phoenix_pipeline_output_cardinality_estimate&start=2024-01-15T10:00:00Z&end=2024-01-15T11:00:00Z&step=60s"
```

##### Performance Metrics

```bash
# Processing rates
curl "http://localhost:9090/api/v1/query?query=rate(otelcol_processor_batch_batch_send_size_sum[5m])"

# Memory usage
curl "http://localhost:9090/api/v1/query?query=otelcol_process_memory_rss"

# CPU usage
curl "http://localhost:9090/api/v1/query?query=rate(otelcol_process_cpu_seconds_total[5m])"
```

##### Control System Metrics

```bash
# Profile switch history
curl "http://localhost:9090/api/v1/query?query=phoenix_control_profile_switches_total"

# Current profile
curl "http://localhost:9090/api/v1/query?query=phoenix_control_current_profile"

# Time since last switch
curl "http://localhost:9090/api/v1/query?query=time() - phoenix_control_last_switch_timestamp"
```

#### Response Format

```json
{
  "status": "success",
  "data": {
    "resultType": "vector",
    "result": [
      {
        "metric": {
          "__name__": "phoenix_pipeline_output_cardinality_estimate",
          "pipeline": "full_fidelity"
        },
        "value": [1640995200, "15234"]
      }
    ]
  }
}
```

## Grafana API

### Base Endpoints

**Base URL:** `http://localhost:3000`  
**Authentication:** admin/admin (default)

#### Dashboard API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/dashboards/home` | GET | Home dashboard |
| `/api/search` | GET | Search dashboards |
| `/api/dashboards/uid/{uid}` | GET | Get dashboard by UID |
| `/api/datasources` | GET | List data sources |

#### Usage Examples

```bash
# List dashboards
curl -u admin:admin http://localhost:3000/api/search

# Get Phoenix dashboard
curl -u admin:admin "http://localhost:3000/api/dashboards/uid/phoenix-overview"

# Check data sources
curl -u admin:admin http://localhost:3000/api/datasources
```

## Control System API

### Configuration File Interface

The control system uses file-based configuration that can be modified directly or through automated scripts.

#### Control File Location

`configs/control/optimization_mode.yaml`

#### Current State Query

```bash
# Read current control state
cat configs/control/optimization_mode.yaml

# Watch for changes
watch -n 5 cat configs/control/optimization_mode.yaml
```

#### Manual Profile Control

```bash
# Set conservative profile
cat > configs/control/optimization_mode.yaml << EOF
current_mode: "conservative"
last_updated: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
pipeline_enables:
  full_fidelity: true
  optimized: false
  experimental_topk: false
EOF

# Set balanced profile
cat > configs/control/optimization_mode.yaml << EOF
current_mode: "balanced"
last_updated: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
pipeline_enables:
  full_fidelity: true
  optimized: true
  experimental_topk: false
EOF

# Set aggressive profile
cat > configs/control/optimization_mode.yaml << EOF
current_mode: "aggressive"
last_updated: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
pipeline_enables:
  full_fidelity: false
  optimized: true
  experimental_topk: true
EOF
```

## Error Handling

### Standard HTTP Error Responses

All APIs follow standard HTTP status codes:

| Code | Description | Typical Causes |
|------|-------------|----------------|
| 200 | OK | Request successful |
| 400 | Bad Request | Invalid parameters or payload |
| 401 | Unauthorized | Missing or invalid authentication |
| 404 | Not Found | Endpoint or resource doesn't exist |
| 429 | Too Many Requests | Rate limiting applied |
| 500 | Internal Server Error | Service malfunction |
| 503 | Service Unavailable | Service temporarily down |

### Error Response Format

```json
{
  "error": "invalid_request",
  "message": "Required field 'resourceMetrics' is missing",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Rate Limiting

### OTLP Ingestion

- **Default Limit:** 1000 requests/minute per client
- **Burst:** Up to 100 concurrent requests
- **Headers:** `X-RateLimit-Remaining`, `X-RateLimit-Reset`

### Metrics Endpoints

- **Default Limit:** 100 requests/minute per client
- **Prometheus scraping:** Excluded from rate limiting

## Authentication and Security

### Default Configuration

- **Grafana:** admin/admin (change in production)
- **Prometheus:** No authentication (internal access only)
- **Collectors:** No authentication (internal network)

### Production Security

```bash
# Enable authentication in docker-compose.yaml
environment:
  GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_ADMIN_PASSWORD}
  GF_AUTH_BASIC_ENABLED: true

# Use reverse proxy for external access
# Configure TLS certificates
# Set up firewall rules
```

## SDK Integration Examples

### Go SDK

```go
package main

import (
    "context"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
    "go.opentelemetry.io/otel/metric"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func main() {
    exporter, err := otlpmetrichttp.New(
        context.Background(),
        otlpmetrichttp.WithEndpoint("http://localhost:4318"),
        otlpmetrichttp.WithInsecure(),
    )
    if err != nil {
        panic(err)
    }

    provider := sdkmetric.NewMeterProvider(
        sdkmetric.WithReader(
            sdkmetric.NewPeriodicReader(exporter),
        ),
    )

    meter := provider.Meter("example-app")
    counter, _ := meter.Int64Counter("requests_total")
    
    counter.Add(context.Background(), 1)
}
```

### Python SDK

```python
from opentelemetry import metrics
from opentelemetry.exporter.otlp.proto.http.metric_exporter import OTLPMetricExporter
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader

exporter = OTLPMetricExporter(
    endpoint="http://localhost:4318/v1/metrics",
    headers={}
)

reader = PeriodicExportingMetricReader(exporter)
provider = MeterProvider(metric_readers=[reader])
metrics.set_meter_provider(provider)

meter = metrics.get_meter("example-app")
counter = meter.create_counter("requests_total")

counter.add(1, {"endpoint": "/api/users"})
```

### JavaScript SDK

```javascript
const { MeterProvider } = require('@opentelemetry/sdk-metrics');
const { OTLPMetricExporter } = require('@opentelemetry/exporter-otlp-http');
const { PeriodicExportingMetricReader } = require('@opentelemetry/sdk-metrics');

const exporter = new OTLPMetricExporter({
  url: 'http://localhost:4318/v1/metrics',
});

const reader = new PeriodicExportingMetricReader({
  exporter: exporter,
  exportIntervalMillis: 10000,
});

const meterProvider = new MeterProvider({
  readers: [reader],
});

const meter = meterProvider.getMeter('example-app');
const counter = meter.createCounter('requests_total');

counter.add(1, { endpoint: '/api/users' });
```