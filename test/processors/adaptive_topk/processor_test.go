package adaptive_topk

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
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
	atp "github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
	processors "github.com/deepaucksharma/Phoenix/test/processors/templates"
)

func TestValidate(t *testing.T) {
	cfg := &adaptive_topk.Config{
		BaseConfig:    nil,
		KValue:        20,
		KMin:          10,
		KMax:          30,
		ResourceField: "process.name",
		CounterField:  "process.cpu_seconds_total",
	}
	assert.NoError(t, cfg.Validate())

	cfg.KValue = 0
	assert.Error(t, cfg.Validate())
}

func TestOnConfigPatchInvalid(t *testing.T) {
	factory := adaptive_topk.NewFactory()
	cfg := factory.CreateDefaultConfig().(*adaptive_topk.Config)
	cfg.Enabled = true

	ctx := context.Background()
	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(ctx, processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewIDWithName(component.MustNewType("adaptive_topk"), ""),
	}, cfg, sink)
	require.NoError(t, err)

	up, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)

	err = proc.Start(ctx, nil)
	require.NoError(t, err)

	badPatch := interfaces.ConfigPatch{
		PatchID:             "bad-k",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("adaptive_topk"), ""),
		ParameterPath:       "k_value",
		NewValue:            "invalid",
	}
	assert.Error(t, up.OnConfigPatch(ctx, badPatch))

	require.NoError(t, proc.Shutdown(ctx))
}

func TestAdaptiveTopkProcessor(t *testing.T) {
	factory := adaptive_topk.NewFactory()
	cfg := factory.CreateDefaultConfig().(*adaptive_topk.Config)
	cfg.Enabled = true

	testCases := []processors.ProcessorTestCase{
		{
			Name:         "Basic",
			InputMetrics: processors.GenerateTestMetrics([]string{"p1", "p2"}),
			ExpectedOutput: func(md pmetric.Metrics) bool {
				return true
			},
		},
	}

	processors.RunProcessorTests(t, factory, cfg, testCases)
}

func TestProcessorFiltering(t *testing.T) {
	ctx := context.Background()
	factory := atp.NewFactory()
	cfg := factory.CreateDefaultConfig().(*atp.Config)
	cfg.KValue = 2
	cfg.KMin = 1
	cfg.KMax = 5
	cfg.Enabled = true

	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(ctx, processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}, ID: component.NewIDWithName(component.MustNewType("adaptive_topk"), "")}, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, proc.Start(ctx, nil))

	metrics := generateMetrics()
	err = proc.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	processed := sink.AllMetrics()
	require.Len(t, processed, 1)
	rms := processed[0].ResourceMetrics()
	require.Equal(t, 2, rms.Len())

	// ensure included attribute exists
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		_, ok := rm.Resource().Attributes().Get("aemf.topk.included")
		assert.True(t, ok)
	}

	coverage := float64(rms.Len()) / 4.0
	assert.InDelta(t, 0.5, coverage, 0.0001)

	require.NoError(t, proc.Shutdown(ctx))
}

func TestDynamicKPatch(t *testing.T) {
	ctx := context.Background()
	factory := atp.NewFactory()
	cfg := factory.CreateDefaultConfig().(*atp.Config)
	cfg.KValue = 2
	cfg.KMin = 1
	cfg.KMax = 5
	cfg.Enabled = true

	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(ctx, processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}, ID: component.NewIDWithName(component.MustNewType("adaptive_topk"), "")}, cfg, sink)
	require.NoError(t, err)
	up, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)
	require.NoError(t, proc.Start(ctx, nil))

	// patch k_value to 3
	patch := interfaces.ConfigPatch{
		PatchID:             "kupdate",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("adaptive_topk"), ""),
		ParameterPath:       "k_value",
		NewValue:            3,
	}
	require.NoError(t, up.OnConfigPatch(ctx, patch))
	status, err := up.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, status.Parameters["k_value"])

	metrics := generateMetrics()
	sink.Reset()
	require.NoError(t, proc.ConsumeMetrics(ctx, metrics))
	processed := sink.AllMetrics()[0]
	assert.Equal(t, 3, processed.ResourceMetrics().Len())

	require.NoError(t, proc.Shutdown(ctx))
}

func generateMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	cpuVals := []float64{100, 80, 60, 40}
	for i, val := range cpuVals {
		rm := metrics.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("process.name", "proc"+string(rune('A'+i)))

		sm := rm.ScopeMetrics().AppendEmpty()
		metric := sm.Metrics().AppendEmpty()
		metric.SetName("process.cpu_seconds_total")
		sum := metric.SetEmptySum()
		dp := sum.DataPoints().AppendEmpty()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		dp.SetDoubleValue(val)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(testNow))
	}
	return metrics
}

var testNow = time.Now()
