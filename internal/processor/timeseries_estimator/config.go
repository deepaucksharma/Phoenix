package timeseries_estimator

import "github.com/deepaucksharma/Phoenix/internal/processor/base"

// Config defines configuration for the timeseries estimator processor.
type Config struct {
	*base.BaseConfig `mapstructure:",squash"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	return c.BaseConfig.Validate()
}
