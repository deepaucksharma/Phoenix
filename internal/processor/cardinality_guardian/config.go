package cardinality_guardian

import "fmt"

// Config defines configuration for the cardinality_guardian processor.
type Config struct {
	MaxUnique int  `mapstructure:"max_unique"`
	Enabled   bool `mapstructure:"enabled"`
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.MaxUnique <= 0 {
		return fmt.Errorf("max_unique must be positive")
	}
	return nil
}
