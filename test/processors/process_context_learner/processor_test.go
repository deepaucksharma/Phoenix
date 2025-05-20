package process_context_learner

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

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	learner "github.com/deepaucksharma/Phoenix/internal/processor/process_context_learner"
)

func TestProcessContextLearnerConfigValidate(t *testing.T) {
	cfg := &learner.Config{DampingFactor: 0, Iterations: 0}
	assert.Error(t, cfg.Validate())

	cfg.DampingFactor = 0.5
	cfg.Iterations = 3
	assert.NoError(t, cfg.Validate())
}

func TestProcessContextLearnerOnPatch(t *testing.T) {
	factory := learner.NewFactory()
	cfg := factory.CreateDefaultConfig().(*learner.Config)
	ctx := context.Background()
	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(ctx, processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}}, cfg, sink)
	require.NoError(t, err)
	up, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)

	patch := interfaces.ConfigPatch{
		PatchID:             "damp",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("process_context_learner"), ""),
		ParameterPath:       "damping_factor",
		NewValue:            0.6,
	}
	require.NoError(t, up.OnConfigPatch(ctx, patch))
	status, err := up.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0.6, status.Parameters["damping_factor"])
}

func TestProcessContextLearner(t *testing.T) {
	factory := learner.NewFactory()
	cfg := factory.CreateDefaultConfig().(*learner.Config)
	cfg.Enabled = true

	sink := new(consumertest.MetricsSink)
	ctx := context.Background()
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewIDWithName(component.MustNewType("process_context_learner"), ""),
	}

	proc, err := factory.CreateMetrics(ctx, settings, cfg, sink)
	require.NoError(t, err)

	err = proc.Start(ctx, nil)
	require.NoError(t, err)

	metrics := generateTestMetrics()
	err = proc.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	processed := sink.AllMetrics()
	require.Len(t, processed, 1)

	rm := processed[0].ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		res := rm.At(i).Resource()
		_, ok := res.Attributes().Get("aemf.process.importance")
		assert.True(t, ok, "importance attribute missing")
	}

	lp := proc.(*learner.ProcessorImpl)
	scores := lp.GetScores()
	require.GreaterOrEqual(t, len(scores), 4)

	assert.Greater(t, scores[1], scores[2])
	assert.Greater(t, scores[2], scores[3])
	assert.Greater(t, scores[2], scores[3])

	err = proc.Shutdown(ctx)
	require.NoError(t, err)
}

func generateTestMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	relationships := []struct{ pid, ppid int }{
		{1, 0},
		{2, 1},
		{3, 2},
		{4, 1},
	}

	for _, r := range relationships {
		rm := metrics.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutInt("process.pid", int64(r.pid))
		rm.Resource().Attributes().PutInt("process.parent_pid", int64(r.ppid))

		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test.scope")

		metric := sm.Metrics().AppendEmpty()
		metric.SetName("test.metric")
		metric.SetEmptyGauge()
		dp := metric.Gauge().DataPoints().AppendEmpty()
		dp.SetIntValue(100)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}

	return metrics
}
