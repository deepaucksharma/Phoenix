// Package e2e contains end-to-end tests for the SA-OMF collector.
package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/deepaucksharma/Phoenix/internal/extension/piccontrolext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_pid"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

// TestControlLoop verifies the basic closed-loop operation of the SA-OMF system.
func TestControlLoop(t *testing.T) {
	// Create components
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	// Create mock components
	mockHost := componenttest.NewNopHost()
	mockMetricsSink := new(consumertest.MetricsSink)
	mockPICControl := newMockPICControl(t)

	// Create processors
	factory := adaptive_pid.NewFactory()
	cfg := factory.CreateDefaultConfig().(*adaptive_pid.Config)
	
	// Set up a controller targeting a specific KPI
	cfg.Controllers[0].KPIMetricName = "aemf_impact_adaptive_topk_resource_coverage_percent"
	cfg.Controllers[0].KPITargetValue = 0.9
	
	processor, err := factory.CreateMetricsProcessor(
		ctx,
		processor.CreateSettings{
			TelemetrySettings: component.TelemetrySettings{
				Logger:         logger,
				TracerProvider: nil,
				MeterProvider:  nil,
				MetricsLevel:   configtelemetry.LevelNone,
			},
			BuildInfo: component.BuildInfo{},
		},
		cfg,
		mockMetricsSink,
	)
	require.NoError(t, err)

	// Start processor
	err = processor.Start(ctx, mockHost)
	require.NoError(t, err)

	// Create test metrics with a coverage value that's below target
	metrics := createTestMetricsWithCoverage(0.7) // Well below target of 0.9
	
	// Process metrics
	err = processor.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	// Verify a ConfigPatch was generated and passed to pic_control
	require.NotEmpty(t, mockPICControl.patches, "No patches were generated")
	
	// Find a patch for k_value
	var kValuePatch *interfaces.ConfigPatch
	for i := range mockPICControl.patches {
		if mockPICControl.patches[i].ParameterPath == "k_value" {
			kValuePatch = &mockPICControl.patches[i]
			break
		}
	}
	
	require.NotNil(t, kValuePatch, "No patch for k_value was generated")
	
	// Verify the patch details
	assert.Equal(t, "adaptive_topk", kValuePatch.TargetProcessorName.String(), "Unexpected target processor")
	
	// Since coverage is too low, k_value should increase
	newValue, ok := kValuePatch.NewValue.(float64)
	require.True(t, ok, "New value should be a float64")
	assert.Greater(t, newValue, float64(30), "k_value should increase when coverage is below target")

	// Shutdown processor
	err = processor.Shutdown(ctx)
	require.NoError(t, err)
}

// mockPICControl is a mock implementation of the PicControl interface
type mockPICControl struct {
	t       *testing.T
	patches []interfaces.ConfigPatch
}

func newMockPICControl(t *testing.T) *mockPICControl {
	return &mockPICControl{
		t:       t,
		patches: []interfaces.ConfigPatch{},
	}
}

func (m *mockPICControl) SubmitConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	m.patches = append(m.patches, patch)
	m.t.Logf("Received patch: %s -> %s = %v", patch.TargetProcessorName, patch.ParameterPath, patch.NewValue)
	return nil
}

// createTestMetricsWithCoverage creates test metrics with a specified coverage value
func createTestMetricsWithCoverage(coverage float64) pmetric.Metrics {
	md := pmetric.NewMetrics()
	
	// Add resource metrics
	rm := md.ResourceMetrics().AppendEmpty()
	
	// Add scope metrics
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("adaptive_topk")
	
	// Add coverage metric
	coverageMetric := sm.Metrics().AppendEmpty()
	coverageMetric.SetName("aemf_impact_adaptive_topk_resource_coverage_percent")
	coverageMetric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(coverage)
	
	return md
}