package adaptive_topk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

func TestConfigValidate(t *testing.T) {
	cfg := &adaptive_topk.Config{
		BaseConfig:    base.WithEnabled(true),
		KValue:        5,
		KMin:          1,
		KMax:          10,
		ResourceField: "process.name",
		CounterField:  "process.cpu_seconds_total",
	}
	require.NoError(t, cfg.Validate())

	cfg.KValue = 0
	assert.Error(t, cfg.Validate())
	cfg.KValue = 5

	cfg.KMin = 0
	assert.Error(t, cfg.Validate())
	cfg.KMin = 1

	cfg.KMax = 0
	assert.Error(t, cfg.Validate())
	cfg.KMax = 10

	cfg.KMin = 5
	cfg.KMax = 3
	assert.Error(t, cfg.Validate())
	cfg.KMin = 1
	cfg.KMax = 10

	cfg.KValue = 11
	assert.Error(t, cfg.Validate())
	cfg.KValue = 5

	cfg.ResourceField = ""
	assert.Error(t, cfg.Validate())
	cfg.ResourceField = "process.name"

	cfg.CounterField = ""
	assert.Error(t, cfg.Validate())
}

func TestConfigEnabled(t *testing.T) {
	cfg := &adaptive_topk.Config{BaseConfig: base.WithEnabled(false)}
	assert.False(t, cfg.IsEnabled())
	cfg.SetEnabled(true)
	assert.True(t, cfg.IsEnabled())
}
