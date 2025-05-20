// Package pipelines contains tests for the dual-pipeline architecture.
package pipelines

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_pid"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

// TestMissingSelfMetrics verifies that the control pipeline handles missing
// self-metrics gracefully, which can happen due to scraper failures or timing issues.
// Scenario: PIPE-05
func TestMissingSelfMetrics(t *testing.T) {
	// Skip if running in short mode - this is an integration test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a test metrics collector to capture emitted metrics
	metricsCollector := metrics.NewMetricsCollector()

	// Create PID controller configuration
	config := &adaptive_pid.Config{
		Controllers: []adaptive_pid.ControllerConfig{
			{
				Name:           "test_controller",
				Enabled:        true,
				KPIMetricName:  "test_kpi",  // This KPI won't be present
				KPITargetValue: 0.9,
				KP:             30,
				KI:             5,
				KD:             0,
				OutputConfigPatches: []adaptive_pid.OutputConfigPatch{
					{
						TargetProcessorName: "adaptive_topk",
						ParameterPath:       "k_value",
						ChangeScaleFactor:   1.0,
						MinValue:            10,
						MaxValue:            60,
					},
				},
			},
		},
	}

	// Create the PID processor
	settings := component.TelemetrySettings{}
	processor, err := adaptive_pid.NewProcessor(config, settings, nil, component.NewIDWithName("processor", "pid_decider"))
	require.NoError(t, err, "Failed to create adaptive_pid processor")

	// Create metrics with NO KPI data
	emptyMetrics := createEmptyMetrics(t)
	
	// Process the metrics - this should detect missing KPI
	patches, err := processor.ProcessMetricsForTest(ctx, emptyMetrics)
	require.NoError(t, err, "Should not error with missing KPI metrics")
	
	// No patches should be generated when KPI is missing
	assert.Empty(t, patches, "No patches should be generated with missing KPI")

	// Verify the KPI missing metric is emitted
	time.Sleep(100 * time.Millisecond) // Give time for metrics to be emitted
	
	kpiMissingMetrics := metricsCollector.GetMetricsByName("aemf_pid_decider_kpi_missing_total")
	assert.NotEmpty(t, kpiMissingMetrics, "Should emit aemf_pid_decider_kpi_missing_total metric")
	
	// Now test recovery - send metrics with the KPI present
	metricsWithKPI := createTestMetricsWithKPI(t, "test_kpi", 0.5)
	
	// Process the metrics again
	patches, err = processor.ProcessMetricsForTest(ctx, metricsWithKPI)
	require.NoError(t, err, "Failed to process metrics with KPI")
	
	// Should have generated patches with KPI present
	assert.NotEmpty(t, patches, "Should generate patches when KPI is present")
	
	// Verify patches target the right parameter
	for _, patch := range patches {
		if patch.ParameterPath == "k_value" {
			// Success, found the expected patch
			return
		}
	}
	
	t.Fail("Expected to find patch for k_value parameter")
}

// createEmptyMetrics creates metrics with no KPI data.
func createEmptyMetrics(t *testing.T) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	// Add a resource metrics section with no relevant KPIs
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
	
	// Add some irrelevant metric
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName("irrelevant_metric")
	gauge := metric.SetEmptyGauge()
	dataPoint := gauge.DataPoints().AppendEmpty()
	dataPoint.SetTimestamp(pmetric.NewTimestampFromTime(time.Now()))
	dataPoint.SetDoubleValue(123.45)
	
	return metrics
}

// createTestMetricsWithKPI creates test metrics containing the KPI.
func createTestMetricsWithKPI(t *testing.T, kpiName string, value float64) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
	
	// Add KPI metric
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(kpiName)
	gauge := metric.SetEmptyGauge()
	dataPoint := gauge.DataPoints().AppendEmpty()
	dataPoint.SetTimestamp(pmetric.NewTimestampFromTime(time.Now()))
	dataPoint.SetDoubleValue(value)
	
	return metrics
}