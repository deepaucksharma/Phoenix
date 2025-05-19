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
