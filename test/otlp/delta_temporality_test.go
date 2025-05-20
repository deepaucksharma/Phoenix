package otlp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/cumulativetodeltaprocessor"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestCumulativeToDeltaTemporality(t *testing.T) {
	ctx := context.Background()
	sink := new(consumertest.MetricsSink)

	factory := cumulativetodeltaprocessor.NewFactory()
	cfg := factory.CreateDefaultConfig().(*cumulativetodeltaprocessor.Config)
	cfg.Include.Metrics = []string{"process.cpu.time", "process.io.read_bytes", "process.io.write_bytes"}

	proc, err := factory.CreateMetricsProcessor(ctx, processortest.NewNopCreateSettings(), cfg, sink)
	require.NoError(t, err)

	require.NoError(t, proc.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { _ = proc.Shutdown(ctx) })

	md := generateCumulativeMetrics()
	require.NoError(t, proc.ConsumeMetrics(ctx, md))

	batches := sink.AllMetrics()
	require.Len(t, batches, 1)

	rm := batches[0].ResourceMetrics().At(0)
	sm := rm.ScopeMetrics().At(0)

	require.Equal(t, 3, sm.Metrics().Len())
	for i := 0; i < sm.Metrics().Len(); i++ {
		m := sm.Metrics().At(i)
		require.Equal(t, pmetric.MetricTypeSum, m.Type())
		require.Equal(t, pmetric.AggregationTemporalityDelta, m.Sum().AggregationTemporality())
	}
}

func generateCumulativeMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()

	addMetric := func(name string, value float64) {
		m := sm.Metrics().AppendEmpty()
		m.SetName(name)
		sum := m.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		dp := sum.DataPoints().AppendEmpty()
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		dp.SetDoubleValue(value)
	}

	addMetric("process.cpu.time", 1.0)
	addMetric("process.io.read_bytes", 1024)
	addMetric("process.io.write_bytes", 2048)

	return md
}
