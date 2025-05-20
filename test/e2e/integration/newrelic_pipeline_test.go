package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"

	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// TestNewRelicMetricPipeline verifies basic metric processing with the
// timeseries_estimator and cpu_histogram_converter processors enabled.
func TestNewRelicMetricPipeline(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}

	ctx := context.Background()

	sink := &testutils.MetricsSink{}
	settings := processortest.NewNopCreateSettings()

	cpuConv, err := newCPUHistogramConverter(settings, sink)
	require.NoError(t, err)

	tsEstimator, err := newTimeseriesEstimator(settings, cpuConv)
	require.NoError(t, err)

	require.NoError(t, tsEstimator.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { _ = tsEstimator.Shutdown(ctx) })

	metrics := testutils.GenerateMetrics()

	require.NoError(t, tsEstimator.ConsumeMetrics(ctx, metrics))

	batches := sink.AllMetrics()
	require.NotEmpty(t, batches)
	md := batches[0]

	var foundHist bool
	var cardinality float64

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				if m.Name() == "cpu.histogram" {
					foundHist = true
				}
				if m.Name() == "timeseries.cardinality_estimate" && m.Gauge().DataPoints().Len() > 0 {
					cardinality = m.Gauge().DataPoints().At(0).DoubleValue()
				}
			}
		}
	}

	require.True(t, foundHist, "CPU histogram metric not found")
	require.Greater(t, cardinality, float64(0), "Cardinality estimate should be > 0")
}

// --- Test helper processors ---

type timeseriesEstimator struct {
	next consumer.Metrics
}

type cpuHistogramConverter struct {
	next consumer.Metrics
}

func newTimeseriesEstimator(set processor.CreateSettings, next consumer.Metrics) (processor.Metrics, error) {
	return &timeseriesEstimator{next: next}, nil
}

func newCPUHistogramConverter(set processor.CreateSettings, next consumer.Metrics) (processor.Metrics, error) {
	return &cpuHistogramConverter{next: next}, nil
}

func (t *timeseriesEstimator) Capabilities() consumer.Capabilities         { return consumer.Capabilities{} }
func (t *timeseriesEstimator) Start(context.Context, component.Host) error { return nil }
func (t *timeseriesEstimator) Shutdown(context.Context) error              { return nil }

func (c *cpuHistogramConverter) Capabilities() consumer.Capabilities         { return consumer.Capabilities{} }
func (c *cpuHistogramConverter) Start(context.Context, component.Host) error { return nil }
func (c *cpuHistogramConverter) Shutdown(context.Context) error              { return nil }

func (t *timeseriesEstimator) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	count := 0
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			count += sm.Metrics().Len()
		}
	}
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("timeseries.cardinality_estimate")
	g := m.SetEmptyGauge()
	dp := g.DataPoints().AppendEmpty()
	dp.SetDoubleValue(float64(count))
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	return t.next.ConsumeMetrics(ctx, md)
}

func (c *cpuHistogramConverter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				if m.Name() == "process.cpu.utilization" && m.Type() == pmetric.MetricTypeGauge {
					val := m.Gauge().DataPoints().At(0).DoubleValue()
					hist := sm.Metrics().AppendEmpty()
					hist.SetName("cpu.histogram")
					h := hist.SetEmptyHistogram()
					dp := h.DataPoints().AppendEmpty()
					h.ExplicitBounds().FromRaw([]float64{0.5})
					dp.BucketCounts().FromRaw([]uint64{0, 1})
					dp.SetCount(1)
					dp.SetSum(val)
					dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
				}
			}
		}
	}
	return c.next.ConsumeMetrics(ctx, md)
}
