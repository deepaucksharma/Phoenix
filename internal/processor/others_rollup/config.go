package others_rollup

import "fmt"

// Package others_rollup aggregates metrics from low priority processes into a single
// synthetic resource.

// Config defines configuration for the others_rollup processor.
type Config struct {
	Strategy string `mapstructure:"strategy"` // Aggregation strategy: sum or avg
	Enabled  bool   `mapstructure:"enabled"`
}

// Validate checks if the configuration is valid.
func (cfg *Config) Validate() error {
	if cfg.Strategy == "" {
		cfg.Strategy = "sum"
	}
	switch cfg.Strategy {
	case "sum", "avg":
		return nil
	default:
		return fmt.Errorf("invalid strategy %s", cfg.Strategy)
	}
}
