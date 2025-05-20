package semantic_correlator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/deepaucksharma/Phoenix/internal/processor/semantic_correlator"
	processors "github.com/deepaucksharma/Phoenix/test/processors/templates"
)

func TestValidate(t *testing.T) {
	cfg := &semantic_correlator.Config{Enabled: true, Method: "granger", Lag: 1, Bins: 5}
	assert.NoError(t, cfg.Validate())
}

func TestSemanticCorrelatorProcessor(t *testing.T) {
	factory := semantic_correlator.NewFactory()
	cfg := factory.CreateDefaultConfig().(*semantic_correlator.Config)

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
