# Phoenix Service Refactoring Example

This document demonstrates how to refactor existing services to use the shared `go-common` package.

## Before: Control Actuator (Current Implementation)

```go
// apps/control-actuator-go/main.go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
    
    "github.com/prometheus/client_golang/api"
    v1 "github.com/prometheus/client_golang/api/prometheus/v1"
    "gopkg.in/yaml.v3"
)

// Duplicated types
type OptimizationMode string

const (
    Conservative OptimizationMode = "conservative"
    Balanced     OptimizationMode = "balanced"
    Aggressive   OptimizationMode = "aggressive"
)

// Duplicated helpers
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

// Manual health check
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
        "version": "1.0.0",
    })
})
```

## After: Using Shared Package

```go
// services/control-actuator/main.go
package main

import (
    "context"
    "time"
    
    "github.com/phoenix-vnext/phoenix/packages/go-common/config"
    "github.com/phoenix-vnext/phoenix/packages/go-common/control"
    "github.com/phoenix-vnext/phoenix/packages/go-common/discovery"
    "github.com/phoenix-vnext/phoenix/packages/go-common/observability"
)

func main() {
    // Initialize configuration
    cfg := config.New()
    cfg.LoadEnv()
    cfg.ValidateSchema("control-actuator.schema.yaml")
    
    // Setup observability
    metrics := observability.NewMetricsServer("control-actuator", cfg.GetString("metrics.port"))
    metrics.RegisterServiceMetrics("control-actuator")
    health := observability.NewHealthChecker("control-actuator", "1.0.0")
    
    // Service discovery
    endpoints := discovery.LoadEndpoints()
    
    // Initialize control loop with shared types
    loop := NewControlLoop(ControlLoopConfig{
        PrometheusURL:    endpoints.Prometheus,
        TargetTS:        cfg.GetFloat64("control.target_ts"),
        UpdateInterval:  cfg.GetDuration("control.interval"),
        PIDConfig: control.PIDConfig{
            Kp: cfg.GetFloat64("pid.kp"),
            Ki: cfg.GetFloat64("pid.ki"),
            Kd: cfg.GetFloat64("pid.kd"),
        },
    })
    
    // Setup HTTP server with standard handlers
    mux := http.NewServeMux()
    mux.Handle("/health", health.Handler())
    mux.Handle("/metrics", metrics.Handler())
    mux.HandleFunc("/mode", loop.HandleModeUpdate)
    mux.HandleFunc("/anomaly", loop.HandleAnomaly)
    
    // Add standard middleware
    handler := observability.InstrumentHandler(mux)
    
    server := &http.Server{
        Addr:    ":" + cfg.GetString("server.port"),
        Handler: handler,
    }
    
    // Start control loop
    go loop.Run(context.Background())
    
    // Start server
    log.Fatal(server.ListenAndServe())
}

// Control loop now uses shared types
type ControlLoop struct {
    config    ControlLoopConfig
    state     *control.ControllerState
    client    *observability.PrometheusClient
}

func (cl *ControlLoop) evaluate() error {
    // Query metrics using shared client
    cardinality, err := cl.client.QueryScalar(
        "phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate",
        map[string]string{"pipeline": "optimized"},
    )
    if err != nil {
        return err
    }
    
    // Use shared PID implementation
    output := cl.state.CalculatePID(cardinality, cl.config.TargetTS)
    
    // Create control signal with shared types
    if newMode := cl.determineMode(output); newMode != cl.state.CurrentMode {
        signal := control.ControlSignal{
            Mode:          newMode,
            Version:       "2.0.0",
            Timestamp:     time.Now(),
            CorrelationID: observability.GenerateTraceID(),
            Source:        "control-actuator",
            Reason:        fmt.Sprintf("PID output: %.2f", output),
        }
        
        // Emit through standardized event system
        if err := cl.publishControlSignal(signal); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Benefits of Refactoring

### 1. **Reduced Code Size**
- Before: ~400 lines
- After: ~200 lines
- Reduction: 50%

### 2. **Standardized Patterns**
- Consistent health checks across all services
- Unified metrics collection
- Standard configuration management

### 3. **Improved Testing**
```go
// Before: Manual mocking
type mockPrometheusAPI struct {
    v1.API
    queryFunc func(ctx context.Context, query string, ts time.Time) (model.Value, v1.Warnings, error)
}

// After: Use test utilities
func TestControlLoop(t *testing.T) {
    // Use shared test helpers
    mockClient := testing.NewMockPrometheusClient()
    mockClient.SetQueryResponse("phoenix_observer_kpi", 25000.0)
    
    loop := NewControlLoop(testing.TestConfig())
    loop.client = mockClient
    
    err := loop.evaluate()
    testing.AssertNoError(t, err)
    testing.AssertMetricValue(t, "control_mode", "aggressive")
}
```

### 4. **Type Safety**
```go
// Before: String-based modes prone to typos
mode := "agressive" // Typo!

// After: Type-safe constants
mode := control.ModeAggressive // Compiler-checked
```

### 5. **Service Communication**
```go
// Before: Manual HTTP calls with no contract
resp, err := http.Post("http://anomaly-detector:8082/webhook", 
    "application/json", bytes.NewReader(data))

// After: Typed client with contract
client := discovery.NewAnomalyDetectorClient(endpoints.AnomalyDetector)
err := client.RegisterWebhook(control.WebhookConfig{
    URL: "http://control-actuator:8081/anomaly",
    Events: []string{"cardinality_explosion", "memory_leak"},
})
```

## Migration Steps

1. **Add Dependency**
   ```bash
   cd services/control-actuator
   go get github.com/phoenix-vnext/phoenix/packages/go-common
   ```

2. **Replace Types**
   - Search for local type definitions
   - Replace with shared types
   - Update imports

3. **Use Shared Utilities**
   - Replace `getEnv()` with `config.GetString()`
   - Replace manual health checks with `health.Handler()`
   - Use `observability.NewPrometheusClient()`

4. **Update Tests**
   - Use shared test utilities
   - Replace custom mocks with provided ones
   - Leverage test helpers

5. **Validate Behavior**
   - Run existing tests
   - Verify metrics are collected
   - Check health endpoints
   - Test service communication

## Common Patterns

### Configuration
```go
// Standardized configuration loading
cfg := config.New()
cfg.LoadFile("config.yaml")
cfg.LoadEnv()
cfg.SetDefaults(map[string]interface{}{
    "server.port": "8080",
    "metrics.port": "9090",
})
```

### Error Handling
```go
// Use shared error types
if err != nil {
    return observability.WrapError(err, "failed to query metrics",
        "service", "control-actuator",
        "operation", "evaluate",
    )
}
```

### Logging
```go
// Structured logging with context
logger := observability.NewLogger("control-actuator")
logger.WithFields(observability.Fields{
    "mode": signal.Mode,
    "correlation_id": signal.CorrelationID,
}).Info("Control mode changed")
```

### Metrics
```go
// Pre-defined metrics
metrics.RecordModeChange(oldMode, newMode)
metrics.RecordPIDOutput(output)
metrics.RecordEvaluationDuration(time.Since(start))
```

## Conclusion

Refactoring services to use the shared `go-common` package significantly reduces code duplication, improves maintainability, and ensures consistency across the Phoenix ecosystem. The migration can be done incrementally, service by service, without disrupting the overall system operation.