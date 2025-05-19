package multi_temporal_adaptive_engine

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/deepaucksharma/Phoenix/internal/processor/multi_temporal_adaptive_engine"
	"github.com/deepaucksharma/Phoenix/pkg/util/timeseries"
	processors_test "github.com/deepaucksharma/Phoenix/test/processors/templates"
)

func TestMultiTemporalAdaptiveEngineProcessor(t *testing.T) {
	factory := multi_temporal_adaptive_engine.NewFactory()
	cfg := factory.CreateDefaultConfig().(*multi_temporal_adaptive_engine.Config)
	cfg.Threshold = 2

	testCases := []processors_test.ProcessorTestCase{
		{
			Name:         "DefaultProcessing",
			InputMetrics: processors_test.GenerateTestMetrics([]string{"proc"}),
			ExpectedOutput: func(md pmetric.Metrics) bool {
				return md.ResourceMetrics().Len() == 1
			},
		},
	}

	processors_test.RunProcessorTests(t, factory, cfg, testCases)

	data := []float64{1, 2, 3, 4, 10, 5, 6}
	idx := timeseries.DetectZScore(data, cfg.Threshold)
	require.Equal(t, []int{4}, idx)
}
