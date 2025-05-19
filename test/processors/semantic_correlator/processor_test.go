package semantic_correlator

import (
	"testing"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/deepaucksharma/Phoenix/internal/processor/semantic_correlator"
	processors_test "github.com/deepaucksharma/Phoenix/test/processors/templates"
)

func TestSemanticCorrelatorProcessor(t *testing.T) {
	factory := semantic_correlator.NewFactory()
	cfg := factory.CreateDefaultConfig().(*semantic_correlator.Config)

	testCases := []processors_test.ProcessorTestCase{
		{
			Name:         "GrangerMethod",
			InputMetrics: processors_test.GenerateTestMetrics([]string{"proc"}),
			ExpectedOutput: func(md pmetric.Metrics) bool {
				return md.ResourceMetrics().Len() == 1
			},
			ConfigPatches: []interfaces.ConfigPatch{
				{
					PatchID:             "set-method",
					TargetProcessorName: component.MustNewID("semantic_correlator"),
					ParameterPath:       "method",
					NewValue:            "granger",
				},
			},
		},
		{
			Name:         "TransferEntropyMethod",
			InputMetrics: processors_test.GenerateTestMetrics([]string{"proc"}),
			ExpectedOutput: func(md pmetric.Metrics) bool {
				return md.ResourceMetrics().Len() == 1
			},
			ConfigPatches: []interfaces.ConfigPatch{
				{
					PatchID:             "set-method-te",
					TargetProcessorName: component.MustNewID("semantic_correlator"),
					ParameterPath:       "method",
					NewValue:            "transfer",
				},
			},
		},
	}

	processors_test.RunProcessorTests(t, factory, cfg, testCases)
}
