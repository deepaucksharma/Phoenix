# Phoenix Platform API Reference

## Overview

The Phoenix Platform API provides programmatic access to experiment management, pipeline configuration, and metrics analysis. The API supports both REST (via gRPC-gateway) and native gRPC protocols.

## Base URL

```
https://api.phoenix.example.com/v1
```

## Authentication

All API requests require JWT authentication:

```bash
curl -H "Authorization: Bearer <JWT_TOKEN>" \
  https://api.phoenix.example.com/v1/experiments
```

### Obtaining a Token

```bash
POST /v1/auth/login
Content-Type: application/json

{
  "username": "user@example.com",
  "password": "password"
}

Response:
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

## Experiments API

### List Experiments

```http
GET /v1/experiments
```

Query Parameters:
- `status` (optional): Filter by status (running, completed, failed)
- `limit` (optional): Number of results (default: 20, max: 100)
- `offset` (optional): Pagination offset

Response:
```json
{
  "experiments": [
    {
      "id": "exp-123",
      "name": "webserver-optimization",
      "status": "running",
      "baseline_pipeline": "process-baseline-v1",
      "candidate_pipeline": "process-priority-filter-v1",
      "created_at": "2024-11-24T10:00:00Z",
      "started_at": "2024-11-24T10:05:00Z",
      "target_nodes": {
        "selector": {
          "app": "webserver"
        }
      }
    }
  ],
  "total": 42,
  "has_more": true
}
```

### Create Experiment

```http
POST /v1/experiments
Content-Type: application/json
```

Request Body:
```json
{
  "name": "database-optimization-test",
  "description": "Reduce database server metrics by 50%",
  "baseline_pipeline": "process-baseline-v1",
  "candidate_pipeline": "process-topk-v1",
  "duration": "24h",
  "target_nodes": {
    "selector": {
      "role": "database",
      "environment": "staging"
    }
  },
  "critical_processes": [
    "postgres",
    "pgbouncer",
    "patroni"
  ],
  "config_overrides": {
    "topk_count": 30,
    "memory_limit_mib": 256
  }
}
```

Response:
```json
{
  "id": "exp-456",
  "name": "database-optimization-test",
  "status": "pending",
  "created_at": "2024-11-24T12:00:00Z",
  "deployment_status": {
    "phase": "configuring",
    "message": "Generating pipeline configurations"
  }
}
```

### Get Experiment Details

```http
GET /v1/experiments/{id}
```

Response:
```json
{
  "id": "exp-123",
  "name": "webserver-optimization",
  "status": "running",
  "baseline_pipeline": "process-baseline-v1",
  "candidate_pipeline": "process-priority-filter-v1",
  "created_at": "2024-11-24T10:00:00Z",
  "started_at": "2024-11-24T10:05:00Z",
  "metrics": {
    "baseline": {
      "cardinality": 50000,
      "ingestion_rate_dpm": 1000000,
      "critical_processes_retained": 25,
      "collector_cpu_cores": 0.5,
      "collector_memory_mib": 256
    },
    "candidate": {
      "cardinality": 12500,
      "ingestion_rate_dpm": 250000,
      "critical_processes_retained": 25,
      "collector_cpu_cores": 0.3,
      "collector_memory_mib": 200
    },
    "reduction_percentage": 75,
    "estimated_monthly_savings_usd": 875
  }
}
```

### Update Experiment

```http
PATCH /v1/experiments/{id}
Content-Type: application/json
```

Request Body:
```json
{
  "duration": "48h",
  "description": "Extended test for weekend traffic"
}
```

### Stop Experiment

```http
POST /v1/experiments/{id}/stop
```

Response:
```json
{
  "id": "exp-123",
  "status": "stopping",
  "message": "Experiment stop initiated"
}
```

### Promote Experiment Variant

```http
POST /v1/experiments/{id}/promote
Content-Type: application/json
```

Request Body:
```json
{
  "variant": "candidate",
  "rollout_strategy": "immediate"
}
```

## Pipelines API

### List Pipeline Templates

```http
GET /v1/pipelines
```

Response:
```json
{
  "pipelines": [
    {
      "name": "process-baseline-v1",
      "version": "1.0.0",
      "description": "No optimization, full process visibility",
      "type": "baseline",
      "expected_reduction": 0
    },
    {
      "name": "process-priority-filter-v1",
      "version": "1.0.0",
      "description": "Filter by process priority",
      "type": "optimization",
      "expected_reduction": 60,
      "configurable_parameters": [
        {
          "name": "critical_processes",
          "type": "array[string]",
          "required": true
        }
      ]
    }
  ]
}
```

### Get Pipeline Configuration

```http
GET /v1/pipelines/{name}
```

Response:
```json
{
  "name": "process-priority-filter-v1",
  "version": "1.0.0",
  "description": "Filter by process priority",
  "configuration": {
    "receivers": {
      "hostmetrics": {
        "collection_interval": "10s",
        "scrapers": {
          "process": {
            "include": [".*"],
            "metrics": [
              "process.cpu.time",
              "process.memory.physical",
              "process.memory.virtual"
            ]
          }
        }
      }
    },
    "processors": {
      "memory_limiter": {
        "limit_mib": 512,
        "spike_limit_mib": 128
      },
      "transform/classify": {
        "metric_statements": [
          {
            "context": "resource",
            "statements": [
              "set(attributes[\"process.priority\"], \"critical\") where attributes[\"process.name\"] =~ \"^(nginx|mysql|redis)\""
            ]
          }
        ]
      }
    }
  }
}
```

### Validate Pipeline Configuration

```http
POST /v1/pipelines/validate
Content-Type: application/json
```

Request Body:
```json
{
  "configuration": {
    "receivers": {},
    "processors": {},
    "exporters": {},
    "service": {}
  }
}
```

## Metrics API

### Get Experiment Metrics

```http
GET /v1/experiments/{id}/metrics
```

Query Parameters:
- `start` (optional): Start time (RFC3339)
- `end` (optional): End time (RFC3339)
- `resolution` (optional): Data resolution (1m, 5m, 1h)

Response:
```json
{
  "experiment_id": "exp-123",
  "time_range": {
    "start": "2024-11-24T10:00:00Z",
    "end": "2024-11-24T12:00:00Z"
  },
  "series": {
    "baseline_cardinality": [
      {"timestamp": "2024-11-24T10:00:00Z", "value": 50000},
      {"timestamp": "2024-11-24T10:05:00Z", "value": 50100}
    ],
    "candidate_cardinality": [
      {"timestamp": "2024-11-24T10:00:00Z", "value": 12500},
      {"timestamp": "2024-11-24T10:05:00Z", "value": 12600}
    ]
  }
}
```

### Get Cost Analysis

```http
GET /v1/experiments/{id}/cost-analysis
```

Response:
```json
{
  "experiment_id": "exp-123",
  "analysis": {
    "baseline_cost": {
      "hourly_usd": 1.73,
      "daily_usd": 41.52,
      "monthly_usd": 1250.00
    },
    "optimized_cost": {
      "hourly_usd": 0.52,
      "daily_usd": 12.48,
      "monthly_usd": 375.00
    },
    "savings": {
      "hourly_usd": 1.21,
      "daily_usd": 29.04,
      "monthly_usd": 875.00,
      "percentage": 70
    }
  }
}
```

## Load Simulation API

### Create Load Simulation

```http
POST /v1/load-simulations
Content-Type: application/json
```

Request Body:
```json
{
  "name": "high-cardinality-test",
  "profile": "high-cardinality",
  "target_nodes": {
    "selector": {
      "test": "load-sim"
    }
  },
  "duration": "1h",
  "parameters": {
    "process_count": 2000,
    "churn_rate": 10
  }
}
```

### List Load Profiles

```http
GET /v1/load-simulations/profiles
```

Response:
```json
{
  "profiles": [
    {
      "name": "realistic",
      "description": "Simulates typical production workload",
      "default_process_count": 100,
      "default_churn_rate": 1
    },
    {
      "name": "high-cardinality",
      "description": "Many unique processes",
      "default_process_count": 2000,
      "default_churn_rate": 5
    }
  ]
}
```

## WebSocket API

### Real-time Experiment Updates

```javascript
const ws = new WebSocket('wss://api.phoenix.example.com/v1/ws');

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'subscribe',
    experiment_id: 'exp-123'
  }));
};

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  // Handle real-time metrics updates
};
```

Message Types:
- `metrics_update`: Real-time metrics
- `status_change`: Experiment status changes
- `alert`: Important notifications

## Error Responses

All error responses follow this format:

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "Pipeline name is required",
    "details": {
      "field": "baseline_pipeline",
      "reason": "missing_required_field"
    }
  }
}
```

Common Error Codes:
- `INVALID_ARGUMENT`: Invalid request parameters
- `NOT_FOUND`: Resource not found
- `ALREADY_EXISTS`: Resource already exists
- `PERMISSION_DENIED`: Insufficient permissions
- `INTERNAL`: Internal server error

## Rate Limiting

API requests are rate limited:
- Authenticated: 1000 requests/hour
- Unauthenticated: 100 requests/hour

Rate limit headers:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1701234567
```

## SDK Examples

### Go Client

```go
import "github.com/phoenix/platform/sdk/go/phoenix"

client := phoenix.NewClient("https://api.phoenix.example.com")
client.SetToken("your-jwt-token")

experiment, err := client.CreateExperiment(&phoenix.ExperimentRequest{
    Name: "test-experiment",
    BaselinePipeline: "process-baseline-v1",
    CandidatePipeline: "process-topk-v1",
})
```

### Python Client

```python
from phoenix_sdk import PhoenixClient

client = PhoenixClient(
    base_url="https://api.phoenix.example.com",
    token="your-jwt-token"
)

experiment = client.create_experiment(
    name="test-experiment",
    baseline_pipeline="process-baseline-v1",
    candidate_pipeline="process-topk-v1"
)
```

### CLI Usage

```bash
# Configure CLI
phoenix config set api.url https://api.phoenix.example.com
phoenix auth login

# Create experiment
phoenix experiment create \
  --name "cli-test" \
  --baseline "process-baseline-v1" \
  --candidate "process-priority-filter-v1" \
  --duration "24h"

# Check status
phoenix experiment status exp-123
```