// Package timeseries_estimator provides a processor that estimates the number of 
// unique time series being processed.
package timeseries_estimator

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// Config defines the configuration for the timeseries_estimator processor.
type Config struct {
	// ProcessorSettings has the common settings for a processor.
	processorhelper.ProcessorSettings `mapstructure:",squash"`

	// Enabled determines whether this processor is enabled.
	Enabled bool `mapstructure:"enabled"`

	// OutputMetricName is the name of the metric that will be output with the estimate.
	OutputMetricName string `mapstructure:"output_metric_name"`

	// EstimatorType determines the algorithm used for estimation.
	// Valid values: "exact", "hll" (HyperLogLog).
	EstimatorType string `mapstructure:"estimator_type"`

	// HLLPrecision sets the precision for the HyperLogLog algorithm (4-16).
	// Higher precision = more accuracy but more memory usage.
	// Default: 10 (creates 1024 registers).
	HLLPrecision int `mapstructure:"hll_precision"`

	// MemoryLimitMB sets a memory usage limit in MB for the exact counting mode.
	// If memory usage exceeds this limit, the processor will use downsampling
	// or switch to HLL. Default: 100
	MemoryLimitMB int `mapstructure:"memory_limit_mb"`

	// RefreshInterval determines how often the processor recalculates
	// the estimate from scratch. Default: 1 hour
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
}

// Validate checks if the processor configuration is valid.
func (cfg *Config) Validate() error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.OutputMetricName == "" {
		return fmt.Errorf("output_metric_name must be specified")
	}

	if cfg.EstimatorType != "exact" && cfg.EstimatorType != "hll" {
		return fmt.Errorf("estimator_type must be either 'exact' or 'hll'")
	}

	if cfg.EstimatorType == "hll" {
		if cfg.HLLPrecision < 4 || cfg.HLLPrecision > 16 {
			return fmt.Errorf("hll_precision must be between 4 and 16")
		}
	}

	if cfg.MemoryLimitMB <= 0 {
		return fmt.Errorf("memory_limit_mb must be greater than 0")
	}

	if cfg.RefreshInterval <= 0 {
		return fmt.Errorf("refresh_interval must be greater than 0")
	}

	return nil
}

// IsEnabled returns whether this processor is enabled.
func (cfg *Config) IsEnabled() bool {
	return cfg.Enabled
}

// CreateDefaultConfig creates the default configuration for the processor.
func createDefaultConfig() component.Config {
	return &Config{
		ProcessorSettings: processorhelper.NewProcessorSettings(component.NewID(typeStr)),
		Enabled:           true,
		OutputMetricName:  "aemf_estimated_active_timeseries",
		EstimatorType:     "hll",
		HLLPrecision:      10,
		MemoryLimitMB:     100,
		RefreshInterval:   time.Hour,
	}
}