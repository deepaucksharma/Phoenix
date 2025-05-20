//go:build test

package metrics

import "go.opentelemetry.io/collector/pdata/pmetric"

// capturedMetrics stores metrics passed to AddMetrics for each emitter instance.
var capturedMetrics = make(map[*MetricsEmitter][]pmetric.Metrics)

// AddMetrics is a test-only helper that records emitted metrics.
func (e *MetricsEmitter) AddMetrics(m pmetric.Metrics) {
	capturedMetrics[e] = append(capturedMetrics[e], m)
}

// resetCapturedMetrics clears recorded metrics for all emitters.
func ResetCapturedMetrics() {
	for k := range capturedMetrics {
		delete(capturedMetrics, k)
	}
}

// metricsFor returns the metrics captured for the given emitter.
func MetricsFor(e *MetricsEmitter) []pmetric.Metrics {
	return capturedMetrics[e]
}
