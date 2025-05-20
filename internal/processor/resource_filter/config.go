package resource_filter

import "fmt"

// FilterStrategy determines how resources are selected.
type FilterStrategy string

const (
	StrategyPriority FilterStrategy = "priority"
	StrategyTopK     FilterStrategy = "topk"
	StrategyHybrid   FilterStrategy = "hybrid"
)

// PriorityLevel represents the priority assigned to a resource.
type PriorityLevel string

const (
	PriorityLow      PriorityLevel = "low"
	PriorityMedium   PriorityLevel = "medium"
	PriorityHigh     PriorityLevel = "high"
	PriorityCritical PriorityLevel = "critical"
)

// AggregationStrategy defines how rollup metrics are aggregated.
type AggregationStrategy string

const (
	AggregationSum AggregationStrategy = "sum"
	AggregationAvg AggregationStrategy = "avg"
)

// PriorityRule defines how to assign a priority to a resource.
type PriorityRule struct {
	Match    string        `mapstructure:"match"`
	Priority PriorityLevel `mapstructure:"priority"`
}

// TopKConfig configures the top-k algorithm.
type TopKConfig struct {
	KValue         int     `mapstructure:"k_value"`
	KMin           int     `mapstructure:"k_min"`
	KMax           int     `mapstructure:"k_max"`
	ResourceField  string  `mapstructure:"resource_field"`
	CounterField   string  `mapstructure:"counter_field"`
	CoverageTarget float64 `mapstructure:"coverage_target"`
}

// RollupConfig controls rollup aggregation of filtered resources.
type RollupConfig struct {
	Enabled           bool                `mapstructure:"enabled"`
	PriorityThreshold PriorityLevel       `mapstructure:"priority_threshold"`
	Strategy          AggregationStrategy `mapstructure:"strategy"`
	NamePrefix        string              `mapstructure:"name_prefix"`
}

// Config holds the configuration for the resource filter functionality.
type Config struct {
	Enabled           bool           `mapstructure:"enabled"`
	FilterStrategy    FilterStrategy `mapstructure:"filter_strategy"`
	PriorityAttribute string         `mapstructure:"priority_attribute"`
	PriorityRules     []PriorityRule `mapstructure:"priority_rules"`
	TopK              TopKConfig     `mapstructure:"topk"`
	Rollup            RollupConfig   `mapstructure:"rollup"`
}

// Validate checks the configuration for common mistakes.
func (c *Config) Validate() error {
	switch c.FilterStrategy {
	case StrategyPriority, StrategyTopK, StrategyHybrid:
		// valid
	default:
		if c.FilterStrategy != "" {
			return fmt.Errorf("invalid filter_strategy: %s", c.FilterStrategy)
		}
	}

	if (c.FilterStrategy == StrategyPriority || c.FilterStrategy == StrategyHybrid) && len(c.PriorityRules) == 0 {
		return fmt.Errorf("priority_rules required for strategy %s", c.FilterStrategy)
	}

	for i, r := range c.PriorityRules {
		if r.Match == "" {
			return fmt.Errorf("priority_rules[%d].match cannot be empty", i)
		}
		switch r.Priority {
		case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		default:
			return fmt.Errorf("priority_rules[%d].priority invalid: %s", i, r.Priority)
		}
	}

	if c.FilterStrategy == StrategyTopK || c.FilterStrategy == StrategyHybrid {
		if c.TopK.KValue <= 0 {
			return fmt.Errorf("topk.k_value must be >0")
		}
		if c.TopK.KMin < 0 || c.TopK.KMax < c.TopK.KMin {
			return fmt.Errorf("invalid topk k_min/k_max")
		}
		if c.TopK.ResourceField == "" || c.TopK.CounterField == "" {
			return fmt.Errorf("topk.resource_field and counter_field must be set")
		}
	}

	if c.Rollup.Enabled {
		switch c.Rollup.PriorityThreshold {
		case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		default:
			return fmt.Errorf("invalid rollup.priority_threshold: %s", c.Rollup.PriorityThreshold)
		}
		switch c.Rollup.Strategy {
		case AggregationSum, AggregationAvg:
		default:
			return fmt.Errorf("invalid rollup.strategy: %s", c.Rollup.Strategy)
		}
		if c.Rollup.NamePrefix == "" {
			return fmt.Errorf("rollup.name_prefix cannot be empty when enabled")
		}
	}

	return nil
}
