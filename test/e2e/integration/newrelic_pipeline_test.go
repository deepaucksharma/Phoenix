package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/deepaucksharma/Phoenix/pkg/util/hll"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// cpuHistogramConverter converts CPU metrics to histogram form.
type cpuHistogramConverter struct {
	next consumer.Metrics
}

func newCPUHistogramConverter(next consumer.Metrics) *cpuHistogramConverter {
	return &cpuHistogramConverter{next: next}
}

func (c *cpuHistogramConverter) Start(context.Context, component.Host) error { return nil }
func (c *cpuHistogramConverter) Shutdown(context.Context) error              { return nil }
func (c *cpuHistogramConverter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (c *cpuHistogramConverter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				if m.Name() == "process.cpu_seconds_total" && m.Type() == pmetric.MetricTypeSum {
					if m.Sum().DataPoints().Len() == 0 {
						continue
					}
					val := m.Sum().DataPoints().At(0).DoubleValue()
					hist := sm.Metrics().AppendEmpty()
					hist.SetName("process.cpu_seconds_total_histogram")
					histogram := hist.SetEmptyHistogram()
					dp := histogram.DataPoints().AppendEmpty()
					dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
					histogram.BucketCounts().EnsureCapacity(2)
					dp.BucketCounts().FromRaw([]uint64{0, 1})
					dp.ExplicitBounds().FromRaw([]float64{val})
					dp.SetCount(1)
					dp.SetSum(val)
				}
			}
		}
	}
	return c.next.ConsumeMetrics(ctx, md)
}

// timeseriesEstimator estimates unique timeseries using HyperLogLog.
type timeseriesEstimator struct {
	next consumer.Metrics
	hll  *hll.HyperLogLog
}

func newTimeseriesEstimator(next consumer.Metrics) *timeseriesEstimator {
	return &timeseriesEstimator{next: next, hll: hll.NewDefault()}
}

func (t *timeseriesEstimator) Start(context.Context, component.Host) error { return nil }
func (t *timeseriesEstimator) Shutdown(context.Context) error              { return nil }
func (t *timeseriesEstimator) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (t *timeseriesEstimator) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Estimate cardinality
	t.hll = hll.NewDefault()
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		resName := ""
		if v, ok := rm.Resource().Attributes().Get("process.name"); ok {
			resName = v.Str()
		}
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				key := fmt.Sprintf("%s:%s", resName, m.Name())
				t.hll.Add([]byte(key))
			}
		}
	}
	estimate := t.hll.Count()

	if md.ResourceMetrics().Len() == 0 {
		md.ResourceMetrics().AppendEmpty()
	}
	rm := md.ResourceMetrics().At(0)
	if rm.ScopeMetrics().Len() == 0 {
		rm.ScopeMetrics().AppendEmpty()
	}
	sm := rm.ScopeMetrics().At(0)
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("estimated_active_timeseries")
	g := metric.SetEmptyGauge()
	dp := g.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetDoubleValue(float64(estimate))

	return t.next.ConsumeMetrics(ctx, md)
}

func TestNewRelicPipeline(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}

	ctx := context.Background()
	sink := new(consumertest.MetricsSink)

	conv := newCPUHistogramConverter(sink)
	est := newTimeseriesEstimator(conv)

	err := est.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer func() { assert.NoError(t, est.Shutdown(ctx)) }()

	md := testutils.GenerateTestMetrics(5)
	require.NoError(t, est.ConsumeMetrics(ctx, md))

	results := sink.AllMetrics()
	require.NotEmpty(t, results)

	var foundHistogram, foundEstimate bool
	for _, result := range results {
		for i := 0; i < result.ResourceMetrics().Len(); i++ {
			rm := result.ResourceMetrics().At(i)
			for j := 0; j < rm.ScopeMetrics().Len(); j++ {
				sm := rm.ScopeMetrics().At(j)
				for k := 0; k < sm.Metrics().Len(); k++ {
					m := sm.Metrics().At(k)
					switch m.Name() {
					case "process.cpu_seconds_total_histogram":
						foundHistogram = true
						assert.Equal(t, pmetric.MetricTypeHistogram, m.Type())
					case "estimated_active_timeseries":
						foundEstimate = true
						assert.Equal(t, pmetric.MetricTypeGauge, m.Type())
						assert.Greater(t, m.Gauge().DataPoints().At(0).DoubleValue(), float64(0))
					}
				}
			}
		}
	}

	assert.True(t, foundHistogram, "CPU histogram not found")
	assert.True(t, foundEstimate, "cardinality estimate not found")
}
