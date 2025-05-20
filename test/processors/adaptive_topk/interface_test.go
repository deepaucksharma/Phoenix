package adaptive_topk

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	atp "github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
)

func TestUpdateableProcessorInterface(t *testing.T) {
	ctx := context.Background()
	factory := atp.NewFactory()
	cfg := factory.CreateDefaultConfig().(*atp.Config)

	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(ctx, processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}, ID: component.NewIDWithName(component.MustNewType("adaptive_topk"), "")}, cfg, sink)
	require.NoError(t, err)
	up, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)
	require.NoError(t, proc.Start(ctx, nil))

	// valid k_value patch
	patch := interfaces.ConfigPatch{
		PatchID:             "k_patch",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("adaptive_topk"), ""),
		ParameterPath:       "k_value",
		NewValue:            cfg.KMin,
		Timestamp:           time.Now().Unix(),
	}
	require.NoError(t, up.OnConfigPatch(ctx, patch))
	status, err := up.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, cfg.KMin, status.Parameters["k_value"])

	// invalid patch
	bad := interfaces.ConfigPatch{
		PatchID:             "bad",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("adaptive_topk"), ""),
		ParameterPath:       "k_value",
		NewValue:            cfg.KMax + 1,
		Timestamp:           time.Now().Unix(),
	}
	assert.Error(t, up.OnConfigPatch(ctx, bad))

	// enable/disable
	enablePatch := interfaces.ConfigPatch{
		PatchID:             "disable",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("adaptive_topk"), ""),
		ParameterPath:       "enabled",
		NewValue:            false,
		Timestamp:           time.Now().Unix(),
	}
	require.NoError(t, up.OnConfigPatch(ctx, enablePatch))
	status, err = up.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.False(t, status.Enabled)

	require.NoError(t, proc.Shutdown(ctx))
}
