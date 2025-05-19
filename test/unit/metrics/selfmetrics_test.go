package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

type capturingMeter struct {
	noop.Meter
	lastName string
}

func (m *capturingMeter) Int64Counter(name string, opts ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	m.lastName = name
	return noop.Int64Counter{}, nil
}

func (m *capturingMeter) Float64Gauge(name string, opts ...metric.Float64GaugeOption) (metric.Float64Gauge, error) {
	m.lastName = name
	return noop.Float64Gauge{}, nil
}

func TestRegisterCounterAndGauge(t *testing.T) {
	meter := &capturingMeter{}
	emitter := metrics.NewMetricsEmitter(meter, "proc", component.MustNewID("test"))

	_, err := emitter.RegisterCounter("hits", "hit count")
	require.NoError(t, err)
	assert.Equal(t, "aemf_proc_hits", meter.lastName)

	_, err = emitter.RegisterGauge("load", "load gauge")
	require.NoError(t, err)
	assert.Equal(t, "aemf_proc_load", meter.lastName)
}

func TestCreatePatchMetricAttributes(t *testing.T) {
	patch := &interfaces.ConfigPatch{
		PatchID:             "id1",
		TargetProcessorName: component.MustNewID("adaptive_topk"),
		ParameterPath:       "k_value",
		NewValue:            42,
		Reason:              "test",
		Severity:            "normal",
		Source:              "manual",
		Timestamp:           1234567890,
	}

	m := metrics.CreatePatchMetric(patch)
	rm := m.ResourceMetrics().At(0)
	sm := rm.ScopeMetrics().At(0)
	metric := sm.Metrics().At(0)

	assert.Equal(t, "aemf_ctrl_proposed_patch", metric.Name())
	dp := metric.Gauge().DataPoints().At(0)
	attrs := dp.Attributes()

	val, ok := attrs.Get("patch_id")
	require.True(t, ok)
	assert.Equal(t, "id1", val.AsString())
	val, _ = attrs.Get("target_processor_name")
	assert.Equal(t, "adaptive_topk", val.AsString())
	val, _ = attrs.Get("parameter_path")
	assert.Equal(t, "k_value", val.AsString())
	val, _ = attrs.Get("reason")
	assert.Equal(t, "test", val.AsString())
	val, _ = attrs.Get("severity")
	assert.Equal(t, "normal", val.AsString())
	val, _ = attrs.Get("source")
	assert.Equal(t, "manual", val.AsString())
	val, _ = attrs.Get("new_value_int")
	assert.Equal(t, int64(42), val.Int())
}
