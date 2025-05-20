package timeseries_estimator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/processor/timeseries_estimator"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

func TestTimeseriesEstimatorFactory(t *testing.T) {
	factory := timeseries_estimator.NewFactory()
	assert.NotNil(t, factory)
	cfg := factory.CreateDefaultConfig()
	assert.IsType(t, &timeseries_estimator.Config{}, cfg)
}

func TestTimeseriesEstimatorHighCardinality(t *testing.T) {
	factory := timeseries_estimator.NewFactory()
	cfg := factory.CreateDefaultConfig().(*timeseries_estimator.Config)

	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(context.Background(), processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
	}, cfg, sink)
	require.NoError(t, err)

	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	md := testutils.GenerateHighCardinalityMetrics(50)
	err = proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	out := sink.AllMetrics()
	require.NotEmpty(t, out)

	found := false
	metrics := out[0]
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				if sm.Metrics().At(k).Name() == "phoenix.timeseries.estimate" {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "expected estimate metric")

	assert.NoError(t, proc.Shutdown(context.Background()))
}

func TestTimeseriesEstimatorEmptyInput(t *testing.T) {
	factory := timeseries_estimator.NewFactory()
	cfg := factory.CreateDefaultConfig().(*timeseries_estimator.Config)

	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(context.Background(), processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
	}, cfg, sink)
	require.NoError(t, err)

	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	md := pmetric.NewMetrics()
	err = proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	out := sink.AllMetrics()
	require.NotEmpty(t, out)
	assert.NoError(t, proc.Shutdown(context.Background()))
}
