package cpu_histogram_converter

import (
	"fmt"

	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

// Config defines configuration for the cpu histogram converter.
type Config struct {
	*base.BaseConfig `mapstructure:",squash"`
	Boundaries       []float64 `mapstructure:"boundaries"`
	MetricNames      []string  `mapstructure:"metric_names"`
}

// Validate checks the configuration for any issues.
func (c *Config) Validate() error {
	if err := c.BaseConfig.Validate(); err != nil {
		return err
	}
	if len(c.Boundaries) == 0 {
		return fmt.Errorf("boundaries cannot be empty")
	}
	for i := 1; i < len(c.Boundaries); i++ {
		if c.Boundaries[i] <= c.Boundaries[i-1] {
			return fmt.Errorf("boundaries must be strictly increasing")
		}
	}
	if len(c.MetricNames) == 0 {
		c.MetricNames = []string{"process.cpu.utilization"}
	}
	return nil
}
