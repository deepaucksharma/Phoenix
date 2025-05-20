package resource_filter

// FilterStrategy defines the strategy used for filtering resources.
type FilterStrategy string

const (
	StrategyPriority FilterStrategy = "priority"
	StrategyTopK     FilterStrategy = "topk"
	StrategyHybrid   FilterStrategy = "hybrid"
)

// PriorityLevel indicates the importance of a resource.
type PriorityLevel string

const (
	PriorityCritical PriorityLevel = "critical"
	PriorityHigh     PriorityLevel = "high"
	PriorityMedium   PriorityLevel = "medium"
	PriorityLow      PriorityLevel = "low"
)

// AggregationType defines rollup aggregation behavior.
type AggregationType string

const (
	AggregationSum AggregationType = "sum"
	AggregationAvg AggregationType = "avg"
)

// PriorityRule maps a match expression to a priority level.
type PriorityRule struct {
	Match    string        `mapstructure:"match"`
	Priority PriorityLevel `mapstructure:"priority"`
}

// TopKConfig configures the top-k strategy.
type TopKConfig struct {
	KValue         int     `mapstructure:"k_value"`
	KMin           int     `mapstructure:"k_min"`
	KMax           int     `mapstructure:"k_max"`
	ResourceField  string  `mapstructure:"resource_field"`
	CounterField   string  `mapstructure:"counter_field"`
	CoverageTarget float64 `mapstructure:"coverage_target"`
}

// RollupConfig controls rollup aggregation of low priority metrics.
type RollupConfig struct {
	Enabled           bool            `mapstructure:"enabled"`
	PriorityThreshold PriorityLevel   `mapstructure:"priority_threshold"`
	Strategy          AggregationType `mapstructure:"strategy"`
	NamePrefix        string          `mapstructure:"name_prefix"`
}

// Config holds the full filter configuration.
type Config struct {
	Enabled           bool           `mapstructure:"enabled"`
	FilterStrategy    FilterStrategy `mapstructure:"filter_strategy"`
	PriorityAttribute string         `mapstructure:"priority_attribute"`
	PriorityRules     []PriorityRule `mapstructure:"priority_rules"`
	TopK              TopKConfig     `mapstructure:"topk"`
	Rollup            RollupConfig   `mapstructure:"rollup"`
}

// Validate validates the configuration. This stub always returns nil.
func (c *Config) Validate() error { return nil }
