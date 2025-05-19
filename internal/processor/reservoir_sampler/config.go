package reservoir_sampler

import "fmt"

// Config defines the configuration for the reservoir_sampler processor.
type Config struct {
    ReservoirSize int  `mapstructure:"reservoir_size"`
    Enabled       bool `mapstructure:"enabled"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
    if c.ReservoirSize <= 0 {
        return fmt.Errorf("reservoir_size must be > 0")
    }
    return nil
}
