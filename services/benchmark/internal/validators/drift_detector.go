package validators

import (
	"fmt"
	"math"
	"time"
)

// DriftDetector detects performance drift over time
type DriftDetector struct {
	baselineMetrics map[string]float64
	driftThreshold  float64
	windowSize      int
}

type DriftMetrics struct {
	Timestamp time.Time
	Metrics   map[string]float64
}

type DriftAnalysis struct {
	DriftDetected      bool
	DriftPercentage    float64
	AffectedMetrics    []string
	TrendDirection     string // "improving", "degrading", "stable"
	RecommendedAction  string
}

func NewDriftDetector() *DriftDetector {
	return &DriftDetector{
		baselineMetrics: make(map[string]float64),
		driftThreshold:  0.1, // 10% drift threshold
		windowSize:      10,  // Compare against last 10 measurements
	}
}

func (dd *DriftDetector) SetBaseline(metrics map[string]float64) {
	dd.baselineMetrics = make(map[string]float64)
	for k, v := range metrics {
		dd.baselineMetrics[k] = v
	}
}

func (dd *DriftDetector) Analyze(currentMetrics map[string]float64, historicalData []DriftMetrics) *DriftAnalysis {
	analysis := &DriftAnalysis{
		DriftDetected:   false,
		AffectedMetrics: []string{},
	}

	if len(dd.baselineMetrics) == 0 {
		analysis.RecommendedAction = "Set baseline metrics first"
		return analysis
	}

	var totalDrift float64
	var driftCount int

	// Compare current metrics against baseline
	for metric, baseline := range dd.baselineMetrics {
		current, exists := currentMetrics[metric]
		if !exists {
			continue
		}

		drift := (current - baseline) / baseline
		if math.Abs(drift) > dd.driftThreshold {
			analysis.DriftDetected = true
			analysis.AffectedMetrics = append(analysis.AffectedMetrics, 
				fmt.Sprintf("%s (%.1f%% drift)", metric, drift*100))
		}
		totalDrift += drift
		driftCount++
	}

	if driftCount > 0 {
		analysis.DriftPercentage = (totalDrift / float64(driftCount)) * 100
	}

	// Determine trend direction
	if len(historicalData) >= dd.windowSize {
		trend := dd.calculateTrend(historicalData[len(historicalData)-dd.windowSize:])
		analysis.TrendDirection = trend
	} else {
		analysis.TrendDirection = "insufficient_data"
	}

	// Set recommended action
	analysis.RecommendedAction = dd.recommendAction(analysis)

	return analysis
}

func (dd *DriftDetector) calculateTrend(data []DriftMetrics) string {
	if len(data) < 2 {
		return "stable"
	}

	var improvements, degradations int
	
	for i := 1; i < len(data); i++ {
		prev := data[i-1].Metrics
		curr := data[i].Metrics
		
		for metric := range prev {
			if currVal, exists := curr[metric]; exists {
				change := (currVal - prev[metric]) / prev[metric]
				
				// For metrics where lower is better (e.g., latency, cost)
				if isLowerBetterMetric(metric) {
					if change < -0.05 {
						improvements++
					} else if change > 0.05 {
						degradations++
					}
				} else {
					// For metrics where higher is better (e.g., throughput)
					if change > 0.05 {
						improvements++
					} else if change < -0.05 {
						degradations++
					}
				}
			}
		}
	}

	if improvements > degradations*2 {
		return "improving"
	} else if degradations > improvements*2 {
		return "degrading"
	}
	return "stable"
}

func (dd *DriftDetector) recommendAction(analysis *DriftAnalysis) string {
	if !analysis.DriftDetected {
		return "No action required - performance within acceptable bounds"
	}

	if analysis.TrendDirection == "degrading" {
		if analysis.DriftPercentage > 20 {
			return "CRITICAL: Immediate investigation required - significant performance degradation"
		}
		return "WARNING: Monitor closely - performance degradation detected"
	}

	if analysis.TrendDirection == "improving" {
		return "INFO: Positive drift detected - consider updating baseline"
	}

	return "INFO: Performance drift detected - review affected metrics"
}

func isLowerBetterMetric(metric string) bool {
	lowerBetterMetrics := []string{
		"latency", "cpu_usage", "memory_usage", "error_rate", 
		"cost", "p99_latency", "p95_latency", "p50_latency",
	}
	
	for _, m := range lowerBetterMetrics {
		if contains(metric, m) {
			return true
		}
	}
	return false
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && str[:len(substr)] == substr
}

func (dd *DriftDetector) Name() string {
	return "drift_detector"
}