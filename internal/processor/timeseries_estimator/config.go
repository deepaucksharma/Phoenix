package timeseries_estimator

import (
	"fmt"

	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

// Config defines configuration for the timeseries estimator processor.
type Config struct {
	*base.BaseConfig `mapstructure:",squash"`

	// EstimatorType selects the estimation algorithm. "exact" tracks every
	// unique series until MaxUniqueTimeSeries is reached before switching
	// to a probabilistic estimator.
	EstimatorType string `mapstructure:"estimator_type"`

	// MaxUniqueTimeSeries sets the maximum number of unique time series to
	// track when EstimatorType is "exact" before falling back to a
	// probabilistic approach.
	MaxUniqueTimeSeries int `mapstructure:"max_unique_time_series"`
}

// Validate checks if the configuration is valid.
func (cfg *Config) Validate() error {
	if err := cfg.BaseConfig.Validate(); err != nil {
		return err
	}

	if cfg.MaxUniqueTimeSeries <= 0 {
		return fmt.Errorf("max_unique_time_series must be positive")
	}

	if cfg.EstimatorType == "" {
		cfg.EstimatorType = "exact"
	}

	switch cfg.EstimatorType {
	case "exact", "hll":
		// valid types
	default:
		return fmt.Errorf("invalid estimator_type: %s", cfg.EstimatorType)
	}

	return nil
}
