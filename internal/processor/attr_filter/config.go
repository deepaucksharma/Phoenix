package attr_filter

import (
	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

// Config defines configuration for the attr_filter processor.
type Config struct {
	base.BaseConfig `mapstructure:",squash"`
	// Attributes lists the attribute keys to remove from metrics.
	Attributes []string `mapstructure:"attributes"`
}
