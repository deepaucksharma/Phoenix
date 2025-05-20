// Package adaptive_topk implements a processor that dynamically selects top-k resources
// based on self-tuning mechanisms.
package adaptive_topk

import (
	"fmt"

	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

// Config defines the configuration for the adaptive_topk processor.
type Config struct {
	*base.BaseConfig `mapstructure:",squash"`
	KValue           int    `mapstructure:"k_value"`        // Number of top items to track
	KMin             int    `mapstructure:"k_min"`          // Minimum value for k
	KMax             int    `mapstructure:"k_max"`          // Maximum value for k
	ResourceField    string `mapstructure:"resource_field"` // Field to use for identifying resources
	CounterField     string `mapstructure:"counter_field"`  // Field to use for ranking resources
}

// Validate checks if the processor configuration is valid.
func (cfg *Config) Validate() error {
	// Validate base config
	if err := cfg.BaseConfig.Validate(); err != nil {
		return err
	}

	if cfg.KValue <= 0 {
		return fmt.Errorf("k_value must be positive")
	}

	if cfg.KMin <= 0 {
		return fmt.Errorf("k_min must be positive")
	}

	if cfg.KMax <= 0 {
		return fmt.Errorf("k_max must be positive")
	}

	if cfg.KMin > cfg.KMax {
		return fmt.Errorf("k_min must be less than or equal to k_max")
	}

	if cfg.KValue < cfg.KMin || cfg.KValue > cfg.KMax {
		return fmt.Errorf("k_value must be within range [k_min, k_max]")
	}

	if cfg.ResourceField == "" {
		return fmt.Errorf("resource_field must be specified")
	}

	if cfg.CounterField == "" {
		return fmt.Errorf("counter_field must be specified")
	}

	return nil
}

// IsEnabled returns whether the processor is enabled.
func (cfg *Config) IsEnabled() bool {
	return cfg.BaseConfig.IsEnabled()
}

// SetEnabled sets the enabled status of the processor.
func (cfg *Config) SetEnabled(enabled bool) {
	cfg.BaseConfig.SetEnabled(enabled)
}
