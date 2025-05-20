package adaptive_topk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
	processors "github.com/deepaucksharma/Phoenix/test/processors/templates"
)

func TestAdaptiveTopkConfigValidate(t *testing.T) {
	cfg := &adaptive_topk.Config{
		BaseConfig:    adaptive_topk.NewFactory().CreateDefaultConfig().(*adaptive_topk.Config).BaseConfig,
		KValue:        5,
		KMin:          1,
		KMax:          4,
		ResourceField: "process.name",
		CounterField:  "cpu",
	}
	assert.Error(t, cfg.Validate())

	cfg.KMax = 10
	cfg.KValue = 5
	assert.NoError(t, cfg.Validate())
}

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

func TestAdaptiveTopkProcessor(t *testing.T) {
	factory := adaptive_topk.NewFactory()
	cfg := factory.CreateDefaultConfig().(*adaptive_topk.Config)
	cfg.KValue = 2
	cfg.Enabled = true

	sink := new(consumertest.MetricsSink)
	ctx := context.Background()
	settings := processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}, ID: component.NewIDWithName(component.MustNewType("adaptive_topk"), "")}

	proc, err := factory.CreateMetrics(ctx, settings, cfg, sink)
	require.NoError(t, err)
	up, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)

	require.NoError(t, proc.Start(ctx, nil))
	metrics := generateTestMetrics()
	require.NoError(t, proc.ConsumeMetrics(ctx, metrics))

	processed := sink.AllMetrics()
	require.Len(t, processed, 1)
	assert.Equal(t, 2, processed[0].ResourceMetrics().Len())

	patch := interfaces.ConfigPatch{
		PatchID:             "set-k",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("adaptive_topk"), ""),
		ParameterPath:       "k_value",
		NewValue:            3,
	}
	require.NoError(t, up.OnConfigPatch(ctx, patch))
	status, err := up.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, status.Parameters["k_value"])

	badPatch := interfaces.ConfigPatch{PatchID: "bad", TargetProcessorName: patch.TargetProcessorName, ParameterPath: "k_value", NewValue: 1000}
	assert.Error(t, up.OnConfigPatch(ctx, badPatch))

	require.NoError(t, proc.Shutdown(ctx))
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

func TestWithTestCases(t *testing.T) {
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

func generateTestMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	values := []struct {
		name string
		cpu  float64
	}{
		{"p1", 10},
		{"p2", 30},
		{"p3", 20},
	}
	for _, v := range values {
		rm := metrics.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("process.name", v.name)
		sm := rm.ScopeMetrics().AppendEmpty()
		m := sm.Metrics().AppendEmpty()
		m.SetName("process.cpu_seconds_total")
		dp := m.SetEmptySum().DataPoints().AppendEmpty()
		dp.SetDoubleValue(v.cpu)
	}
	return metrics
}