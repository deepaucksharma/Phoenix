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
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v3"
)

type OptimizationMode string

const (
	Conservative OptimizationMode = "conservative"
	Balanced     OptimizationMode = "balanced"
	Aggressive   OptimizationMode = "aggressive"
)

type ControlConfig struct {
	OptimizationMode OptimizationMode `yaml:"optimization_mode"`
	LastUpdated      time.Time        `yaml:"last_updated"`
	Version          string           `yaml:"version"`
}

type ControlLoop struct {
	prometheusAPI    v1.API
	configPath       string
	targetTS         float64
	conservativeMax  float64
	aggressiveMin    float64
	hysteresisFactor float64
	stabilityPeriod  time.Duration
	
	// PID controller state
	lastError        float64
	integralError    float64
	lastUpdateTime   time.Time
	currentMode      OptimizationMode
	lastTime         time.Time
	
	// PID tuning parameters
	Kp               float64  // Proportional gain
	Ki               float64  // Integral gain
	Kd               float64  // Derivative gain
	integralLimit    float64  // Anti-windup limit
	
	// Metrics
	transitionCount  int
	stabilityScore   float64
}

func NewControlLoop() (*ControlLoop, error) {
	// Initialize Prometheus client
	client, err := api.NewClient(api.Config{
		Address: getEnv("PROMETHEUS_URL", "http://prometheus:9090"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return &ControlLoop{
		prometheusAPI:    v1.NewAPI(client),
		configPath:       getEnv("CONTROL_CONFIG_PATH", "/configs/control/optimization_mode.yaml"),
		targetTS:         getEnvFloat("TARGET_OPTIMIZED_PIPELINE_TS_COUNT", 20000),
		conservativeMax:  getEnvFloat("THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS", 15000),
		aggressiveMin:    getEnvFloat("THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS", 25000),
		hysteresisFactor: getEnvFloat("HYSTERESIS_FACTOR", 0.1),
		stabilityPeriod:  time.Duration(getEnvInt("ADAPTIVE_CONTROLLER_STABILITY_SECONDS", 120)) * time.Second,
		currentMode:      Balanced,
		lastUpdateTime:   time.Now(),
		lastTime:         time.Now(),
		// PID parameters (tuned for cardinality control)
		Kp:               getEnvFloat("PID_KP", 0.5),     // Proportional gain
		Ki:               getEnvFloat("PID_KI", 0.1),     // Integral gain
		Kd:               getEnvFloat("PID_KD", 0.05),    // Derivative gain
		integralLimit:    getEnvFloat("PID_INTEGRAL_LIMIT", 10000), // Anti-windup limit
		stabilityScore:   1.0,
	}, nil
}

func (cl *ControlLoop) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(getEnvInt("ADAPTIVE_CONTROLLER_INTERVAL_SECONDS", 60)) * time.Second)
	defer ticker.Stop()

	log.Println("Starting Phoenix Control Loop (Go implementation)")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := cl.evaluate(); err != nil {
				log.Printf("Control loop evaluation error: %v", err)
			}
		}
	}
}

func (cl *ControlLoop) evaluate() error {
	// Query current cardinality
	currentTS, err := cl.queryCardinality()
	if err != nil {
		return fmt.Errorf("failed to query cardinality: %w", err)
	}

	now := time.Now()
	dt := now.Sub(cl.lastTime).Seconds()
	if dt <= 0 {
		dt = 60.0 // Default to 60s if first run
	}
	
	// Calculate error (negative error means we're below target)
	error := currentTS - cl.targetTS
	
	// Proportional term
	P := cl.Kp * error
	
	// Integral term with anti-windup
	cl.integralError += error * dt
	if cl.integralError > cl.integralLimit {
		cl.integralError = cl.integralLimit
	} else if cl.integralError < -cl.integralLimit {
		cl.integralError = -cl.integralLimit
	}
	I := cl.Ki * cl.integralError
	
	// Derivative term (with filtering to reduce noise)
	derivative := 0.0
	if dt > 0 {
		derivative = (error - cl.lastError) / dt
	}
	D := cl.Kd * derivative
	
	// Full PID output
	pidOutput := P + I + D
	
	log.Printf("PID: Error=%.1f, P=%.2f, I=%.2f, D=%.2f, Output=%.2f, TS=%.0f", 
		error, P, I, D, pidOutput, currentTS)
	
	// Determine new mode with hysteresis
	newMode := cl.calculateMode(currentTS, pidOutput)
	
	// Check stability period
	if newMode != cl.currentMode {
		if time.Since(cl.lastUpdateTime) < cl.stabilityPeriod {
			log.Printf("Skipping mode change (stability period): current=%s, proposed=%s", cl.currentMode, newMode)
			return nil
		}
	}
	
	// Update control signal if changed
	if newMode != cl.currentMode {
		if err := cl.updateControlSignal(newMode); err != nil {
			return fmt.Errorf("failed to update control signal: %w", err)
		}
		
		cl.transitionCount++
		oldMode := cl.currentMode
		cl.currentMode = newMode
		cl.lastUpdateTime = now
		
		log.Printf("Control mode changed: %s -> %s (TS: %.0f, Target: %.0f, PID: %.2f)",
			oldMode, newMode, currentTS, cl.targetTS, pidOutput)
		
		// Reset integral on mode change to prevent windup
		if abs(cl.integralError) > cl.targetTS*0.5 {
			cl.integralError *= 0.5 // Soft reset
		}
	}
	
	// Update stability score
	cl.updateStabilityScore(error)
	
	// Store state for next iteration
	cl.lastError = error
	cl.lastTime = now
	
	return nil
}

func (cl *ControlLoop) queryCardinality() (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{pipeline="optimized"}`
	result, warnings, err := cl.prometheusAPI.Query(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	
	if len(warnings) > 0 {
		log.Printf("Prometheus query warnings: %v", warnings)
	}

	// Extract value from result
	if result.Type() != model.ValVector {
		return 0, fmt.Errorf("unexpected result type: %v", result.Type())
	}

	vector := result.(model.Vector)
	if len(vector) == 0 {
		return 0, fmt.Errorf("no data returned from query")
	}

	return float64(vector[0].Value), nil
}

func (cl *ControlLoop) calculateMode(currentTS, pidOutput float64) OptimizationMode {
	// Apply hysteresis to thresholds
	conservativeThreshold := cl.conservativeMax
	aggressiveThreshold := cl.aggressiveMin
	
	if cl.currentMode == Conservative {
		conservativeThreshold *= (1 + cl.hysteresisFactor)
	} else if cl.currentMode == Aggressive {
		aggressiveThreshold *= (1 - cl.hysteresisFactor)
	}
	
	// Determine mode based on thresholds and PID output
	if currentTS < conservativeThreshold && pidOutput < 0 {
		return Conservative
	} else if currentTS > aggressiveThreshold && pidOutput > 0 {
		return Aggressive
	}
	
	return Balanced
}

func (cl *ControlLoop) updateControlSignal(mode OptimizationMode) error {
	config := ControlConfig{
		OptimizationMode: mode,
		LastUpdated:      time.Now(),
		Version:          "2.0",
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write atomically
	tempFile := cl.configPath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, cl.configPath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	return nil
}

func (cl *ControlLoop) updateStabilityScore(error float64) {
	// Simple exponential moving average of error magnitude
	alpha := 0.1
	cl.stabilityScore = (1-alpha)*cl.stabilityScore + alpha*(1/(1+abs(error/cl.targetTS)))
}

func (cl *ControlLoop) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"current_mode":      cl.currentMode,
		"transition_count":  cl.transitionCount,
		"stability_score":   cl.stabilityScore,
		"integral_error":    cl.integralError,
		"last_error":        cl.lastError,
		"uptime_seconds":    time.Since(cl.lastUpdateTime).Seconds(),
		"pid_kp":            cl.Kp,
		"pid_ki":            cl.Ki,
		"pid_kd":            cl.Kd,
		"pid_integral_limit": cl.integralLimit,
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		var f float64
		fmt.Sscanf(value, "%f", &f)
		return f
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		fmt.Sscanf(value, "%d", &i)
		return i
	}
	return defaultValue
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	ctx := context.Background()
	
	controlLoop, err := NewControlLoop()
	if err != nil {
		log.Fatalf("Failed to initialize control loop: %v", err)
	}

	// Start API endpoints
	go func() {
		// Health check endpoint
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status": "healthy",
				"version": "1.0.0",
			})
		})

		// Metrics endpoint
		http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			metrics := controlLoop.GetMetrics()
			json.NewEncoder(w).Encode(metrics)
		})

		// Mode change endpoint (for testing)
		http.HandleFunc("/mode", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var req struct {
				Mode   string `json:"mode"`
				Reason string `json:"reason"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Validate mode
			if req.Mode != "conservative" && req.Mode != "balanced" && req.Mode != "aggressive" {
				http.Error(w, "Invalid mode", http.StatusBadRequest)
				return
			}

			// Force mode change
			controlLoop.mu.Lock()
			controlLoop.currentMode = req.Mode
			controlLoop.transitionCount++
			controlLoop.mu.Unlock()

			if err := controlLoop.updateControlFile(req.Mode, req.Reason); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status": "success",
				"mode": req.Mode,
			})
		})

		// Anomaly webhook endpoint
		http.HandleFunc("/anomaly", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var anomaly struct {
				AnomalyID     string  `json:"anomaly_id"`
				Severity      string  `json:"severity"`
				MetricName    string  `json:"metric_name"`
				CurrentValue  float64 `json:"current_value"`
				ExpectedValue float64 `json:"expected_value"`
				Confidence    float64 `json:"confidence"`
				Timestamp     string  `json:"timestamp"`
			}

			if err := json.NewDecoder(r.Body).Decode(&anomaly); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			log.Printf("Received anomaly webhook: %+v", anomaly)

			// Take action based on severity
			if anomaly.Severity == "critical" || anomaly.Severity == "high" {
				// Switch to aggressive mode if cardinality explosion
				if anomaly.MetricName == "phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate" &&
					anomaly.CurrentValue > float64(controlLoop.aggressiveMinTS) {
					controlLoop.mu.Lock()
					controlLoop.currentMode = "aggressive"
					controlLoop.transitionCount++
					controlLoop.mu.Unlock()
					controlLoop.updateControlFile("aggressive", "anomaly_detected")
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status": "acknowledged",
				"anomaly_id": anomaly.AnomalyID,
			})
		})

		log.Println("Control Actuator API listening on :8081")
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	// Run control loop
	if err := controlLoop.Run(ctx); err != nil {
		log.Fatalf("Control loop failed: %v", err)
	}
}