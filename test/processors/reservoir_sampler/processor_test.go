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
	"github.com/deepaucksharma/Phoenix/test/testutils"
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

func TestReservoirSamplerProcessor_Sampling(t *testing.T) {
	factory := reservoir_sampler.NewFactory()
	cfg := factory.CreateDefaultConfig().(*reservoir_sampler.Config)
	cfg.Enabled = true
	cfg.ReservoirSize = 5

	sink := new(consumertest.MetricsSink)
	ctx := context.Background()
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewIDWithName(component.MustNewType("reservoir_sampler"), ""),
	}

	proc, err := factory.CreateMetrics(ctx, settings, cfg, sink)
	require.NoError(t, err)

	up, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)

	require.NoError(t, proc.Start(ctx, nil))

	metrics := testutils.GenerateTestMetrics(20)
	require.NoError(t, proc.ConsumeMetrics(ctx, metrics))

	processed := sink.AllMetrics()
	require.NotEmpty(t, processed)
	assert.LessOrEqual(t, processed[0].ResourceMetrics().Len(), cfg.ReservoirSize)

	require.NoError(t, proc.Shutdown(ctx))
	_, err = up.GetConfigStatus(ctx)
	require.NoError(t, err)
}

func TestReservoirSamplerProcessor_ConfigPatches(t *testing.T) {
	factory := reservoir_sampler.NewFactory()
	cfg := factory.CreateDefaultConfig().(*reservoir_sampler.Config)
	cfg.Enabled = true

	sink := new(consumertest.MetricsSink)
	ctx := context.Background()
	settings := processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}, ID: component.NewIDWithName(component.MustNewType("reservoir_sampler"), "")}

	proc, err := factory.CreateMetrics(ctx, settings, cfg, sink)
	require.NoError(t, err)
	up, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)
	require.NoError(t, proc.Start(ctx, nil))

	patchSize := interfaces.ConfigPatch{
		PatchID:             "resize",
		TargetProcessorName: settings.ID,
		ParameterPath:       "reservoir_size",
		NewValue:            20,
	}
	require.NoError(t, up.OnConfigPatch(ctx, patchSize))

	patchDisable := interfaces.ConfigPatch{
		PatchID:             "disable",
		TargetProcessorName: settings.ID,
		ParameterPath:       "enabled",
		NewValue:            false,
	}
	require.NoError(t, up.OnConfigPatch(ctx, patchDisable))

	status, err := up.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, 20, status.Parameters["reservoir_size"])
	assert.False(t, status.Enabled)

	sink.Reset()
	metrics := testutils.GenerateTestMetrics(5)
	require.NoError(t, proc.ConsumeMetrics(ctx, metrics))
	processed := sink.AllMetrics()
	require.NotEmpty(t, processed)
	assert.Equal(t, metrics.ResourceMetrics().Len(), processed[0].ResourceMetrics().Len())

	require.NoError(t, proc.Shutdown(ctx))
}

func TestReservoirSamplerProcessor_PIDResize(t *testing.T) {
	factory := reservoir_sampler.NewFactory()
	cfg := factory.CreateDefaultConfig().(*reservoir_sampler.Config)
	cfg.Enabled = true
	cfg.ReservoirSize = 10

	sink := new(consumertest.MetricsSink)
	ctx := context.Background()
	settings := processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}, ID: component.NewIDWithName(component.MustNewType("reservoir_sampler"), "")}

	proc, err := factory.CreateMetrics(ctx, settings, cfg, sink)
	require.NoError(t, err)
	up, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)
	require.NoError(t, proc.Start(ctx, nil))

	metrics := testutils.GenerateTestMetrics(100)
	require.NoError(t, proc.ConsumeMetrics(ctx, metrics))

	status, err := up.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Greater(t, status.Parameters["reservoir_size"].(int), 10)

	require.NoError(t, proc.Shutdown(ctx))
}
