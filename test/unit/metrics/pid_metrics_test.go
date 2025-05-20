package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

// TestShouldEmit verifies the ShouldEmit logic.
func TestShouldEmit(t *testing.T) {
	m := metrics.NewPIDMetrics("test", nil)
	m.SetEmitInterval(time.Minute)

	// Last emission just happened; should not emit.
	m.LastEmission = time.Now()
	assert.False(t, m.ShouldEmit())

	// Last emission long ago; should emit.
	m.LastEmission = time.Now().Add(-2 * time.Minute)
	assert.True(t, m.ShouldEmit())
}

// TestEmitMetrics ensures metrics are produced and forwarded to the parent emitter.
func TestEmitMetrics(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("pid")
	parent := metrics.NewMetricsEmitter(meter, "parent", component.MustNewID("test"))

	metrics.ResetCapturedMetrics()

	p := metrics.NewPIDMetrics("ctrl", parent)
	p.Update(1, 2, 3, 4, 5, 6, 7, 8)

	before := time.Now()
	out := p.EmitMetrics(context.Background())

	// Verify last emission updated
	assert.True(t, p.LastEmission.After(before) || p.LastEmission.Equal(before))

	// Verify metric names
	rm := out.ResourceMetrics().At(0)
	sm := rm.ScopeMetrics().At(0)
	require.Equal(t, 8, sm.Metrics().Len())
	expected := []string{
		"aemf.controller.pid.error",
		"aemf.controller.pid.p_term",
		"aemf.controller.pid.i_term",
		"aemf.controller.pid.d_term",
		"aemf.controller.pid.raw_output",
		"aemf.controller.pid.output",
		"aemf.controller.pid.setpoint",
		"aemf.controller.pid.measurement",
	}
	for i, name := range expected {
		assert.Equal(t, name, sm.Metrics().At(i).Name())
	}

	// Metrics should be forwarded to parent emitter
	captured := metrics.MetricsFor(parent)
	require.Len(t, captured, 1)
	assert.Equal(t, out, captured[0])
}
