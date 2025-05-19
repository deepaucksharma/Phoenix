package cardinality_guardian

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	cg "github.com/deepaucksharma/Phoenix/internal/processor/cardinality_guardian"
	testutils "github.com/deepaucksharma/Phoenix/test/testutils"
)

func TestCardinalityReduction(t *testing.T) {
	factory := cg.NewFactory()
	cfg := factory.CreateDefaultConfig().(*cg.Config)
	cfg.MaxUnique = 50

	sink := new(consumertest.MetricsSink)
	ctx := context.Background()
	settings := processor.CreateSettings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewID(typeStr),
	}

	proc, err := factory.CreateMetricsProcessor(ctx, settings, cfg, sink)
	require.NoError(t, err)

	metrics := testutils.GenerateHighCardinalityMetrics(200)
	err = proc.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	out := sink.AllMetrics()
	require.Len(t, out, 1)

	unique := make(map[int64]struct{})
	rmSlice := out[0].ResourceMetrics()
	for i := 0; i < rmSlice.Len(); i++ {
		rm := rmSlice.At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				if m.Name() != "test.metric" {
					continue
				}
				dps := m.Gauge().DataPoints()
				for d := 0; d < dps.Len(); d++ {
					dp := dps.At(d)
					if v, ok := dp.Attributes().Get("cg_bucket"); ok {
						unique[v.Int()] = struct{}{}
					}
				}
			}
		}
	}

	assert.LessOrEqual(t, len(unique), cfg.MaxUnique)
}

func TestConfigPatching(t *testing.T) {
	factory := cg.NewFactory()
	cfg := factory.CreateDefaultConfig().(*cg.Config)
	sink := new(consumertest.MetricsSink)
	ctx := context.Background()
	settings := processor.CreateSettings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}, ID: component.NewID(typeStr)}

	proc, err := factory.CreateMetricsProcessor(ctx, settings, cfg, sink)
	require.NoError(t, err)

	up := proc.(interfaces.UpdateableProcessor)

	patch := interfaces.ConfigPatch{
		PatchID:             uuid.NewString(),
		TargetProcessorName: component.NewID(typeStr),
		ParameterPath:       "max_unique",
		NewValue:            200,
	}

	err = up.OnConfigPatch(ctx, patch)
	require.NoError(t, err)

	status, err := up.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, 200, status.Parameters["max_unique"])

	badPatch := interfaces.ConfigPatch{
		PatchID:             uuid.NewString(),
		TargetProcessorName: component.NewID(typeStr),
		ParameterPath:       "max_unique",
		NewValue:            0,
	}
	assert.Error(t, up.OnConfigPatch(ctx, badPatch))
}
