package multi_temporal_adaptive_engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/deepaucksharma/Phoenix/internal/processor/multi_temporal_adaptive_engine"
	processors "github.com/deepaucksharma/Phoenix/test/processors/templates"
)

func TestValidate(t *testing.T) {
	cfg := &multi_temporal_adaptive_engine.Config{Enabled: true, Threshold: 2}
	assert.NoError(t, cfg.Validate())
}

func TestMultiTemporalAdaptiveEngineProcessor(t *testing.T) {
	factory := multi_temporal_adaptive_engine.NewFactory()
	cfg := factory.CreateDefaultConfig().(*multi_temporal_adaptive_engine.Config)

	testCases := []processors.ProcessorTestCase{
		{
			Name:         "Basic",
			InputMetrics: processors.GenerateTestMetrics([]string{"p1"}),
			ExpectedOutput: func(md pmetric.Metrics) bool {
				return md.ResourceMetrics().Len() == 1
			},
		},
	}

	processors.RunProcessorTests(t, factory, cfg, testCases)
}
