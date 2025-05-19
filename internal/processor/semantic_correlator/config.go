package semantic_correlator

// Config defines configuration for the semantic_correlator processor.
type Config struct {
	Enabled bool   `mapstructure:"enabled"`
	Method  string `mapstructure:"method"`
	Lag     int    `mapstructure:"lag"`
	Bins    int    `mapstructure:"bins"`
}

func (c *Config) Validate() error { return nil }

func (c *Config) IsEnabled() bool   { return c.Enabled }
func (c *Config) SetEnabled(v bool) { c.Enabled = v }
