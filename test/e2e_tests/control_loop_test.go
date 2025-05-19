package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_pid"
	"github.com/deepaucksharma/Phoenix/internal/processor/priority_tagger"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// TestControlLoop sets up a minimal metrics pipeline with priority tagger and
// adaptive PID processors to verify basic integration.
func TestControlLoop(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Sink to receive metrics at the end of the pipeline.
	sink := new(consumertest.MetricsSink)

	// Configure adaptive PID processor.
	pidFactory := adaptive_pid.NewFactory()
	pidCfg := pidFactory.CreateDefaultConfig().(*adaptive_pid.Config)
	pidCfg.Controllers = []adaptive_pid.ControllerConfig{
		{
			Name:              "ctrl",
			Enabled:           true,
			KPIMetricName:     "aemf_impact_test_metric",
			KPITargetValue:    0.80,
			KP:                10,
			KI:                2,
			KD:                0,
			HysteresisPercent: 1,
			OutputConfigPatches: []adaptive_pid.OutputConfigPatch{
				{
					TargetProcessorName: "priority_tagger",
					ParameterPath:       "enabled",
					ChangeScaleFactor:   0,
					MinValue:            0,
					MaxValue:            1,
				},
			},
		},
	}
	pidProc, err := pidFactory.CreateMetrics(ctx, processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewIDWithName(component.MustNewType("pid_decider"), ""),
	}, pidCfg, sink)
	require.NoError(t, err)

	// Configure priority tagger processor.
	taggerFactory := priority_tagger.NewFactory()
	taggerCfg := taggerFactory.CreateDefaultConfig().(*priority_tagger.Config)
	taggerCfg.Enabled = true
	taggerCfg.Rules = []priority_tagger.Rule{
		{Match: ".*", Priority: "low"},
	}

	taggerProc, err := taggerFactory.CreateMetrics(ctx, processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewIDWithName(component.MustNewType("priority_tagger"), ""),
	}, taggerCfg, pidProc)
	require.NoError(t, err)

	// Start processors.
	require.NoError(t, pidProc.Start(ctx, nil))
	require.NoError(t, taggerProc.Start(ctx, nil))
	defer pidProc.Shutdown(ctx)
	defer taggerProc.Shutdown(ctx)

	// Create metrics containing KPI values and process attributes.
	metrics := testutils.GenerateTestMetrics(1)
	kpi := testutils.GenerateControlLoopMetrics(map[string]float64{"aemf_impact_test_metric": 0.5})
	// Append KPI metrics to the first resource metrics.
	if metrics.ResourceMetrics().Len() > 0 && kpi.ResourceMetrics().Len() > 0 {
		src := kpi.ResourceMetrics().At(0)
		dst := metrics.ResourceMetrics().At(0)
		for i := 0; i < src.ScopeMetrics().Len(); i++ {
			sm := dst.ScopeMetrics().AppendEmpty()
			src.ScopeMetrics().At(i).CopyTo(sm)
		}
	}

	// Execute pipeline.
	err = taggerProc.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	processed := sink.AllMetrics()
	require.NotEmpty(t, processed)
	// Ensure a priority attribute was added by the priority tagger.
	rm := processed[0].ResourceMetrics().At(0)
	val, ok := rm.Resource().Attributes().Get("aemf.process.priority")
	assert.True(t, ok)
	assert.Equal(t, "low", val.Str())
}
