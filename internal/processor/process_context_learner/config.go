package process_context_learner

import "fmt"

// Config defines the configuration for the process_context_learner processor.
type Config struct {
	Enabled       bool    `mapstructure:"enabled"`
	DampingFactor float64 `mapstructure:"damping_factor"`
	Iterations    int     `mapstructure:"iterations"`
}

// Validate checks if the configuration is valid.
func (cfg *Config) Validate() error {
	if cfg.DampingFactor <= 0 || cfg.DampingFactor >= 1 {
		return fmt.Errorf("damping_factor must be between 0 and 1")
	}
	if cfg.Iterations <= 0 {
		return fmt.Errorf("iterations must be positive")
	}
	return nil
}
