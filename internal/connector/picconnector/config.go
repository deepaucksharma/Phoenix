// Package picconnector implements an exporter that forwards configuration 
// patches from pid_decider to pic_control.
package picconnector

// Config defines the configuration for the pic_connector exporter.
// Currently, the connector doesn't require any specific configuration,
// but this structure is included for future extension.
type Config struct {
}

// Validate checks if the exporter configuration is valid.
func (cfg *Config) Validate() error {
	return nil
}