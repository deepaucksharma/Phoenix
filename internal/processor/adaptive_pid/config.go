// Package adaptive_pid implements the pid_decider processor which generates configuration
// patches using PID control loops to maintain KPI targets.
package adaptive_pid

import (
	"fmt"
	
	"go.opentelemetry.io/collector/component"
)

// Config defines the configuration for the pid_decider processor.
type Config struct {
	Controllers []ControllerConfig `mapstructure:"controllers"`
}

// ControllerConfig defines the configuration for a single PID controller.
type ControllerConfig struct {
	Name                string              `mapstructure:"name"`
	Enabled             bool                `mapstructure:"enabled"`
	KPIMetricName       string              `mapstructure:"kpi_metric_name"`
	KPITargetValue      float64             `mapstructure:"kpi_target_value"`
	KP                  float64             `mapstructure:"kp"`
	KI                  float64             `mapstructure:"ki"`
	KD                  float64             `mapstructure:"kd"`
	IntegralWindupLimit float64             `mapstructure:"integral_windup_limit"`
	HysteresisPercent   float64             `mapstructure:"hysteresis_percent"`
	OutputConfigPatches []OutputConfigPatch `mapstructure:"output_config_patches"`
	UseBayesian         bool                `mapstructure:"use_bayesian"`
	StallThreshold      int                 `mapstructure:"stall_threshold"`
}

// OutputConfigPatch defines how a PID controller affects a processor parameter.
type OutputConfigPatch struct {
	TargetProcessorName string  `mapstructure:"target_processor_name"`
	ParameterPath       string  `mapstructure:"parameter_path"`
	ChangeScaleFactor   float64 `mapstructure:"change_scale_factor"`
	MinValue            float64 `mapstructure:"min_value"`
	MaxValue            float64 `mapstructure:"max_value"`
}

// Validate checks if the processor configuration is valid.
func (cfg *Config) Validate() error {
	if len(cfg.Controllers) == 0 {
		return fmt.Errorf("at least one controller must be configured")
	}

	for i, controller := range cfg.Controllers {
		if controller.KPIMetricName == "" {
			return fmt.Errorf("controllers[%d]: kpi_metric_name cannot be empty", i)
		}

		if controller.Enabled && len(controller.OutputConfigPatches) == 0 {
			return fmt.Errorf("controllers[%d]: enabled controller must have at least one output_config_patch", i)
		}

		for j, patch := range controller.OutputConfigPatches {
				// Check if target processor name is empty
				if patch.TargetProcessorName == "" {
							return fmt.Errorf("controllers[%d].output_config_patches[%d]: target_processor_name cannot be empty", i, j)
			}
			if patch.ParameterPath == "" {
				return fmt.Errorf("controllers[%d].output_config_patches[%d]: parameter_path cannot be empty", i, j)
			}
			if patch.MinValue >= patch.MaxValue {
				return fmt.Errorf(
					"controllers[%d].output_config_patches[%d]: min_value (%f) must be less than max_value (%f)",
					i, j, patch.MinValue, patch.MaxValue,
				)
			}
		}

		if controller.UseBayesian && controller.StallThreshold <= 0 {
			return fmt.Errorf("controllers[%d]: stall_threshold must be >0 when use_bayesian enabled", i)
		}
	}

	return nil
}
