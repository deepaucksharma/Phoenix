package cpu_histogram_converter

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

	"github.com/deepaucksharma/Phoenix/internal/processor/base"
	"github.com/deepaucksharma/Phoenix/internal/processor/cpu_histogram_converter"
)

func generateCPUMetrics(values []float64) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("process.cpu.utilization")
	gauge := m.SetEmptyGauge()
	for _, v := range values {
		dp := gauge.DataPoints().AppendEmpty()
		dp.SetDoubleValue(v)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}
	return md
}

func TestCPUHistogramFactory(t *testing.T) {
	factory := cpu_histogram_converter.NewFactory()
	assert.NotNil(t, factory)
	cfg := factory.CreateDefaultConfig()
	assert.IsType(t, &cpu_histogram_converter.Config{}, cfg)
}

func TestCPUHistogramConverterPatterns(t *testing.T) {
	factory := cpu_histogram_converter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*cpu_histogram_converter.Config)

	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(context.Background(), processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
	}, cfg, sink)
	require.NoError(t, err)

	require.NoError(t, proc.Start(context.Background(), nil))

	patterns := [][]float64{
		{10, 10, 10},     // steady
		{0, 100, 0, 100}, // spikes
		{0, 0, 0},        // zero usage
	}
	for _, ptn := range patterns {
		md := generateCPUMetrics(ptn)
		err = proc.ConsumeMetrics(context.Background(), md)
		require.NoError(t, err)
	}

	out := sink.AllMetrics()
	require.NotEmpty(t, out)
	processed := out[len(out)-1]
	foundHistogram := false
	for i := 0; i < processed.ResourceMetrics().Len(); i++ {
		rm := processed.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				if m.Type() == pmetric.MetricTypeHistogram && m.Name() == "process.cpu.utilization" {
					foundHistogram = true
				}
			}
		}
	}
	assert.True(t, foundHistogram, "histogram metric not found")

	assert.NoError(t, proc.Shutdown(context.Background()))
}

func TestCPUHistogramConverterEmptyInput(t *testing.T) {
	factory := cpu_histogram_converter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*cpu_histogram_converter.Config)

	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(context.Background(), processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
	}, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, proc.Start(context.Background(), nil))

	md := pmetric.NewMetrics()
	err = proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)
	assert.NotEmpty(t, sink.AllMetrics())
	assert.NoError(t, proc.Shutdown(context.Background()))
}

func TestCPUHistogramConverterInvalidConfig(t *testing.T) {
	cfg := &cpu_histogram_converter.Config{
		BaseConfig: base.NewBaseConfig(),
		Boundaries: []float64{50, 10},
	}
	err := cfg.Validate()
	assert.Error(t, err)
}
