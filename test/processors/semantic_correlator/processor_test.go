package semantic_correlator

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

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/semantic_correlator"
)

func TestSemanticCorrelatorConfigValidate(t *testing.T) {
	cfg := &semantic_correlator.Config{}
	assert.NoError(t, cfg.Validate())
}

func TestSemanticCorrelatorProcessor(t *testing.T) {
	factory := semantic_correlator.NewFactory()
	cfg := factory.CreateDefaultConfig().(*semantic_correlator.Config)
	sink := new(consumertest.MetricsSink)
	ctx := context.Background()
	settings := processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}, ID: component.NewIDWithName(component.MustNewType("semantic_correlator"), "")}
	proc, err := factory.CreateMetrics(ctx, settings, cfg, sink)
	require.NoError(t, err)

	// This processor does not implement UpdateableProcessor
	if _, ok := proc.(interfaces.UpdateableProcessor); ok {
		t.Fatalf("semantic_correlator should not implement UpdateableProcessor")
	}

	require.NoError(t, proc.Start(ctx, nil))
	metrics := generateTestMetrics()
	require.NoError(t, proc.ConsumeMetrics(ctx, metrics))
	processed := sink.AllMetrics()
	require.Len(t, processed, 1)
	assert.Equal(t, metrics.ResourceMetrics().Len(), processed[0].ResourceMetrics().Len())
	require.NoError(t, proc.Shutdown(ctx))
}

func generateTestMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("test")
	m.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(1)
	return md
}
