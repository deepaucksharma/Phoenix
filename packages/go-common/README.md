# Phoenix Go Common Package

Shared Go utilities and types for Phoenix microservices.

## Installation

From any Phoenix service:
```bash
go get github.com/phoenix-vnext/phoenix/packages/go-common
```

## Usage Examples

### 1. Control Types
```go
import "github.com/phoenix-vnext/phoenix/packages/go-common/control"

// Create a control signal
signal := control.ControlSignal{
    Mode:          control.ModeAggressive,
    Version:       "2.0.0",
    Timestamp:     time.Now(),
    CorrelationID: uuid.New().String(),
    Source:        "control-actuator",
    Reason:        "cardinality threshold exceeded",
}
```

### 2. Observability
```go
import "github.com/phoenix-vnext/phoenix/packages/go-common/observability"

// Initialize metrics server
metrics := observability.NewMetricsServer("my-service", "8080")
metrics.RegisterServiceMetrics("my-service")

// Use standardized health check
health := observability.NewHealthChecker()
http.HandleFunc("/health", health.Handler)
```

### 3. Configuration
```go
import "github.com/phoenix-vnext/phoenix/packages/go-common/config"

// Load configuration with validation
cfg := config.New()
cfg.LoadFile("config.yaml")
cfg.ValidateSchema("service-config.schema.yaml")

// Get values with defaults
port := cfg.GetString("server.port", "8080")
timeout := cfg.GetDuration("server.timeout", 30*time.Second)
```

### 4. Service Discovery
```go
import "github.com/phoenix-vnext/phoenix/packages/go-common/discovery"

// Get service endpoints
endpoints := discovery.LoadEndpoints()
client := NewPrometheusClient(endpoints.Prometheus)
```

### 5. Testing Utilities
```go
import "github.com/phoenix-vnext/phoenix/packages/go-common/testing"

// Use test helpers
func TestMyHandler(t *testing.T) {
    recorder := testing.NewTestRecorder()
    req := testing.NewTestRequest("GET", "/api/metrics", nil)
    
    handler.ServeHTTP(recorder, req)
    
    testing.AssertStatus(t, recorder, http.StatusOK)
    testing.AssertJSON(t, recorder, expectedResponse)
}
```

## Package Structure

```
go-common/
├── control/         # Control signal types and utilities
├── observability/   # Metrics, logging, tracing
├── config/         # Configuration management
├── discovery/      # Service discovery
├── health/         # Health check standards
├── testing/        # Test utilities and mocks
└── utils/          # General utilities
```

## Development

### Running Tests
```bash
cd packages/go-common
go test ./...
```

### Adding New Packages
1. Create new directory under `go-common/`
2. Add package with clear interfaces
3. Include comprehensive tests
4. Update this README with usage examples

## Standards

- All packages must have >80% test coverage
- Interfaces preferred over concrete types
- No external dependencies without approval
- Follow Go best practices and idioms