# Phoenix Analytics Service

Advanced analytics and visualization service for Phoenix-vNext metrics pipeline.

## Features

- **Trend Analysis**: Detect trends, anomalies, and seasonality in metrics
- **Correlation Analysis**: Find relationships between different metrics
- **Advanced Visualizations**: Generate time series charts, heatmaps, scatter plots, and histograms
- **Real-time Analysis**: Query live data from Prometheus

## API Endpoints

### Trend Analysis
```bash
POST /api/v1/trends/analyze
{
  "metric": "phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate",
  "duration": "1h",
  "window_size": 100
}
```

### Correlation Analysis
```bash
POST /api/v1/correlations/analyze
{
  "metrics": [
    "otelcol_processor_accepted_metric_points",
    "process_cpu_seconds_total",
    "process_resident_memory_bytes"
  ],
  "duration": "1h",
  "min_samples": 30
}
```

### Generate Visualization
```bash
POST /api/v1/visualizations/generate
{
  "type": "timeseries",
  "query": "rate(otelcol_processor_accepted_metric_points[5m])",
  "duration": "1h",
  "options": {
    "title": "Metric Processing Rate"
  }
}
```

## Running Locally

```bash
# Build
go build -o analytics cmd/main.go

# Run
PROMETHEUS_ADDR=http://localhost:9090 ./analytics
```

## Docker

```bash
# Build image
docker build -t phoenix-analytics .

# Run container
docker run -p 8080:8080 -e PROMETHEUS_ADDR=http://prometheus:9090 phoenix-analytics
```

## Environment Variables

- `ANALYTICS_PORT`: Service port (default: 8080)
- `PROMETHEUS_ADDR`: Prometheus server address (default: http://prometheus:9090)