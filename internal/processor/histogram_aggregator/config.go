package histogram_aggregator

import (
	"fmt"

	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

// Config defines configuration for histogram aggregation processor.
type Config struct {
	*base.BaseConfig `mapstructure:",squash"`

	// MaxBuckets defines the maximum number of histogram buckets to preserve.
	// If a histogram has more buckets, they will be compacted.
	MaxBuckets int `mapstructure:"max_buckets"`

	// TargetProcessors contains a list of process names for which to apply specialized
	// histogram configuration. If empty, applies to all processes.
	TargetProcessors []string `mapstructure:"target_processors"`

	// CustomBoundaries allows defining specific bucket boundaries for
	// important metrics. When specified, histograms will be rebucketed
	// to these boundaries.
	CustomBoundaries map[string][]float64 `mapstructure:"custom_boundaries"`
}

// Validate ensures all required parameters have valid values.
func (cfg *Config) Validate() error {
	if err := cfg.BaseConfig.Validate(); err != nil {
		return err
	}

	if cfg.MaxBuckets <= 0 {
		return fmt.Errorf("max_buckets must be positive")
	}

	return nil
}