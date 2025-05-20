package adaptivepid

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_pid"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

// TestAdaptivePIDProcessor_MetricsIntegration tests the integration between the
// adaptive_pid processor and the unified metrics collection system
func TestAdaptivePIDProcessor_MetricsIntegration(t *testing.T) {
	// Create a new logger
	logger := zaptest.NewLogger(t)

	// Create a shared metrics collector
	metricsCollector := metrics.NewUnifiedMetricsCollector(logger)
	metricsCollector.AddDefaultAttribute("component", "adaptive_pid")

	// Create a factory and configuration
	factory := adaptive_pid.NewFactory()
	cfg := factory.CreateDefaultConfig().(*adaptive_pid.Config)

	// Configure a coverage controller
	cfg.Controllers = []adaptive_pid.ControllerConfig{
		{
			Name:              "coverage_controller",
			Enabled:           true,
			KPIMetricName:     "aemf_impact_resource_filter_coverage_percent_avg_1m",
			KPITargetValue:    0.95, // Target 95% coverage
			KP:                10,
			KI:                2,
			KD:                0.5,
			HysteresisPercent: 5,
			OutputConfigPatches: []adaptive_pid.OutputConfigPatch{
				{
					TargetProcessorName: "metric_pipeline",
					ParameterPath:       "resource_filter.topk.k_value",
					ChangeScaleFactor:   -15.0,
					MinValue:            10,
					MaxValue:            40,
				},
			},
		},
	}

	// Create a test metrics sink
	sink := new(consumertest.MetricsSink)

	// Create telemetry settings with the logger
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: logger,
		},
		ID: component.NewIDWithName(component.MustNewType("adaptive_pid"), ""),
	}

	// Create the processor
	proc, err := factory.CreateMetrics(context.Background(), settings, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Ensure it implements the UpdateableProcessor interface
	updateableProc, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok, "Processor does not implement UpdateableProcessor")

	// Start the processor
	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	// Create test metrics with KPI values
	metrics := generateTestMetricsWithKPI(0.80) // 80% coverage, below target of 95%

	// Process metrics to trigger the PID controller
	err = proc.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)

	// Verify the metrics were processed
	processedMetrics := sink.AllMetrics()
	assert.NotEmpty(t, processedMetrics)

	// Generate more metrics with a different coverage value
	metrics = generateTestMetricsWithKPI(0.90) // 90% coverage, getting closer to target
	err = proc.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)

	// Generate metrics at target
	metrics = generateTestMetricsWithKPI(0.95) // 95% coverage, at target
	err = proc.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)

	// Generate metrics above target
	metrics = generateTestMetricsWithKPI(0.99) // 99% coverage, above target
	err = proc.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)

	// Get the config status
	status, err := updateableProc.GetConfigStatus(context.Background())
	require.NoError(t, err)
	assert.True(t, status.Enabled)

	// Shut down the processor
	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}

// generateTestMetricsWithKPI creates test metrics with a specific KPI value
func generateTestMetricsWithKPI(kpiValue float64) pmetric.Metrics {
	metrics := pmetric.NewMetrics()

	// Create a resource metric
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "phoenix-test")

	// Add a scope metric
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("phoenix.impact")

	// Add the KPI metric
	kpiMetric := sm.Metrics().AppendEmpty()
	kpiMetric.SetName("aemf_impact_resource_filter_coverage_percent_avg_1m")
	kpiMetric.SetDescription("Coverage percentage for resource filter")
	kpiMetric.SetEmptyGauge()
	dp := kpiMetric.Gauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(kpiValue)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Add some additional metrics
	cardinality := sm.Metrics().AppendEmpty()
	cardinality.SetName("aemf_metrics_cardinality")
	cardinality.SetDescription("Metric cardinality")
	cardinality.SetEmptyGauge()
	cardDp := cardinality.Gauge().DataPoints().AppendEmpty()
	cardDp.SetDoubleValue(500.0)
	cardDp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	return metrics
}

// TestAdaptivePIDProcessor_MultipleControllers tests that multiple controllers can
// operate simultaneously with the unified metrics system
func TestAdaptivePIDProcessor_MultipleControllers(t *testing.T) {
	// Create a factory and configuration
	factory := adaptive_pid.NewFactory()
	cfg := factory.CreateDefaultConfig().(*adaptive_pid.Config)

	// Configure multiple controllers
	cfg.Controllers = []adaptive_pid.ControllerConfig{
		{
			Name:              "coverage_controller",
			Enabled:           true,
			KPIMetricName:     "aemf_impact_resource_filter_coverage_percent_avg_1m",
			KPITargetValue:    0.95, // Target 95% coverage
			KP:                10,
			KI:                2,
			KD:                0.5,
			HysteresisPercent: 5,
			OutputConfigPatches: []adaptive_pid.OutputConfigPatch{
				{
					TargetProcessorName: "metric_pipeline",
					ParameterPath:       "resource_filter.topk.k_value",
					ChangeScaleFactor:   -15.0,
					MinValue:            10,
					MaxValue:            40,
				},
			},
		},
		{
			Name:              "cardinality_controller",
			Enabled:           true,
			KPIMetricName:     "aemf_metrics_cardinality",
			KPITargetValue:    500, // Target 500 unique metrics
			KP:                15,
			KI:                3,
			KD:                0.3,
			HysteresisPercent: 10,
			OutputConfigPatches: []adaptive_pid.OutputConfigPatch{
				{
					TargetProcessorName: "metric_pipeline",
					ParameterPath:       "resource_filter.rollup.priority_threshold",
					ValueMap: map[float64]string{
						-10.0: "low",    // When cardinality is too low, rollup only low priority
						0.0:   "medium", // At target, rollup low and medium priority
						10.0:  "high",   // When cardinality is too high, rollup everything except critical
					},
				},
			},
		},
	}

	// Create a test metrics sink
	sink := new(consumertest.MetricsSink)

	// Create telemetry settings with the logger
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewIDWithName(component.MustNewType("adaptive_pid"), ""),
	}

	// Create the processor
	proc, err := factory.CreateMetrics(context.Background(), settings, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Start the processor
	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	// Create test metrics that will trigger both controllers
	metrics := pmetric.NewMetrics()

	// Create a resource metric
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "phoenix-test")

	// Add a scope metric
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("phoenix.impact")

	// Add the coverage KPI metric
	coverageMetric := sm.Metrics().AppendEmpty()
	coverageMetric.SetName("aemf_impact_resource_filter_coverage_percent_avg_1m")
	coverageMetric.SetEmptyGauge()
	coverageDp := coverageMetric.Gauge().DataPoints().AppendEmpty()
	coverageDp.SetDoubleValue(0.80) // 80% coverage, below target of 95%
	coverageDp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Add the cardinality KPI metric
	cardinalityMetric := sm.Metrics().AppendEmpty()
	cardinalityMetric.SetName("aemf_metrics_cardinality")
	cardinalityMetric.SetEmptyGauge()
	cardinalityDp := cardinalityMetric.Gauge().DataPoints().AppendEmpty()
	cardinalityDp.SetDoubleValue(700.0) // 700 metrics, above target of 500
	cardinalityDp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Process metrics to trigger both controllers
	err = proc.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)

	// Verify the metrics were processed
	processedMetrics := sink.AllMetrics()
	assert.NotEmpty(t, processedMetrics)

	// Shutdown the processor
	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}