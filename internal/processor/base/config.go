// Package base provides the base implementation for SA-OMF processors.
package base

// BaseConfig provides a common configuration structure for processors.
type BaseConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// IsEnabled returns whether the processor is enabled.
func (c *BaseConfig) IsEnabled() bool {
	return c.Enabled
}

// SetEnabled sets the enabled status of the processor.
func (c *BaseConfig) SetEnabled(enabled bool) {
	c.Enabled = enabled
}

// Validate validates the configuration.
func (c *BaseConfig) Validate() error {
	return nil
}

// WithEnabled returns a new BaseConfig with the enabled status set.
func WithEnabled(enabled bool) *BaseConfig {
	return &BaseConfig{
		Enabled: enabled,
	}
}