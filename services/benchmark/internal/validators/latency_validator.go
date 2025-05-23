package validators

import (
	"fmt"
	"time"
)

// LatencyValidator validates pipeline processing latency
type LatencyValidator struct {
	thresholds LatencyThresholds
}

type LatencyThresholds struct {
	P50Max time.Duration
	P95Max time.Duration
	P99Max time.Duration
}

type LatencyMetrics struct {
	P50 time.Duration
	P95 time.Duration
	P99 time.Duration
}

func NewLatencyValidator() *LatencyValidator {
	return &LatencyValidator{
		thresholds: LatencyThresholds{
			P50Max: 20 * time.Millisecond,
			P95Max: 40 * time.Millisecond,
			P99Max: 50 * time.Millisecond,
		},
	}
}

func (lv *LatencyValidator) Validate(metrics LatencyMetrics) (bool, []string) {
	var passed = true
	var failures []string

	if metrics.P50 > lv.thresholds.P50Max {
		passed = false
		failures = append(failures, fmt.Sprintf("P50 latency %.2fms exceeds threshold %.2fms", 
			metrics.P50.Seconds()*1000, lv.thresholds.P50Max.Seconds()*1000))
	}

	if metrics.P95 > lv.thresholds.P95Max {
		passed = false
		failures = append(failures, fmt.Sprintf("P95 latency %.2fms exceeds threshold %.2fms", 
			metrics.P95.Seconds()*1000, lv.thresholds.P95Max.Seconds()*1000))
	}

	if metrics.P99 > lv.thresholds.P99Max {
		passed = false
		failures = append(failures, fmt.Sprintf("P99 latency %.2fms exceeds threshold %.2fms", 
			metrics.P99.Seconds()*1000, lv.thresholds.P99Max.Seconds()*1000))
	}

	return passed, failures
}

func (lv *LatencyValidator) Name() string {
	return "latency_validator"
}