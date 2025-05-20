package pic_control_ext

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

const (
	// The value of extension "type" in configuration.
	typeStr = "pic_control_ext"
)

// NewFactory creates a factory for the pic_control_ext extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		typeStr,
		createDefaultConfig,
		createExtension,
		component.StabilityLevelDevelopment,
	)
}

// createExtension creates the extension based on the config.
func createExtension(
	ctx context.Context,
	set extension.CreateSettings,
	cfg component.Config,
) (extension.Extension, error) {
	config, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid configuration")
	}

	return newPicControlExtension(config, component.TelemetrySettings{
		Logger: set.Logger,
	})
}
