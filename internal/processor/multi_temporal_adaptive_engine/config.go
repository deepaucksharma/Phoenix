package multi_temporal_adaptive_engine

// Config defines configuration for the multi_temporal_adaptive_engine processor.
type Config struct {
	Enabled   bool    `mapstructure:"enabled"`
	Threshold float64 `mapstructure:"threshold"`
}

func (c *Config) Validate() error   { return nil }
func (c *Config) IsEnabled() bool   { return c.Enabled }
func (c *Config) SetEnabled(v bool) { c.Enabled = v }
