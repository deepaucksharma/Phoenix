// Package safety provides monitoring capabilities for system resources
package safety

// Config defines configuration for the safety monitor
type Config struct {
	CPUUsageThresholdMCores int     `mapstructure:"cpu_usage_threshold_mcores"`
	MemoryThresholdMiB      int     `mapstructure:"memory_threshold_mib"`
	SafeModeCooldownSeconds int     `mapstructure:"safe_mode_cooldown_seconds"`
	OverrideExpirySeconds   int     `mapstructure:"override_expiry_seconds"`
	OverrideMultiplier      float64 `mapstructure:"override_multiplier"`
	MetricsCheckIntervalMs  int     `mapstructure:"metrics_check_interval_ms"`
}

// DefaultConfig returns the default configuration for the safety monitor
func DefaultConfig() *Config {
	return &Config{
		CPUUsageThresholdMCores: 500,  // 0.5 cores
		MemoryThresholdMiB:      200,  // 200 MiB
		SafeModeCooldownSeconds: 30,   // 30 seconds
		OverrideExpirySeconds:   300,  // 5 minutes
		OverrideMultiplier:      1.5,  // Increase thresholds by 50%
		MetricsCheckIntervalMs:  1000, // Check every second
	}
}
