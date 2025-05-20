// Package metric_pipeline implements a combined processor that handles resource filtering
// and metric transformation in a single processing step for efficiency.
package metric_pipeline

import (
	"github.com/deepaucksharma/Phoenix/internal/processor/resource_filter"
)

// ResourceFilterConfig holds the configuration for resource filtering
type ResourceFilterConfig struct {
	Enabled           bool                             `mapstructure:"enabled"`
	FilterStrategy    resource_filter.FilterStrategy   `mapstructure:"filter_strategy"`
	PriorityAttribute string                           `mapstructure:"priority_attribute"`
	PriorityRules     []resource_filter.PriorityRule   `mapstructure:"priority_rules"`
	TopK              resource_filter.TopKConfig       `mapstructure:"topk"`
	Rollup            resource_filter.RollupConfig     `mapstructure:"rollup"`
}

// HistogramConfig holds configuration for histogram generation
type HistogramConfig struct {
	Enabled    bool                       `mapstructure:"enabled"`
	MaxBuckets int                        `mapstructure:"max_buckets"`
	Metrics    map[string]HistogramMetric `mapstructure:"metrics"`
}

// HistogramMetric defines how to generate a histogram for a specific metric
type HistogramMetric struct {
	Boundaries []float64 `mapstructure:"boundaries"`
}

// AttributeAction defines an action to perform on an attribute
type AttributeAction struct {
	Key    string      `mapstructure:"key"`
	Action string      `mapstructure:"action"` // insert, update, delete, upsert
	Value  interface{} `mapstructure:"value"`  // Value for insert or update
}

// AttributeConfig holds configuration for attribute processing
type AttributeConfig struct {
	Actions []AttributeAction `mapstructure:"actions"`
}

// TransformationConfig holds the configuration for metric transformation
type TransformationConfig struct {
	Histograms HistogramConfig `mapstructure:"histograms"`
	Attributes AttributeConfig  `mapstructure:"attributes"`
}

// Config defines the configuration for the metric_pipeline processor
type Config struct {
	ResourceFilter  ResourceFilterConfig  `mapstructure:"resource_filter"`
	Transformation TransformationConfig `mapstructure:"transformation"`
}

// Validate validates the processor configuration
func (cfg *Config) Validate() error {
	// Configure a resource filter config for validation
	rfConfig := &resource_filter.Config{
		Enabled:           cfg.ResourceFilter.Enabled,
		FilterStrategy:    cfg.ResourceFilter.FilterStrategy,
		PriorityAttribute: cfg.ResourceFilter.PriorityAttribute,
		PriorityRules:     cfg.ResourceFilter.PriorityRules,
		TopK:              cfg.ResourceFilter.TopK,
		Rollup:            cfg.ResourceFilter.Rollup,
	}
	
	// Use the resource filter's validation
	return rfConfig.Validate()
}

// IsEnabled returns whether the processor is enabled
func (cfg *Config) IsEnabled() bool {
	return true // Always enabled by default - individual components can be disabled
}

// SetEnabled sets the enabled state
func (cfg *Config) SetEnabled(enabled bool) {
	// This processor is always enabled, but individual components can be disabled
}