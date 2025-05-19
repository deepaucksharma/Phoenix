package others_rollup

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
	"github.com/deepaucksharma/Phoenix/internal/processor/others_rollup"
)

func TestOthersRollupProcessor(t *testing.T) {
	factory := others_rollup.NewFactory()
	cfg := factory.CreateDefaultConfig().(*others_rollup.Config)
	cfg.Enabled = true
	cfg.Strategy = "sum"

	sink := new(consumertest.MetricsSink)
	ctx := context.Background()
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewIDWithName(component.MustNewType("others_rollup"), ""),
	}

	proc, err := factory.CreateMetrics(ctx, settings, cfg, sink)
	require.NoError(t, err)
	updateable, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)

	err = proc.Start(ctx, nil)
	require.NoError(t, err)

	t.Run("SumStrategy", func(t *testing.T) {
		metrics := generateTestMetrics()
		err := proc.ConsumeMetrics(ctx, metrics)
		require.NoError(t, err)

		processed := sink.AllMetrics()
		require.Len(t, processed, 1)
		rm := processed[0].ResourceMetrics()
		require.Equal(t, 2, rm.Len())

		foundOthers := false
		for i := 0; i < rm.Len(); i++ {
			res := rm.At(i).Resource()
			name, _ := res.Attributes().Get("process.name")
			if name.Str() == "others" {
				foundOthers = true
				sm := rm.At(i).ScopeMetrics().At(0)
				// cpu metric
				m := sm.Metrics().At(0)
				val := m.Gauge().DataPoints().At(0).DoubleValue()
				assert.Equal(t, 3.0, val)
				// requests metric
				m2 := sm.Metrics().At(1)
				val2 := m2.Sum().DataPoints().At(0).DoubleValue()
				assert.Equal(t, 30.0, val2)
			} else {
				// high priority resource should remain
				priority, _ := res.Attributes().Get("aemf.process.priority")
				assert.Equal(t, "high", priority.Str())
			}
		}
		assert.True(t, foundOthers, "others resource missing")
	})

	t.Run("PatchToAvg", func(t *testing.T) {
		patch := interfaces.ConfigPatch{
			PatchID:             "switch-avg",
			TargetProcessorName: component.NewIDWithName(component.MustNewType("others_rollup"), ""),
			ParameterPath:       "strategy",
			NewValue:            "avg",
		}
		err := updateable.OnConfigPatch(ctx, patch)
		require.NoError(t, err)

		sink.Reset()
		metrics := generateTestMetrics()
		err = proc.ConsumeMetrics(ctx, metrics)
		require.NoError(t, err)

		processed := sink.AllMetrics()
		require.Len(t, processed, 1)
		rm := processed[0].ResourceMetrics()
		require.Equal(t, 2, rm.Len())

		for i := 0; i < rm.Len(); i++ {
			res := rm.At(i).Resource()
			name, _ := res.Attributes().Get("process.name")
			if name.Str() == "others" {
				sm := rm.At(i).ScopeMetrics().At(0)
				m := sm.Metrics().At(0)
				val := m.Gauge().DataPoints().At(0).DoubleValue()
				assert.Equal(t, 1.5, val)
				m2 := sm.Metrics().At(1)
				val2 := m2.Sum().DataPoints().At(0).DoubleValue()
				assert.Equal(t, 15.0, val2)
			}
		}
		status, err := updateable.GetConfigStatus(ctx)
		require.NoError(t, err)
		assert.Equal(t, "avg", status.Parameters["strategy"])
	})

	err = proc.Shutdown(ctx)
	require.NoError(t, err)
}

func generateTestMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	now := pcommon.NewTimestampFromTime(testNow)
	resources := []struct {
		name     string
		priority string
		cpu      float64
		requests float64
	}{
		{"proc-low1", "low", 1.0, 10.0},
		{"proc-low2", "low", 2.0, 20.0},
		{"proc-high", "high", 3.0, 30.0},
	}

	for _, r := range resources {
		rm := metrics.ResourceMetrics().AppendEmpty()
		res := rm.Resource()
		res.Attributes().PutStr("process.name", r.name)
		res.Attributes().PutStr("aemf.process.priority", r.priority)

		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test.scope")

		m1 := sm.Metrics().AppendEmpty()
		m1.SetName("cpu")
		gauge := m1.SetEmptyGauge()
		dp1 := gauge.DataPoints().AppendEmpty()
		dp1.SetDoubleValue(r.cpu)
		dp1.SetTimestamp(now)

		m2 := sm.Metrics().AppendEmpty()
		m2.SetName("requests")
		sum := m2.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		dp2 := sum.DataPoints().AppendEmpty()
		dp2.SetDoubleValue(r.requests)
		dp2.SetTimestamp(now)
	}

	return metrics
}

var testNow = time.Now()
