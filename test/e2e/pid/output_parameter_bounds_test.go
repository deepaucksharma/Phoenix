// Package pid contains end-to-end tests for PID controller behavior.
package pid

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_pid"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

// TestOutputParameterBounds verifies that PID controller output values are
// properly clamped to defined min/max bounds.
// Scenario: PID-07
func TestOutputParameterBounds(t *testing.T) {
	// Skip if running in short mode - this is an integration test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a test metrics collector to capture emitted metrics
	metricsCollector := metrics.NewMetricsCollector()

	// Create PID controller with high gain to force output out of bounds
	// This is the configuration for a PID controller that would generate
	// very large outputs to test clamping behavior
	config := &adaptive_pid.Config{
		Controllers: []adaptive_pid.ControllerConfig{
			{
				Name:           "test_controller",
				Enabled:        true,
				KPIMetricName:  "test_kpi",
				KPITargetValue: 0.9,  // Target value
				KP:             500,   // Very high proportional gain to force large outputs
				KI:             0,     // No integral term for simplicity
				KD:             0,     // No derivative term for simplicity
				OutputConfigPatches: []adaptive_pid.OutputConfigPatch{
					{
						TargetProcessorName: "adaptive_topk",
						ParameterPath:       "k_value",
						ChangeScaleFactor:   1.0,
						MinValue:            5,    // Minimum allowed value
						MaxValue:            100,  // Maximum allowed value
					},
				},
			},
		},
	}

	// Create the PID processor
	settings := component.TelemetrySettings{}
	processor, err := adaptive_pid.NewProcessor(config, settings, nil, component.NewIDWithName(component.MustNewType("processor"), "adaptive_pid"))
	require.NoError(t, err, "Failed to create adaptive_pid processor")

	// Create mock metrics to simulate a large error
	metrics := createTestMetrics(t, "test_kpi", 0.1) // Far from target of 0.9
	
	// Process the metrics to generate patches
	patches, err := processor.ProcessMetricsForTest(ctx, metrics)
	require.NoError(t, err, "Failed to process metrics")
	
	// Verify we have at least one patch
	require.NotEmpty(t, patches, "Should generate at least one patch")
	
	// Find the patch for k_value
	var kValuePatch *interfaces.ConfigPatch
	for _, p := range patches {
		if p.ParameterPath == "k_value" {
			kValuePatch = &p
			break
		}
	}
	
	require.NotNil(t, kValuePatch, "Should generate a patch for k_value")
	
	// Verify the value is clamped to the maximum
	value, ok := kValuePatch.NewValue.(float64)
	require.True(t, ok, "Patch value should be a float64")
	
	// Check that the value is exactly the max value (100)
	assert.Equal(t, 100.0, value, "Patch value should be clamped to max value")
	
	// Verify aemf_pid_output_clamped_total metric was emitted
	time.Sleep(100 * time.Millisecond) // Give time for metrics to be emitted
	
	clampedMetrics := metricsCollector.GetMetricsByName("aemf_pid_output_clamped_total")
	assert.NotEmpty(t, clampedMetrics, "Should emit aemf_pid_output_clamped_total metric")
	
	// Now test with a value that would go below the minimum
	// Create metrics to force a negative error (target < actual)
	metrics = createTestMetrics(t, "test_kpi", 1.5) // Above target of 0.9
	
	// Process the metrics to generate patches
	patches, err = processor.ProcessMetricsForTest(ctx, metrics)
	require.NoError(t, err, "Failed to process metrics")
	
	// Find the patch for k_value
	kValuePatch = nil
	for _, p := range patches {
		if p.ParameterPath == "k_value" {
			kValuePatch = &p
			break
		}
	}
	
	require.NotNil(t, kValuePatch, "Should generate a patch for k_value")
	
	// Verify the value is clamped to the minimum
	value, ok = kValuePatch.NewValue.(float64)
	require.True(t, ok, "Patch value should be a float64")
	
	// Check that the value is exactly the min value (5)
	assert.Equal(t, 5.0, value, "Patch value should be clamped to min value")
}

// createTestMetrics creates test metrics for PID controller testing.
func createTestMetrics(t *testing.T, name string, value float64) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
	
	// Add KPI metric
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	gauge := metric.SetEmptyGauge()
	dataPoint := gauge.DataPoints().AppendEmpty()
	dataPoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dataPoint.SetDoubleValue(value)
	
	return metrics
}