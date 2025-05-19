package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"

	"github.com/deepaucksharma/Phoenix/internal/processor/multi_temporal_adaptive_engine"
)

func TestEngineLifecycle(t *testing.T) {
	factory := multi_temporal_adaptive_engine.NewFactory()
	cfg := factory.CreateDefaultConfig().(*multi_temporal_adaptive_engine.Config)

	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetricsProcessor(context.Background(), processor.CreateSettings{}, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, proc)

	host := &componenttest.TestHost{}
	require.NoError(t, proc.Start(context.Background(), host))
	require.NoError(t, proc.Shutdown(context.Background()))
}
