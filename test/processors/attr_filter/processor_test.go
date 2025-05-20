package attr_filter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/processor/attr_filter"
)

func createProcessor(t *testing.T, attrs []string, sink *consumertest.MetricsSink) processor.Metrics {
	factory := attr_filter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*attr_filter.Config)
	cfg.Attributes = attrs

	proc, err := factory.CreateMetrics(
		context.Background(),
		processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}},
		cfg,
		sink,
	)
	require.NoError(t, err)
	require.NotNil(t, proc)
	return proc
}

func generateMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	attrs := rm.Resource().Attributes()
	attrs.PutStr("process.pid", "1234")
	attrs.PutStr("container.id", "abcd")
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("test")
	m := sm.Metrics().AppendEmpty()
	m.SetName("test.metric")
	m.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(1.0)
	return md
}

func TestAttrFilter_RemoveProcessPID(t *testing.T) {
	sink := new(consumertest.MetricsSink)
	proc := createProcessor(t, []string{"process.pid"}, sink)
	require.NoError(t, proc.Start(context.Background(), nil))

	md := generateMetrics()
	err := proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	out := sink.AllMetrics()[0]
	rm := out.ResourceMetrics().At(0)
	_, exists := rm.Resource().Attributes().Get("process.pid")
	assert.False(t, exists)
	_, exists = rm.Resource().Attributes().Get("container.id")
	assert.True(t, exists)

	require.NoError(t, proc.Shutdown(context.Background()))
}

func TestAttrFilter_RemoveContainerID(t *testing.T) {
	sink := new(consumertest.MetricsSink)
	proc := createProcessor(t, []string{"container.id"}, sink)
	require.NoError(t, proc.Start(context.Background(), nil))

	md := generateMetrics()
	err := proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	out := sink.AllMetrics()[0]
	rm := out.ResourceMetrics().At(0)
	_, exists := rm.Resource().Attributes().Get("container.id")
	assert.False(t, exists)
	_, exists = rm.Resource().Attributes().Get("process.pid")
	assert.True(t, exists)

	require.NoError(t, proc.Shutdown(context.Background()))
}
