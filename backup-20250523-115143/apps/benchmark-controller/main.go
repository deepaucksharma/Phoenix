// Phoenix Benchmark Controller - Main Implementation
// This service validates pipeline performance across edge and New Relic

package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "sync"
    "time"

    "github.com/prometheus/client_golang/api"
    v1 "github.com/prometheus/client_golang/api/prometheus/v1"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/push"
    _ "github.com/mattn/go-sqlite3"
)

// Metrics for tracking benchmark results
var (
    ingestLatencyGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "phoenix_benchmark_ingest_latency_seconds",
            Help: "End-to-end ingest latency from edge to New Relic",
        },
        []string{"percentile", "pipeline"},
    )
    
    costPerTSGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "phoenix_benchmark_cost_per_timeseries_usd",
            Help: "Cost per active time series in USD",
        },
        []string{"pipeline", "source"},
    )
    
    entityYieldGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "phoenix_benchmark_entity_yield_ratio",
            Help: "Ratio of entities with metrics vs pushed entities",
        },
        []string{"pipeline"},
    )
    
    featureDriftGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "phoenix_benchmark_feature_drift_percent",
            Help: "Feature coverage drift over time",
        },
        []string{"feature", "pipeline"},
    )
    
    validationErrorsCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "phoenix_benchmark_validation_errors_total",
            Help: "Total validation errors by type",
        },
        []string{"error_type", "pipeline"},
    )
)

func init() {
    prometheus.MustRegister(ingestLatencyGauge)
    prometheus.MustRegister(costPerTSGauge)
    prometheus.MustRegister(entityYieldGauge)
    prometheus.MustRegister(featureDriftGauge)
    prometheus.MustRegister(validationErrorsCounter)
}

// BenchmarkController orchestrates all validation activities
type BenchmarkController struct {
    promClient      v1.API
    db              *sql.DB
    pushgateway     string
    config          *Config
    mu              sync.RWMutex
    lastResults     map[string]interface{}
}

// Config holds all configuration values
type Config struct {
    ValidationInterval   time.Duration
    MaxIngestLatencyP95 time.Duration
    MinCostReduction    float64
    MinEntityYield      float64
    MaxFeatureDrift     float64
    PrometheusURL       string
    PushgatewayURL      string
    DatabasePath        string
}

// ValidationResult contains all benchmark metrics
type ValidationResult struct {
    Timestamp       time.Time
    Pipeline        string
    IngestLatencyP95 float64
    CostPerTS       float64
    EntityYield     float64
    FeatureDrift    map[string]float64
    Passed          bool
    FailureReasons  []string
}

// NewBenchmarkController creates a new controller instance
func NewBenchmarkController(config *Config) (*BenchmarkController, error) {
    // Initialize Prometheus client
    promClient, err := api.NewClient(api.Config{
        Address: config.PrometheusURL,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
    }

    // Initialize database
    db, err := sql.Open("sqlite3", config.DatabasePath)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Create tables if not exists
    if err := createTables(db); err != nil {
        return nil, fmt.Errorf("failed to create tables: %w", err)
    }

    return &BenchmarkController{
        promClient:      v1.NewAPI(promClient),
        db:              db,
        pushgateway:     config.PushgatewayURL,
        config:          config,
        lastResults:     make(map[string]interface{}),
    }, nil
}

// Run starts the continuous validation loop
func (bc *BenchmarkController) Run(ctx context.Context) error {
    ticker := time.NewTicker(bc.config.ValidationInterval)
    defer ticker.Stop()

    log.Println("Phoenix Benchmark Controller started")

    // Run initial validation
    if err := bc.runValidation(ctx); err != nil {
        log.Printf("Initial validation failed: %v", err)
    }

    for {
        select {
        case <-ticker.C:
            if err := bc.runValidation(ctx); err != nil {
                log.Printf("Validation error: %v", err)
                validationErrorsCounter.WithLabelValues("validation_failed", "all").Inc()
            }
        case <-ctx.Done():
            log.Println("Benchmark Controller shutting down")
            return ctx.Err()
        }
    }
}

// runValidation performs a complete validation cycle
func (bc *BenchmarkController) runValidation(ctx context.Context) error {
    pipelines := []string{"full_fidelity", "optimised", "experimental"}
    results := make([]ValidationResult, 0, len(pipelines))

    for _, pipeline := range pipelines {
        result, err := bc.validatePipeline(ctx, pipeline)
        if err != nil {
            log.Printf("Failed to validate pipeline %s: %v", pipeline, err)
            continue
        }
        
        results = append(results, result)
        
        // Store results
        if err := bc.storeResult(result); err != nil {
            log.Printf("Failed to store result: %v", err)
        }
        
        // Push metrics to Pushgateway
        if err := bc.pushMetrics(result); err != nil {
            log.Printf("Failed to push metrics: %v", err)
        }
        
        // Check thresholds and log failures
        if !result.Passed {
            bc.handleValidationFailure(result)
        }
    }

    return nil
}

// validatePipeline validates a single pipeline
func (bc *BenchmarkController) validatePipeline(ctx context.Context, pipeline string) (ValidationResult, error) {
    result := ValidationResult{
        Timestamp:      time.Now(),
        Pipeline:       pipeline,
        FeatureDrift:   make(map[string]float64),
        Passed:         true,
        FailureReasons: []string{},
    }

    // 1. Measure Ingest Latency (simplified - using mock data)
    latency := 15.0 + float64(len(pipeline))*0.5 // Mock latency calculation
    result.IngestLatencyP95 = latency
    
    if latency > bc.config.MaxIngestLatencyP95.Seconds() {
        result.Passed = false
        result.FailureReasons = append(result.FailureReasons, 
            fmt.Sprintf("Ingest latency P95 (%.2fs) exceeds threshold (%.2fs)", 
                latency, bc.config.MaxIngestLatencyP95.Seconds()))
    }

    // 2. Calculate Cost per Time Series
    costPerTS, err := bc.calculateCostPerTS(ctx, pipeline)
    if err != nil {
        return result, fmt.Errorf("failed to calculate cost: %w", err)
    }
    result.CostPerTS = costPerTS

    // Compare with baseline
    baselineCost := 0.001 // $0.001 per TS as baseline
    costReduction := 1 - (costPerTS / baselineCost)
    if costReduction < bc.config.MinCostReduction {
        result.Passed = false
        result.FailureReasons = append(result.FailureReasons,
            fmt.Sprintf("Cost reduction (%.2f%%) below threshold (%.2f%%)",
                costReduction*100, bc.config.MinCostReduction*100))
    }

    // 3. Calculate Entity Yield
    entityYield := 0.95 - float64(len(pipeline))*0.01 // Mock calculation
    result.EntityYield = entityYield

    if entityYield < bc.config.MinEntityYield {
        result.Passed = false
        result.FailureReasons = append(result.FailureReasons,
            fmt.Sprintf("Entity yield (%.2f) below threshold (%.2f)",
                entityYield, bc.config.MinEntityYield))
    }

    // 4. Check Feature Coverage Drift
    features := []string{"cpu_metrics", "memory_metrics", "disk_io", "process_details"}
    for _, feature := range features {
        drift := -0.02 - float64(len(feature))*0.001 // Mock drift
        result.FeatureDrift[feature] = drift
        
        if drift < bc.config.MaxFeatureDrift {
            result.Passed = false
            result.FailureReasons = append(result.FailureReasons,
                fmt.Sprintf("Feature %s drift (%.2f%%) exceeds threshold (%.2f%%)",
                    feature, drift*100, bc.config.MaxFeatureDrift*100))
        }
    }

    return result, nil
}

// calculateCostPerTS determines cost efficiency
func (bc *BenchmarkController) calculateCostPerTS(ctx context.Context, pipeline string) (float64, error) {
    // Query active time series from Prometheus
    query := fmt.Sprintf(`phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{phoenix_pipeline_label="%s"}`, pipeline)
    result, _, err := bc.promClient.Query(ctx, query, time.Now())
    if err != nil {
        return 0, err
    }

    // Parse result and calculate cost
    // This is simplified - real implementation would parse the vector result
    tsCount := 10000.0 // Mock value
    costPerTS := 0.00001 * (1 - float64(len(pipeline))*0.1) // Mock calculation
    
    return costPerTS, nil
}

// pushMetrics sends results to Pushgateway
func (bc *BenchmarkController) pushMetrics(result ValidationResult) error {
    // Update Prometheus metrics
    ingestLatencyGauge.WithLabelValues("p95", result.Pipeline).Set(result.IngestLatencyP95)
    costPerTSGauge.WithLabelValues(result.Pipeline, "calculated").Set(result.CostPerTS)
    entityYieldGauge.WithLabelValues(result.Pipeline).Set(result.EntityYield)
    
    for feature, drift := range result.FeatureDrift {
        featureDriftGauge.WithLabelValues(feature, result.Pipeline).Set(drift)
    }

    // Push to gateway
    pusher := push.New(bc.pushgateway, "phoenix_benchmark")
    pusher.Collector(ingestLatencyGauge)
    pusher.Collector(costPerTSGauge)
    pusher.Collector(entityYieldGauge)
    pusher.Collector(featureDriftGauge)
    
    return pusher.Push()
}

// storeResult persists validation results to database
func (bc *BenchmarkController) storeResult(result ValidationResult) error {
    jsonData, err := json.Marshal(result)
    if err != nil {
        return err
    }

    _, err = bc.db.Exec(`
        INSERT INTO benchmark_results 
        (timestamp, pipeline, data, passed) 
        VALUES (?, ?, ?, ?)
    `, result.Timestamp, result.Pipeline, jsonData, result.Passed)
    
    return err
}

// handleValidationFailure processes failed validations
func (bc *BenchmarkController) handleValidationFailure(result ValidationResult) {
    log.Printf("VALIDATION FAILED for pipeline %s: %v", result.Pipeline, result.FailureReasons)
    
    // Increment error counter for each failure reason
    for _, reason := range result.FailureReasons {
        validationErrorsCounter.WithLabelValues("threshold_exceeded", result.Pipeline).Inc()
    }
}

// createTables creates database schema
func createTables(db *sql.DB) error {
    schema := `
    CREATE TABLE IF NOT EXISTS benchmark_results (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME NOT NULL,
        pipeline TEXT NOT NULL,
        data TEXT NOT NULL,
        passed BOOLEAN NOT NULL
    );
    
    CREATE INDEX IF NOT EXISTS idx_benchmark_timestamp ON benchmark_results(timestamp);
    CREATE INDEX IF NOT EXISTS idx_benchmark_pipeline ON benchmark_results(pipeline);
    `
    
    _, err := db.Exec(schema)
    return err
}

func main() {
    config := &Config{
        ValidationInterval:   getEnvDuration("VALIDATION_INTERVAL", 60*time.Second),
        MaxIngestLatencyP95:  getEnvDuration("MAX_INGEST_LATENCY_P95", 30*time.Second),
        MinCostReduction:     getEnvFloat("MIN_COST_REDUCTION", 0.65),
        MinEntityYield:       getEnvFloat("MIN_ENTITY_YIELD", 0.95),
        MaxFeatureDrift:      getEnvFloat("MAX_FEATURE_DRIFT", -0.05),
        PrometheusURL:        getEnvString("PROMETHEUS_URL", "http://prometheus:9090"),
        PushgatewayURL:       getEnvString("PUSHGATEWAY_URL", "http://pushgateway:9091"),
        DatabasePath:         getEnvString("DATABASE_PATH", "/data/benchmark.db"),
    }

    controller, err := NewBenchmarkController(config)
    if err != nil {
        log.Fatal(err)
    }

    // Start HTTP server for health checks
    go func() {
        http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("OK"))
        })
        log.Fatal(http.ListenAndServe(":8080", nil))
    }()

    ctx := context.Background()
    if err := controller.Run(ctx); err != nil {
        log.Fatal(err)
    }
}

// Helper functions
func getEnvString(key, defaultValue string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
    if v := os.Getenv(key); v != "" {
        var f float64
        fmt.Sscanf(v, "%f", &f)
        return f
    }
    return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
    if v := os.Getenv(key); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            return d
        }
    }
    return defaultValue
}