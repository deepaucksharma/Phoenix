package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type BenchmarkController struct {
	prometheusAPI v1.API
	scenarios     []BenchmarkScenario
	results       []BenchmarkResult
	mu            sync.Mutex
}

type BenchmarkScenario struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	LoadProfile       LoadProfile            `json:"load_profile"`
	Duration          time.Duration          `json:"duration"`
	WarmupDuration    time.Duration          `json:"warmup_duration"`
	ExpectedBehavior  map[string]interface{} `json:"expected_behavior"`
}

type LoadProfile struct {
	Type              string  `json:"type"` // constant, ramp, spike, wave
	BaseLoad          int     `json:"base_load"`
	PeakLoad          int     `json:"peak_load"`
	RampDuration      string  `json:"ramp_duration,omitempty"`
	MetricsPerHost    int     `json:"metrics_per_host"`
	HostCount         int     `json:"host_count"`
	CardinalityFactor float64 `json:"cardinality_factor"`
}

type BenchmarkResult struct {
	Scenario          string                 `json:"scenario"`
	StartTime         time.Time              `json:"start_time"`
	EndTime           time.Time              `json:"end_time"`
	Metrics           map[string]float64     `json:"metrics"`
	ControlBehavior   []ControlTransition    `json:"control_behavior"`
	ResourceUsage     ResourceMetrics        `json:"resource_usage"`
	Passed            bool                   `json:"passed"`
	FailureReasons    []string               `json:"failure_reasons,omitempty"`
}

type ControlTransition struct {
	Timestamp time.Time `json:"timestamp"`
	FromMode  string    `json:"from_mode"`
	ToMode    string    `json:"to_mode"`
	Reason    string    `json:"reason"`
}

type ResourceMetrics struct {
	AvgCPU      float64 `json:"avg_cpu_percent"`
	MaxCPU      float64 `json:"max_cpu_percent"`
	AvgMemory   float64 `json:"avg_memory_mb"`
	MaxMemory   float64 `json:"max_memory_mb"`
	P99Latency  float64 `json:"p99_latency_ms"`
}

func NewBenchmarkController() (*BenchmarkController, error) {
	client, err := api.NewClient(api.Config{
		Address: getEnv("PROMETHEUS_URL", "http://prometheus:9090"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	bc := &BenchmarkController{
		prometheusAPI: v1.NewAPI(client),
	}

	// Initialize benchmark scenarios
	bc.initializeScenarios()

	return bc, nil
}

func (bc *BenchmarkController) initializeScenarios() {
	bc.scenarios = []BenchmarkScenario{
		{
			Name:        "baseline_steady_state",
			Description: "Validate system behavior under steady state load",
			LoadProfile: LoadProfile{
				Type:              "constant",
				BaseLoad:          100,
				MetricsPerHost:    50,
				HostCount:         10,
				CardinalityFactor: 1.0,
			},
			Duration:       10 * time.Minute,
			WarmupDuration: 2 * time.Minute,
			ExpectedBehavior: map[string]interface{}{
				"optimization_mode":        "balanced",
				"cardinality_reduction":    15.0,
				"signal_preservation":      0.98,
				"max_memory_usage_mb":      512,
			},
		},
		{
			Name:        "cardinality_spike",
			Description: "Test control loop response to sudden cardinality increase",
			LoadProfile: LoadProfile{
				Type:              "spike",
				BaseLoad:          100,
				PeakLoad:          500,
				RampDuration:      "30s",
				MetricsPerHost:    100,
				HostCount:         20,
				CardinalityFactor: 2.5,
			},
			Duration:       15 * time.Minute,
			WarmupDuration: 2 * time.Minute,
			ExpectedBehavior: map[string]interface{}{
				"mode_transition_time_sec": 120,
				"final_mode":               "aggressive",
				"cardinality_reduction":    40.0,
				"signal_preservation":      0.90,
			},
		},
		{
			Name:        "gradual_growth",
			Description: "Validate smooth transitions during gradual load increase",
			LoadProfile: LoadProfile{
				Type:              "ramp",
				BaseLoad:          50,
				PeakLoad:          300,
				RampDuration:      "10m",
				MetricsPerHost:    75,
				HostCount:         15,
				CardinalityFactor: 1.5,
			},
			Duration:       20 * time.Minute,
			WarmupDuration: 2 * time.Minute,
			ExpectedBehavior: map[string]interface{}{
				"max_transitions":       3,
				"stability_score":       0.8,
				"resource_efficiency":   0.7,
			},
		},
		{
			Name:        "wave_pattern",
			Description: "Test hysteresis under oscillating load",
			LoadProfile: LoadProfile{
				Type:              "wave",
				BaseLoad:          100,
				PeakLoad:          250,
				MetricsPerHost:    60,
				HostCount:         12,
				CardinalityFactor: 1.2,
			},
			Duration:       30 * time.Minute,
			WarmupDuration: 5 * time.Minute,
			ExpectedBehavior: map[string]interface{}{
				"max_transitions":       5,
				"mode_stability":        true,
				"avg_response_time_ms":  50,
			},
		},
	}
}

func (bc *BenchmarkController) RunBenchmark(ctx context.Context, scenarioName string) (*BenchmarkResult, error) {
	var scenario *BenchmarkScenario
	for _, s := range bc.scenarios {
		if s.Name == scenarioName {
			scenario = &s
			break
		}
	}

	if scenario == nil {
		return nil, fmt.Errorf("scenario not found: %s", scenarioName)
	}

	log.Printf("Starting benchmark: %s", scenario.Name)

	result := &BenchmarkResult{
		Scenario:  scenario.Name,
		StartTime: time.Now(),
		Metrics:   make(map[string]float64),
	}

	// Apply load profile
	if err := bc.applyLoadProfile(ctx, scenario.LoadProfile); err != nil {
		return nil, fmt.Errorf("failed to apply load profile: %w", err)
	}

	// Warmup period
	log.Printf("Warmup period: %v", scenario.WarmupDuration)
	select {
	case <-time.After(scenario.WarmupDuration):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Start monitoring
	monitorCtx, cancel := context.WithTimeout(ctx, scenario.Duration)
	defer cancel()

	metricsCollector := bc.startMetricsCollection(monitorCtx, result)
	
	// Wait for benchmark duration
	<-monitorCtx.Done()
	
	// Stop monitoring and collect final metrics
	metricsCollector.Wait()
	
	result.EndTime = time.Now()
	
	// Validate results against expected behavior
	bc.validateResults(result, scenario.ExpectedBehavior)
	
	// Store result
	bc.mu.Lock()
	bc.results = append(bc.results, *result)
	bc.mu.Unlock()
	
	log.Printf("Benchmark completed: %s (passed: %v)", scenario.Name, result.Passed)
	
	return result, nil
}

func (bc *BenchmarkController) applyLoadProfile(ctx context.Context, profile LoadProfile) error {
	// Send configuration to synthetic generator
	config := map[string]interface{}{
		"load_type":          profile.Type,
		"base_load":          profile.BaseLoad,
		"peak_load":          profile.PeakLoad,
		"metrics_per_host":   profile.MetricsPerHost,
		"host_count":         profile.HostCount,
		"cardinality_factor": profile.CardinalityFactor,
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		return err
	}

	generatorURL := getEnv("SYNTHETIC_GENERATOR_URL", "http://synthetic-metrics-generator:8080")
	resp, err := http.Post(generatorURL+"/configure", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("generator configuration failed: %s", resp.Status)
	}

	return nil
}

func (bc *BenchmarkController) startMetricsCollection(ctx context.Context, result *BenchmarkResult) *sync.WaitGroup {
	var wg sync.WaitGroup
	
	// Collect metrics every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer ticker.Stop()
		
		var (
			cpuSamples    []float64
			memorySamples []float64
			transitions   []ControlTransition
			lastMode      string
		)
		
		for {
			select {
			case <-ctx.Done():
				// Calculate final metrics
				result.ResourceUsage = bc.calculateResourceMetrics(cpuSamples, memorySamples)
				result.ControlBehavior = transitions
				return
			case <-ticker.C:
				// Collect current metrics
				metrics, err := bc.collectCurrentMetrics()
				if err != nil {
					log.Printf("Error collecting metrics: %v", err)
					continue
				}
				
				// Track resource usage
				if cpu, ok := metrics["cpu_usage"].(float64); ok {
					cpuSamples = append(cpuSamples, cpu)
				}
				if mem, ok := metrics["memory_usage"].(float64); ok {
					memorySamples = append(memorySamples, mem)
				}
				
				// Track control transitions
				if mode, ok := metrics["optimization_mode"].(string); ok {
					if lastMode != "" && mode != lastMode {
						transitions = append(transitions, ControlTransition{
							Timestamp: time.Now(),
							FromMode:  lastMode,
							ToMode:    mode,
							Reason:    "cardinality_threshold",
						})
					}
					lastMode = mode
				}
				
				// Update result metrics
				for k, v := range metrics {
					if fval, ok := v.(float64); ok {
						result.Metrics[k] = fval
					}
				}
			}
		}
	}()
	
	return &wg
}

func (bc *BenchmarkController) collectCurrentMetrics() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics := make(map[string]interface{})

	// Query various metrics
	queries := map[string]string{
		"cardinality_optimized":     `phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{pipeline="optimized"}`,
		"cardinality_full":          `phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{pipeline="full_fidelity"}`,
		"signal_preservation":       `phoenix:signal_preservation_score`,
		"cpu_usage":                 `phoenix:cpu_utilization_percentage`,
		"memory_usage":              `phoenix:memory_utilization_percentage`,
		"pipeline_latency_p99":      `phoenix:pipeline_latency_p99`,
		"control_stability_score":   `phoenix:control_stability_score`,
	}

	for name, query := range queries {
		result, _, err := bc.prometheusAPI.Query(ctx, query, time.Now())
		if err != nil {
			return nil, fmt.Errorf("query %s failed: %w", name, err)
		}

		if vector, ok := result.(model.Vector); ok && len(vector) > 0 {
			metrics[name] = float64(vector[0].Value)
		}
	}

	// Get current optimization mode
	modeResult, _, err := bc.prometheusAPI.Query(ctx, `phoenix_control_optimization_mode`, time.Now())
	if err == nil {
		if vector, ok := modeResult.(model.Vector); ok && len(vector) > 0 {
			metrics["optimization_mode"] = string(vector[0].Metric["mode"])
		}
	}

	// Calculate derived metrics
	if cardOpt, ok := metrics["cardinality_optimized"].(float64); ok {
		if cardFull, ok := metrics["cardinality_full"].(float64); ok && cardFull > 0 {
			metrics["cardinality_reduction"] = 100 * (1 - cardOpt/cardFull)
		}
	}

	return metrics, nil
}

func (bc *BenchmarkController) calculateResourceMetrics(cpuSamples, memorySamples []float64) ResourceMetrics {
	rm := ResourceMetrics{}
	
	if len(cpuSamples) > 0 {
		rm.AvgCPU = average(cpuSamples)
		rm.MaxCPU = max(cpuSamples)
	}
	
	if len(memorySamples) > 0 {
		rm.AvgMemory = average(memorySamples)
		rm.MaxMemory = max(memorySamples)
	}
	
	return rm
}

func (bc *BenchmarkController) validateResults(result *BenchmarkResult, expected map[string]interface{}) {
	result.Passed = true
	result.FailureReasons = []string{}

	for key, expectedValue := range expected {
		actualValue, exists := result.Metrics[key]
		if !exists {
			// Check in other fields
			switch key {
			case "max_transitions":
				actualValue = float64(len(result.ControlBehavior))
			case "optimization_mode":
				if len(result.ControlBehavior) > 0 {
					lastTransition := result.ControlBehavior[len(result.ControlBehavior)-1]
					if lastTransition.ToMode != expectedValue.(string) {
						result.Passed = false
						result.FailureReasons = append(result.FailureReasons,
							fmt.Sprintf("Expected final mode %s, got %s", expectedValue, lastTransition.ToMode))
					}
				}
				continue
			default:
				result.Passed = false
				result.FailureReasons = append(result.FailureReasons,
					fmt.Sprintf("Metric %s not collected", key))
				continue
			}
		}

		// Validate based on type
		switch expected := expectedValue.(type) {
		case float64:
			if actualValue > expected*1.1 || actualValue < expected*0.9 {
				result.Passed = false
				result.FailureReasons = append(result.FailureReasons,
					fmt.Sprintf("%s: expected ~%.2f, got %.2f", key, expected, actualValue))
			}
		case bool:
			if expected && actualValue < 0.5 {
				result.Passed = false
				result.FailureReasons = append(result.FailureReasons,
					fmt.Sprintf("%s: expected true, got false", key))
			}
		}
	}
}

func (bc *BenchmarkController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/benchmark/run":
		bc.handleRunBenchmark(w, r)
	case "/benchmark/results":
		bc.handleGetResults(w, r)
	case "/benchmark/scenarios":
		bc.handleListScenarios(w, r)
	case "/health":
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	default:
		http.NotFound(w, r)
	}
}

func (bc *BenchmarkController) handleRunBenchmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Scenario string `json:"scenario"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Run benchmark asynchronously
	go func() {
		ctx := context.Background()
		result, err := bc.RunBenchmark(ctx, req.Scenario)
		if err != nil {
			log.Printf("Benchmark failed: %v", err)
		} else {
			log.Printf("Benchmark completed: %+v", result)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "started",
		"scenario": req.Scenario,
	})
}

func (bc *BenchmarkController) handleGetResults(w http.ResponseWriter, r *http.Request) {
	bc.mu.Lock()
	results := bc.results
	bc.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (bc *BenchmarkController) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	scenarios := make([]map[string]interface{}, len(bc.scenarios))
	for i, s := range bc.scenarios {
		scenarios[i] = map[string]interface{}{
			"name":        s.Name,
			"description": s.Description,
			"duration":    s.Duration.String(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenarios)
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func main() {
	log.Println("Starting Phoenix Benchmark Controller")

	controller, err := NewBenchmarkController()
	if err != nil {
		log.Fatalf("Failed to initialize benchmark controller: %v", err)
	}

	port := getEnv("PORT", "8080")
	log.Printf("Server listening on port %s", port)
	
	if err := http.ListenAndServe(":"+port, controller); err != nil {
		log.Fatal(err)
	}
}