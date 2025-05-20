package testutils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
)

// TestGenerateConfigPatchesTargetsConfiguredProcessor verifies that patches
// produced by PIDControlHelper target the configured processor name.
func TestGenerateConfigPatchesTargetsConfiguredProcessor(t *testing.T) {
	helper := NewPIDControlHelper()
	helper.AddController("c1", 1, 0, 0, 1)
	helper.SetKPIValue("metric", 2)

	mapping := map[string]ControllerMapping{
		"c1": {
			KPIMetricName:   "metric",
			TargetProcessor: "adaptive_topk",
			ParameterPath:   "k_value",
			ScaleFactor:     1,
			MinValue:        5,
			MaxValue:        50,
		},
	}

	patches := helper.GenerateConfigPatches(context.Background(), mapping)
	require.Len(t, patches, 1)
	expectedID := component.NewID(component.MustNewType("adaptive_topk"))
	assert.Equal(t, expectedID, patches[0].TargetProcessorName)
}
