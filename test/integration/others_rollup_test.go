package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"

	"github.com/deepaucksharma/Phoenix/internal/processor/others_rollup"
	"github.com/deepaucksharma/Phoenix/internal/processor/priority_tagger"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// TestOthersRollupIntegration verifies that low priority processes are rolled up
// into a single synthetic resource when used with priority_tagger.
func TestOthersRollupIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup priority_tagger
	ptFactory := priority_tagger.NewFactory()
	ptCfg := ptFactory.CreateDefaultConfig().(*priority_tagger.Config)
	ptCfg.Rules = []priority_tagger.Rule{
		{Match: "nginx.*", Priority: "high"},
		{Match: ".*mysql.*", Priority: "critical"},
		{Match: "background.*", Priority: "low"},
		{Match: "other.*", Priority: "low"},
	}
	ptCfg.Enabled = true

	// Setup others_rollup
	orFactory := others_rollup.NewFactory()
	orCfg := orFactory.CreateDefaultConfig().(*others_rollup.Config)
	orCfg.Enabled = true
	orCfg.Strategy = "sum"

	sink := new(consumertest.MetricsSink)

	// Build pipeline: priority_tagger -> others_rollup -> sink
	orProc, err := orFactory.CreateMetricsProcessor(ctx, processor.CreateSettings{}, orCfg, sink)
	require.NoError(t, err)
	ptProc, err := ptFactory.CreateMetricsProcessor(ctx, processor.CreateSettings{}, ptCfg, orProc)
	require.NoError(t, err)

	require.NoError(t, orProc.Start(ctx, nil))
	require.NoError(t, ptProc.Start(ctx, nil))

	metrics := testutils.GenerateMetrics()
	require.NoError(t, ptProc.ConsumeMetrics(ctx, metrics))

	out := sink.AllMetrics()
	require.Len(t, out, 1)

	rms := out[0].ResourceMetrics()
	require.Equal(t, 3, rms.Len())

	var othersFound bool
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		attrs := rm.Resource().Attributes()
		if v, ok := attrs.Get("process.name"); ok && v.Str() == "others" {
			othersFound = true
			prio, _ := attrs.Get("aemf.process.priority")
			require.Equal(t, "low", prio.Str())
		}
	}
	require.True(t, othersFound, "aggregated 'others' resource not found")

	require.NoError(t, ptProc.Shutdown(ctx))
	require.NoError(t, orProc.Shutdown(ctx))
}
