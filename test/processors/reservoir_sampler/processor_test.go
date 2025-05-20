package reservoir_sampler

// Unit tests for the reservoir_sampler processor checking sampling limits, PID adjustments, and config patches.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/reservoir_sampler"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

func TestReservoirSamplerConfigValidate(t *testing.T) {
	cfg := &reservoir_sampler.Config{ReservoirSize: 0}
	assert.Error(t, cfg.Validate())

	cfg.ReservoirSize = 5
	assert.NoError(t, cfg.Validate())
}

func TestReservoirSamplerProcessor(t *testing.T) {
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
	require.NotNil(t, proc)

	updateableProc, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)

	err = proc.Start(ctx, nil)
	require.NoError(t, err)

	metrics := testutils.GenerateTestMetrics(20)
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		if i%2 == 0 {
			rm.Resource().Attributes().PutStr("aemf.process.priority", "high")
		} else {
			rm.Resource().Attributes().PutStr("aemf.process.priority", "low")
		}
	}

	err = proc.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	processed := sink.AllMetrics()
	require.Len(t, processed, 1)

	sampleCount := processed[0].ResourceMetrics().Len()
	assert.Equal(t, 10, sampleCount)

	status, err := updateableProc.GetConfigStatus(ctx)
	require.NoError(t, err)
	sizeAfterIncrease := status.Parameters["reservoir_size"].(int)
	assert.Greater(t, sizeAfterIncrease, 5)

	sink.Reset()

	metricsFew := testutils.GenerateTestMetrics(8)
	for i := 0; i < metricsFew.ResourceMetrics().Len(); i++ {
		rm := metricsFew.ResourceMetrics().At(i)
		if i%2 == 0 {
			rm.Resource().Attributes().PutStr("aemf.process.priority", "high")
		} else {
			rm.Resource().Attributes().PutStr("aemf.process.priority", "low")
		}
	}

	err = proc.ConsumeMetrics(ctx, metricsFew)
	require.NoError(t, err)

	status, err = updateableProc.GetConfigStatus(ctx)
	require.NoError(t, err)
	sizeAfterDecrease := status.Parameters["reservoir_size"].(int)
	assert.Less(t, sizeAfterDecrease, sizeAfterIncrease)

	patchSize := interfaces.ConfigPatch{
		PatchID:             "set-size",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("reservoir_sampler"), ""),
		ParameterPath:       "reservoir_size",
		NewValue:            12,
	}
	err = updateableProc.OnConfigPatch(ctx, patchSize)
	require.NoError(t, err)

	status, err = updateableProc.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, 12, status.Parameters["reservoir_size"])

	patchEnable := interfaces.ConfigPatch{
		PatchID:             "toggle-enable",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("reservoir_sampler"), ""),
		ParameterPath:       "enabled",
		NewValue:            false,
	}
	err = updateableProc.OnConfigPatch(ctx, patchEnable)
	require.NoError(t, err)

	status, err = updateableProc.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.False(t, status.Enabled)

	err = proc.Shutdown(ctx)
	require.NoError(t, err)
}
