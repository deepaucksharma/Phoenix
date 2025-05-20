package timeseriesestimator

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	tse "github.com/deepaucksharma/Phoenix/internal/processor/timeseries_estimator"
)

// Test that the processor switches from exact to HLL when the limit is exceeded.
func TestTimeseriesEstimatorSwitch(t *testing.T) {
	factory := tse.NewFactory()
	cfg := factory.CreateDefaultConfig().(*tse.Config)
	cfg.MaxUniqueTimeSeries = 2
	cfg.EstimatorType = "exact"

	next := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(context.Background(), processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
	}, cfg, next)
	require.NoError(t, err)

	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = proc.Shutdown(context.Background()) })

	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()

	for i := 0; i < 3; i++ {
		m := sm.Metrics().AppendEmpty()
		m.SetName(fmt.Sprintf("metric%d", i))
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(float64(i))
	}

	err = proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	up := proc.(interfaces.UpdateableProcessor)
	status, err := up.GetConfigStatus(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "hll", status.Parameters["estimator_type"])
}
