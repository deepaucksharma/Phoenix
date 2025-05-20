package reservoir_sampler

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
	"github.com/deepaucksharma/Phoenix/internal/processor/reservoir_sampler"
	processors "github.com/deepaucksharma/Phoenix/test/processors/templates"
)

func TestValidate(t *testing.T) {
	cfg := &reservoir_sampler.Config{ReservoirSize: 10, Enabled: true}
	assert.NoError(t, cfg.Validate())

	cfg.ReservoirSize = 0
	assert.Error(t, cfg.Validate())
}

func TestOnConfigPatchInvalid(t *testing.T) {
	factory := reservoir_sampler.NewFactory()
	cfg := factory.CreateDefaultConfig().(*reservoir_sampler.Config)
	cfg.Enabled = true

	ctx := context.Background()
	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(ctx, processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewIDWithName(component.MustNewType("reservoir_sampler"), ""),
	}, cfg, sink)
	require.NoError(t, err)

	up, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)

	err = proc.Start(ctx, nil)
	require.NoError(t, err)

	badPatch := interfaces.ConfigPatch{
		PatchID:             "bad-type",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("reservoir_sampler"), ""),
		ParameterPath:       "reservoir_size",
		NewValue:            "invalid",
	}
	assert.Error(t, up.OnConfigPatch(ctx, badPatch))

	require.NoError(t, proc.Shutdown(ctx))
}

func TestReservoirSamplerProcessor(t *testing.T) {
	factory := reservoir_sampler.NewFactory()
	cfg := factory.CreateDefaultConfig().(*reservoir_sampler.Config)
	cfg.Enabled = true

	testCases := []processors.ProcessorTestCase{
		{
			Name:         "PassThrough",
			InputMetrics: processors.GenerateTestMetrics([]string{"p1", "p2"}),
			ExpectedOutput: func(md pmetric.Metrics) bool {
				return md.ResourceMetrics().Len() > 0
			},
		},
	}

	processors.RunProcessorTests(t, factory, cfg, testCases)
}
