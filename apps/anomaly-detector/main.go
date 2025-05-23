package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type AnomalyDetector struct {
	prometheusAPI      v1.API
	detectors          map[string]Detector
	alerts             []Alert
	mu                 sync.RWMutex
	alertWebhookURL    string
	controlWebhookURL  string
}

type Detector interface {
	Detect(ctx context.Context, metrics map[string][]DataPoint) ([]Anomaly, error)
	GetName() string
}

type DataPoint struct {
	Timestamp time.Time
	Value     float64
	Labels    map[string]string
}

type Anomaly struct {
	DetectorName string            `json:"detector_name"`
	MetricName   string            `json:"metric_name"`
	Timestamp    time.Time         `json:"timestamp"`
	Value        float64           `json:"value"`
	Expected     float64           `json:"expected"`
	Severity     string            `json:"severity"`
	Confidence   float64           `json:"confidence"`
	Labels       map[string]string `json:"labels"`
	Description  string            `json:"description"`
}

type Alert struct {
	ID          string    `json:"id"`
	Anomaly     Anomaly   `json:"anomaly"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
	ActionTaken string    `json:"action_taken,omitempty"`
}

// Statistical detector using z-score
type StatisticalDetector struct {
	windowSize int
	threshold  float64
}

func (sd *StatisticalDetector) GetName() string {
	return "statistical_zscore"
}

func (sd *StatisticalDetector) Detect(ctx context.Context, metrics map[string][]DataPoint) ([]Anomaly, error) {
	var anomalies []Anomaly

	for metricName, dataPoints := range metrics {
		if len(dataPoints) < sd.windowSize {
			continue
		}

		// Calculate moving statistics
		for i := sd.windowSize; i < len(dataPoints); i++ {
			window := dataPoints[i-sd.windowSize : i]
			mean, stdDev := calculateStats(window)
			
			current := dataPoints[i]
			zScore := math.Abs((current.Value - mean) / stdDev)
			
			if zScore > sd.threshold {
				anomalies = append(anomalies, Anomaly{
					DetectorName: sd.GetName(),
					MetricName:   metricName,
					Timestamp:    current.Timestamp,
					Value:        current.Value,
					Expected:     mean,
					Severity:     sd.calculateSeverity(zScore),
					Confidence:   math.Min(zScore/10.0, 1.0),
					Labels:       current.Labels,
					Description:  fmt.Sprintf("Value %.2f is %.1f standard deviations from mean %.2f", current.Value, zScore, mean),
				})
			}
		}
	}

	return anomalies, nil
}

func (sd *StatisticalDetector) calculateSeverity(zScore float64) string {
	switch {
	case zScore > 5:
		return "critical"
	case zScore > 4:
		return "high"
	case zScore > 3:
		return "medium"
	default:
		return "low"
	}
}

// Rate of change detector
type RateOfChangeDetector struct {
	threshold float64
}

func (rd *RateOfChangeDetector) GetName() string {
	return "rate_of_change"
}

func (rd *RateOfChangeDetector) Detect(ctx context.Context, metrics map[string][]DataPoint) ([]Anomaly, error) {
	var anomalies []Anomaly

	for metricName, dataPoints := range metrics {
		if len(dataPoints) < 2 {
			continue
		}

		for i := 1; i < len(dataPoints); i++ {
			prev := dataPoints[i-1]
			current := dataPoints[i]
			
			timeDiff := current.Timestamp.Sub(prev.Timestamp).Seconds()
			if timeDiff == 0 {
				continue
			}
			
			rateOfChange := math.Abs((current.Value - prev.Value) / timeDiff)
			expectedRate := math.Abs(prev.Value) * 0.1 / 60 // 10% change per minute expected max
			
			if rateOfChange > rd.threshold && rateOfChange > expectedRate*3 {
				anomalies = append(anomalies, Anomaly{
					DetectorName: rd.GetName(),
					MetricName:   metricName,
					Timestamp:    current.Timestamp,
					Value:        current.Value,
					Expected:     prev.Value,
					Severity:     rd.calculateSeverity(rateOfChange, expectedRate),
					Confidence:   math.Min(rateOfChange/(expectedRate*10), 1.0),
					Labels:       current.Labels,
					Description:  fmt.Sprintf("Rapid change detected: %.2f/s (expected max: %.2f/s)", rateOfChange, expectedRate),
				})
			}
		}
	}

	return anomalies, nil
}

func (rd *RateOfChangeDetector) calculateSeverity(rate, expected float64) string {
	ratio := rate / expected
	switch {
	case ratio > 10:
		return "critical"
	case ratio > 5:
		return "high"
	case ratio > 3:
		return "medium"
	default:
		return "low"
	}
}

// Pattern detector for known bad patterns
type PatternDetector struct {
	patterns []Pattern
}

type Pattern struct {
	Name        string
	Description string
	Detect      func([]DataPoint) bool
	Severity    string
}

func (pd *PatternDetector) GetName() string {
	return "pattern_matching"
}

func (pd *PatternDetector) Detect(ctx context.Context, metrics map[string][]DataPoint) ([]Anomaly, error) {
	var anomalies []Anomaly

	// Define patterns
	pd.patterns = []Pattern{
		{
			Name:        "cardinality_explosion",
			Description: "Exponential growth in cardinality detected",
			Severity:    "critical",
			Detect: func(points []DataPoint) bool {
				if len(points) < 5 {
					return false
				}
				// Check for exponential growth
				growthRates := make([]float64, 0)
				for i := 1; i < len(points); i++ {
					if points[i-1].Value > 0 {
						growthRate := (points[i].Value - points[i-1].Value) / points[i-1].Value
						growthRates = append(growthRates, growthRate)
					}
				}
				// If growth rate is consistently increasing, it's exponential
				increasingCount := 0
				for i := 1; i < len(growthRates); i++ {
					if growthRates[i] > growthRates[i-1] && growthRates[i] > 0.1 {
						increasingCount++
					}
				}
				return increasingCount >= 3
			},
		},
		{
			Name:        "memory_leak",
			Description: "Continuous memory growth without recovery",
			Severity:    "high",
			Detect: func(points []DataPoint) bool {
				if len(points) < 10 {
					return false
				}
				// Check for monotonic increase
				increasingCount := 0
				for i := 1; i < len(points); i++ {
					if points[i].Value > points[i-1].Value {
						increasingCount++
					}
				}
				return float64(increasingCount)/float64(len(points)-1) > 0.8
			},
		},
		{
			Name:        "oscillation",
			Description: "Rapid oscillation detected, possible control loop issue",
			Severity:    "medium",
			Detect: func(points []DataPoint) bool {
				if len(points) < 6 {
					return false
				}
				// Count direction changes
				directionChanges := 0
				for i := 2; i < len(points); i++ {
					prev := points[i-1].Value - points[i-2].Value
					curr := points[i].Value - points[i-1].Value
					if prev*curr < 0 { // Different signs = direction change
						directionChanges++
					}
				}
				return directionChanges >= len(points)/2
			},
		},
	}

	for metricName, dataPoints := range metrics {
		for _, pattern := range pd.patterns {
			if pattern.Detect(dataPoints) {
				lastPoint := dataPoints[len(dataPoints)-1]
				anomalies = append(anomalies, Anomaly{
					DetectorName: pd.GetName(),
					MetricName:   metricName,
					Timestamp:    lastPoint.Timestamp,
					Value:        lastPoint.Value,
					Expected:     0, // Pattern detection doesn't have expected value
					Severity:     pattern.Severity,
					Confidence:   0.9,
					Labels:       lastPoint.Labels,
					Description:  fmt.Sprintf("Pattern '%s' detected: %s", pattern.Name, pattern.Description),
				})
			}
		}
	}

	return anomalies, nil
}

func NewAnomalyDetector() (*AnomalyDetector, error) {
	client, err := api.NewClient(api.Config{
		Address: getEnv("PROMETHEUS_URL", "http://prometheus:9090"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	ad := &AnomalyDetector{
		prometheusAPI:     v1.NewAPI(client),
		detectors:         make(map[string]Detector),
		alerts:            make([]Alert, 0),
		alertWebhookURL:   getEnv("ALERT_WEBHOOK_URL", ""),
		controlWebhookURL: getEnv("CONTROL_WEBHOOK_URL", "http://control-actuator:8080/anomaly"),
	}

	// Register detectors
	ad.detectors["statistical"] = &StatisticalDetector{
		windowSize: 20,
		threshold:  3.0,
	}
	ad.detectors["rate_of_change"] = &RateOfChangeDetector{
		threshold: 100.0, // 100 units per second
	}
	ad.detectors["pattern"] = &PatternDetector{}

	return ad, nil
}

func (ad *AnomalyDetector) Run(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Starting Phoenix Anomaly Detection System")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := ad.detectAnomalies(ctx); err != nil {
				log.Printf("Anomaly detection error: %v", err)
			}
		}
	}
}

func (ad *AnomalyDetector) detectAnomalies(ctx context.Context) error {
	// Define metrics to monitor
	metricsToMonitor := []string{
		"phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate",
		"phoenix:cardinality_growth_rate",
		"phoenix:memory_utilization_percentage",
		"phoenix:cpu_utilization_percentage",
		"phoenix:pipeline_error_rate",
		"phoenix:control_mode_transitions_total",
	}

	allAnomalies := make([]Anomaly, 0)

	for _, metricName := range metricsToMonitor {
		// Query last 30 minutes of data
		data, err := ad.queryMetricHistory(ctx, metricName, 30*time.Minute)
		if err != nil {
			log.Printf("Failed to query %s: %v", metricName, err)
			continue
		}

		// Run all detectors
		for _, detector := range ad.detectors {
			anomalies, err := detector.Detect(ctx, map[string][]DataPoint{metricName: data})
			if err != nil {
				log.Printf("Detector %s failed: %v", detector.GetName(), err)
				continue
			}
			allAnomalies = append(allAnomalies, anomalies...)
		}
	}

	// Process detected anomalies
	for _, anomaly := range allAnomalies {
		ad.processAnomaly(anomaly)
	}

	return nil
}

func (ad *AnomalyDetector) queryMetricHistory(ctx context.Context, metricName string, duration time.Duration) ([]DataPoint, error) {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	query := fmt.Sprintf(`%s[%s]`, metricName, duration.String())
	result, warnings, err := ad.prometheusAPI.QueryRange(ctx, query, v1.Range{
		Start: startTime,
		End:   endTime,
		Step:  30 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	if len(warnings) > 0 {
		log.Printf("Query warnings: %v", warnings)
	}

	dataPoints := make([]DataPoint, 0)

	switch result.Type() {
	case model.ValMatrix:
		matrix := result.(model.Matrix)
		for _, series := range matrix {
			labels := make(map[string]string)
			for k, v := range series.Metric {
				labels[string(k)] = string(v)
			}
			for _, sample := range series.Values {
				dataPoints = append(dataPoints, DataPoint{
					Timestamp: sample.Timestamp.Time(),
					Value:     float64(sample.Value),
					Labels:    labels,
				})
			}
		}
	}

	return dataPoints, nil
}

func (ad *AnomalyDetector) processAnomaly(anomaly Anomaly) {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	// Create alert
	alert := Alert{
		ID:        fmt.Sprintf("%s-%d", anomaly.MetricName, time.Now().Unix()),
		Anomaly:   anomaly,
		CreatedAt: time.Now(),
		Status:    "active",
	}

	// Check if similar alert already exists
	for _, existing := range ad.alerts {
		if existing.Anomaly.MetricName == anomaly.MetricName &&
			existing.Anomaly.DetectorName == anomaly.DetectorName &&
			existing.Status == "active" &&
			time.Since(existing.CreatedAt) < 5*time.Minute {
			// Skip duplicate alert
			return
		}
	}

	ad.alerts = append(ad.alerts, alert)
	log.Printf("Anomaly detected: %s - %s (severity: %s, confidence: %.2f)",
		anomaly.MetricName, anomaly.Description, anomaly.Severity, anomaly.Confidence)

	// Take action based on severity
	if anomaly.Severity == "critical" || anomaly.Severity == "high" {
		ad.takeAction(alert)
	}

	// Send webhook notification
	if ad.alertWebhookURL != "" {
		go ad.sendWebhook(alert)
	}
}

func (ad *AnomalyDetector) takeAction(alert Alert) {
	// Send control signal for critical anomalies
	if alert.Anomaly.MetricName == "phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate" &&
		alert.Anomaly.Description == "cardinality_explosion" {
		
		// Notify control loop to switch to aggressive mode
		payload := map[string]interface{}{
			"anomaly_type": "cardinality_explosion",
			"severity":     alert.Anomaly.Severity,
			"timestamp":    alert.Anomaly.Timestamp,
			"recommended_action": "switch_to_aggressive",
		}

		jsonData, _ := json.Marshal(payload)
		resp, err := http.Post(ad.controlWebhookURL, "application/json", bytes.NewReader(jsonData))
		if err != nil {
			log.Printf("Failed to notify control loop: %v", err)
		} else {
			resp.Body.Close()
			alert.ActionTaken = "Notified control loop to switch to aggressive mode"
		}
	}
}

func (ad *AnomalyDetector) sendWebhook(alert Alert) {
	jsonData, err := json.Marshal(alert)
	if err != nil {
		log.Printf("Failed to marshal alert: %v", err)
		return
	}

	resp, err := http.Post(ad.alertWebhookURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		log.Printf("Failed to send webhook: %v", err)
		return
	}
	defer resp.Body.Close()
}

func (ad *AnomalyDetector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/alerts":
		ad.handleGetAlerts(w, r)
	case "/health":
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	default:
		http.NotFound(w, r)
	}
}

func (ad *AnomalyDetector) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	ad.mu.RLock()
	alerts := ad.alerts
	ad.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// Helper functions
func calculateStats(points []DataPoint) (mean, stdDev float64) {
	n := float64(len(points))
	if n == 0 {
		return 0, 0
	}

	// Calculate mean
	sum := 0.0
	for _, p := range points {
		sum += p.Value
	}
	mean = sum / n

	// Calculate standard deviation
	sumSquares := 0.0
	for _, p := range points {
		diff := p.Value - mean
		sumSquares += diff * diff
	}
	variance := sumSquares / n
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	log.Println("Starting Phoenix Anomaly Detection System")

	detector, err := NewAnomalyDetector()
	if err != nil {
		log.Fatalf("Failed to initialize anomaly detector: %v", err)
	}

	// Start detection loop in background
	ctx := context.Background()
	go func() {
		if err := detector.Run(ctx); err != nil {
			log.Fatalf("Anomaly detection loop failed: %v", err)
		}
	}()

	// Start HTTP server
	port := getEnv("PORT", "8080")
	log.Printf("Server listening on port %s", port)
	
	if err := http.ListenAndServe(":"+port, detector); err != nil {
		log.Fatal(err)
	}
}