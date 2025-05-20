package pic_control_ext

import (
	"fmt"
	"path/filepath"
)

// Config defines configuration for the pic_control_ext extension.
type Config struct {
	// PolicyFilePath is the path to the policy configuration file
	PolicyFilePath string `mapstructure:"policy_file_path"`

	// WatchPolicyFile indicates whether to watch for changes to the policy file
	WatchPolicyFile bool `mapstructure:"watch_policy_file"`

	// PolicyUpdateIntervalSeconds is the interval in seconds at which to check for policy updates
	PolicyUpdateIntervalSeconds int `mapstructure:"policy_update_interval_seconds"`

	// MaxChangesPerWindow is the maximum number of configuration changes allowed per window
	MaxChangesPerWindow int `mapstructure:"max_changes_per_window"`

	// RateLimitWindowSeconds is the window size in seconds for rate limiting
	RateLimitWindowSeconds int `mapstructure:"rate_limit_window_seconds"`

	// HistorySize is the number of patch records to keep in history
	HistorySize int `mapstructure:"history_size"`

	// SafeModeCooldownSeconds is the cooldown period in seconds after entering safe mode
	SafeModeCooldownSeconds int `mapstructure:"safe_mode_cooldown_seconds"`
}

// Validate validates the extension configuration.
func (c *Config) Validate() error {
	if c.PolicyFilePath != "" {
		if !filepath.IsAbs(c.PolicyFilePath) {
			return fmt.Errorf("policy_file_path must be an absolute path")
		}
	}

	if c.PolicyUpdateIntervalSeconds < 0 {
		return fmt.Errorf("policy_update_interval_seconds must be non-negative")
	}

	if c.MaxChangesPerWindow <= 0 {
		return fmt.Errorf("max_changes_per_window must be positive")
	}

	if c.RateLimitWindowSeconds <= 0 {
		return fmt.Errorf("rate_limit_window_seconds must be positive")
	}

	if c.HistorySize <= 0 {
		return fmt.Errorf("history_size must be positive")
	}

	if c.SafeModeCooldownSeconds < 0 {
		return fmt.Errorf("safe_mode_cooldown_seconds must be non-negative")
	}

	return nil
}

// CreateDefaultConfig creates the default configuration for the extension.
func createDefaultConfig() *Config {
	return &Config{
		PolicyFilePath:              "/etc/sa-omf/policy.yaml",
		WatchPolicyFile:             true,
		PolicyUpdateIntervalSeconds: 60,
		MaxChangesPerWindow:         10,
		RateLimitWindowSeconds:      60,
		HistorySize:                 100,
		SafeModeCooldownSeconds:     300,
	}
}
