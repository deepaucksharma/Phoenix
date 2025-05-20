package resource_filter

import (
	"testing"

	"github.com/stretchr/testify/assert"

	internal "github.com/deepaucksharma/Phoenix/internal/processor/resource_filter"
)

// Test that a zero-value config validates without error.
func TestDefaultConfigValidation(t *testing.T) {
	var cfg internal.Config
	assert.NoError(t, cfg.Validate())
}

func TestConfigValidationErrors(t *testing.T) {
	t.Run("invalid strategy", func(t *testing.T) {
		cfg := internal.Config{FilterStrategy: "bogus"}
		assert.Error(t, cfg.Validate())
	})

	t.Run("missing priority rules", func(t *testing.T) {
		cfg := internal.Config{FilterStrategy: internal.StrategyPriority}
		assert.Error(t, cfg.Validate())
	})

	t.Run("invalid topk", func(t *testing.T) {
		cfg := internal.Config{FilterStrategy: internal.StrategyTopK}
		assert.Error(t, cfg.Validate())
	})

	t.Run("invalid rollup", func(t *testing.T) {
		cfg := internal.Config{
			FilterStrategy: internal.StrategyPriority,
			PriorityRules:  []internal.PriorityRule{{Match: ".*", Priority: internal.PriorityLow}},
			Rollup: internal.RollupConfig{
				Enabled:    true,
				NamePrefix: "", // missing
			},
		}
		assert.Error(t, cfg.Validate())
	})
}
