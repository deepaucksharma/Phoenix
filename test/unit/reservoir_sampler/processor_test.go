package reservoir_sampler

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
    iftest "github.com/deepaucksharma/Phoenix/test/interfaces"
)

func TestReservoirSamplerProcessor(t *testing.T) {
    factory := NewFactory()
    cfg := factory.CreateDefaultConfig().(*Config)
    cfg.ReservoirSize = 5
    cfg.Enabled = true

    sink := new(consumertest.MetricsSink)

    ctx := context.Background()
    settings := processor.CreateSettings{
        TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
        ID: component.NewID("reservoir_sampler"),
    }

    proc, err := factory.CreateMetricsProcessor(ctx, settings, cfg, sink)
    require.NoError(t, err)
    require.NotNil(t, proc)

    updateable, ok := proc.(interfaces.UpdateableProcessor)
    require.True(t, ok)

    require.NoError(t, proc.Start(ctx, nil))

    suite := iftest.UpdateableProcessorSuite{
        Processor: updateable,
        ValidPatches: []iftest.TestPatch{
            {
                Name: "UpdateSize",
                Patch: interfaces.ConfigPatch{
                    PatchID: "size",
                    TargetProcessorName: component.NewID("reservoir_sampler"),
                    ParameterPath: "reservoir_size",
                    NewValue: 10,
                },
                ExpectedValue: 10,
            },
            {
                Name: "ToggleEnabled",
                Patch: interfaces.ConfigPatch{
                    PatchID: "en",
                    TargetProcessorName: component.NewID("reservoir_sampler"),
                    ParameterPath: "enabled",
                    NewValue: false,
                },
                ExpectedValue: false,
            },
        },
        InvalidPatches: []iftest.TestPatch{
            {
                Name: "InvalidSize",
                Patch: interfaces.ConfigPatch{
                    PatchID: "badsize",
                    TargetProcessorName: component.NewID("reservoir_sampler"),
                    ParameterPath: "reservoir_size",
                    NewValue: -1,
                },
            },
        },
    }
    iftest.RunUpdateableProcessorTests(t, suite)

    t.Run("ProcessMetrics", func(t *testing.T) {
        metrics := generateTestMetrics()
        enablePatch := interfaces.ConfigPatch{
            PatchID: "enable",
            TargetProcessorName: component.NewID("reservoir_sampler"),
            ParameterPath: "enabled",
            NewValue: true,
        }
        require.NoError(t, updateable.OnConfigPatch(ctx, enablePatch))

        err = proc.ConsumeMetrics(ctx, metrics)
        require.NoError(t, err)

        processed := sink.AllMetrics()
        require.NotEmpty(t, processed)
        count := processed[0].ResourceMetrics().Len()
        assert.LessOrEqual(t, count, cfg.ReservoirSize)
    })

    require.NoError(t, proc.Shutdown(ctx))
}

func generateTestMetrics() pmetric.Metrics {
    metrics := pmetric.NewMetrics()
    for i := 0; i < 20; i++ {
        rm := metrics.ResourceMetrics().AppendEmpty()
        rm.Resource().Attributes().PutStr("process.name", "proc")
        sm := rm.ScopeMetrics().AppendEmpty()
        sm.Scope().SetName("test")
        m := sm.Metrics().AppendEmpty()
        m.SetName("test.metric")
        m.SetEmptyGauge()
        dp := m.Gauge().DataPoints().AppendEmpty()
        dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
        dp.SetDoubleValue(float64(i))
    }
    return metrics
}
