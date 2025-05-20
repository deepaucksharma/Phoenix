// Package adaptive_pid implements a processor that uses PID control for adaptive configuration.
package adaptive_pid

import (
	"context"
	"fmt"
	"math"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/control/pid"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/base"
	"github.com/deepaucksharma/Phoenix/pkg/util/bayesian"
	"github.com/deepaucksharma/Phoenix/pkg/util/typeconv"
)

var _ component.Config = (*Config)(nil)

// pidProcessor implements the pid_decider processor
type pidProcessor struct {
	*base.BaseProcessor
	config      *Config
	controllers []*controller
}

// controller represents a single PID control loop
type controller struct {
	config      ControllerConfig
	pid         *pid.Controller
	lastOutputs map[string]float64 // Last output for each parameter path
	lastValues  map[string]float64 // Last observed values for each KPI
	optimizer   *bayesian.Optimizer
	stallCount  int
	lastPIDOut  float64
}

// Ensure our processor implements the required interfaces
var _ processor.Metrics = (*pidProcessor)(nil)
var _ interfaces.UpdateableProcessor = (*pidProcessor)(nil)

// newProcessor creates a new pid_decider processor
func newProcessor(config *Config, settings component.TelemetrySettings, id component.ID) (*pidProcessor, error) {
	p := &pidProcessor{
		BaseProcessor: base.NewBaseProcessor(settings.Logger, nil, typeStr, id),
		config:        config,
		controllers:   make([]*controller, 0, len(config.Controllers)),
	}

	// Initialize controllers
	for _, cfg := range config.Controllers {
		if !cfg.Enabled {
			continue
		}

		// Create PID controller
		pidController := pid.NewController(cfg.KP, cfg.KI, cfg.KD, cfg.KPITargetValue)

		// Set integral windup limit if specified
		if cfg.IntegralWindupLimit > 0 {
			pidController.SetIntegralLimit(cfg.IntegralWindupLimit)
		}

		controller := &controller{
			config:      cfg,
			pid:         pidController,
			lastOutputs: make(map[string]float64),
			lastValues:  make(map[string]float64),
			stallCount:  0,
		}

		if cfg.UseBayesian || len(cfg.OutputConfigPatches) > 1 {
			bounds := make([][2]float64, len(cfg.OutputConfigPatches))
			for i, pch := range cfg.OutputConfigPatches {
				bounds[i] = [2]float64{pch.MinValue, pch.MaxValue}
			}
			controller.optimizer = bayesian.NewOptimizer(bounds)
		}

		// Initialize last outputs to midpoint of ranges
		for _, patch := range cfg.OutputConfigPatches {
			// Start with midpoint of range as default
			defaultValue := (patch.MinValue + patch.MaxValue) / 2
			controller.lastOutputs[patch.ParameterPath] = defaultValue
		}

		p.controllers = append(p.controllers, controller)
	}

	return p, nil
}

// NewProcessor creates a new pid_decider processor - exported for testing
func NewProcessor(config *Config, settings component.TelemetrySettings, id component.ID) (*pidProcessor, error) {
	return newProcessor(config, settings, id)
}

// Start implements the Component interface
func (p *pidProcessor) Start(ctx context.Context, host component.Host) error {
	return p.BaseProcessor.Start(ctx, host)
}

// Shutdown implements the Component interface
func (p *pidProcessor) Shutdown(ctx context.Context) error {
	return p.BaseProcessor.Shutdown(ctx)
}

// Capabilities implements the processor.Metrics interface
func (p *pidProcessor) Capabilities() consumer.Capabilities {
	return p.BaseProcessor.Capabilities()
}

// ConsumeMetrics processes incoming metrics and submits patches to pic_control
func (p *pidProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Only acquire a read lock during processing to allow parallel metric processing
	p.RLock()
	patches, err := p.processMetricsInternal(ctx, md)
	p.RUnlock()

	if err != nil {
		return err
	}

	// Forward metrics to next consumer if one is configured
	// Also outside our lock to prevent possible deadlocks
	if next := p.GetNext(); next != nil {
		return next.ConsumeMetrics(ctx, md)
	}

	return nil
}

// processMetricsInternal handles the actual processing of metrics
// Assumes the caller holds at least a read lock
func (p *pidProcessor) processMetricsInternal(ctx context.Context, md pmetric.Metrics) ([]interfaces.ConfigPatch, error) {
	// Extract KPI values from metrics
	kpiValues := extractKPIValues(md)
	if len(kpiValues) == 0 {
		return nil, nil
	}

	// Generate patches - we'll implement this in ProcessMetricsForTest
	patches := make([]interfaces.ConfigPatch, 0)

	// Process each enabled controller
	for _, controller := range p.controllers {
		if !controller.config.Enabled {
			continue
		}

		// Look for this controller's KPI metric
		kpiValue, exists := kpiValues[controller.config.KPIMetricName]
		if !exists {
			continue // Skip if KPI metric not found
		}

		// Compute PID output
		pidOutput := controller.pid.Compute(kpiValue)

		// Check for oscillation (will be used by circuit breaker later)
		oscillating := false
		if controller.lastPIDOut*pidOutput < 0 &&
			math.Abs(pidOutput) > 0.1 &&
			math.Abs(controller.lastPIDOut) > 0.1 {
			oscillating = true
		}
		controller.lastPIDOut = pidOutput

		// Generate patches based on the output
		controllerPatches := controller.generatePatches(ctx, pidOutput, kpiValue, oscillating)
		patches = append(patches, controllerPatches...)
	}

	return patches, nil
}

// OnConfigPatch implements the UpdateableProcessor interface
func (p *pidProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.Lock()
	defer p.Unlock()

	// Helper to extract controller name from target processor name
	getControllerName := func(targetName component.ID) string {
		parts := strings.Split(targetName.String(), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return ""
	}

	// Helper to find controller by name
	findControllerIndex := func(name string) (int, bool) {
		for i, ctrl := range p.config.Controllers {
			if ctrl.Name == name {
				return i, true
			}
		}
		return -1, false
	}

	switch patch.ParameterPath {
	case "enabled":
		// Find the controller by name
		controllerName := getControllerName(patch.TargetProcessorName)
		if controllerName == "" {
			return fmt.Errorf("invalid target processor name: %s", patch.TargetProcessorName.String())
		}

		i, found := findControllerIndex(controllerName)
		if !found {
			return fmt.Errorf("controller not found: %s", controllerName)
		}

		// Convert to bool using type converter
		enabled, err := typeconv.ToBool(patch.NewValue)
		if err != nil {
			return fmt.Errorf("invalid value for enabled: %v", err)
		}

		// Update controller state
		p.config.Controllers[i].Enabled = enabled
		p.GetLogger().Info("Updated controller enabled state",
			zap.String("controller", controllerName),
			zap.Bool("enabled", enabled))
		return nil

	case "kpi_target_value":
		// Find the controller by name
		controllerName := getControllerName(patch.TargetProcessorName)
		if controllerName == "" {
			return fmt.Errorf("invalid target processor name: %s", patch.TargetProcessorName.String())
		}

		// Find controller in config
		configIdx, found := findControllerIndex(controllerName)
		if !found {
			return fmt.Errorf("controller not found in config: %s", controllerName)
		}

		// Find the runtime controller instance
		var ctrlInstance *controller
		found = false
		for _, ctrl := range p.controllers {
			if ctrl.config.Name == controllerName {
				ctrlInstance = ctrl
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("controller runtime instance not found: %s", controllerName)
		}

		// Convert to float64 using type converter
		targetValue, err := typeconv.ToFloat64(patch.NewValue)
		if err != nil {
			return fmt.Errorf("invalid value for kpi_target_value: %v", err)
		}

		// Update the controller configuration
		p.config.Controllers[configIdx].KPITargetValue = targetValue

		// Update the PID controller's setpoint
		ctrlInstance.pid.SetSetpoint(targetValue)

		// Reset integral term when setpoint changes significantly to prevent windup
		ctrlInstance.pid.ResetIntegral()

		p.GetLogger().Info("Updated controller KPI target value",
			zap.String("controller", controllerName),
			zap.Float64("target_value", targetValue))
		return nil

	case "kp":
		// Find the controller by name
		controllerName := getControllerName(patch.TargetProcessorName)
		if controllerName == "" {
			return fmt.Errorf("invalid target processor name: %s", patch.TargetProcessorName.String())
		}

		// Find controller in config
		configIdx, found := findControllerIndex(controllerName)
		if !found {
			return fmt.Errorf("controller not found in config: %s", controllerName)
		}

		// Find the runtime controller instance
		var ctrlInstance *controller
		for _, ctrl := range p.controllers {
			if ctrl.config.Name == controllerName {
				ctrlInstance = ctrl
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("controller runtime instance not found: %s", controllerName)
		}

		// Convert to float64 using type converter
		kp, err := typeconv.ToFloat64(patch.NewValue)
		if err != nil {
			return fmt.Errorf("invalid value for kp: %v", err)
		}

		// Get current tuning values to update only kp
		_, currentKI, currentKD := ctrlInstance.pid.GetTunings()

		// Update the controller configuration
		p.config.Controllers[configIdx].KP = kp

		// Update the PID controller's tuning
		ctrlInstance.pid.SetTunings(kp, currentKI, currentKD)

		p.GetLogger().Info("Updated controller proportional gain",
			zap.String("controller", controllerName),
			zap.Float64("kp", kp))
		return nil

	case "ki":
		// Find the controller by name
		controllerName := getControllerName(patch.TargetProcessorName)
		if controllerName == "" {
			return fmt.Errorf("invalid target processor name: %s", patch.TargetProcessorName.String())
		}

		// Find controller in config
		configIdx, found := findControllerIndex(controllerName)
		if !found {
			return fmt.Errorf("controller not found in config: %s", controllerName)
		}

		// Find the runtime controller instance
		var ctrlInstance *controller
		for _, ctrl := range p.controllers {
			if ctrl.config.Name == controllerName {
				ctrlInstance = ctrl
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("controller runtime instance not found: %s", controllerName)
		}

		// Convert to float64 using type converter
		ki, err := typeconv.ToFloat64(patch.NewValue)
		if err != nil {
			return fmt.Errorf("invalid value for ki: %v", err)
		}

		// Get current tuning values to update only ki
		currentKP, _, currentKD := ctrlInstance.pid.GetTunings()

		// Update the controller configuration
		p.config.Controllers[configIdx].KI = ki

		// Update the PID controller's tuning
		ctrlInstance.pid.SetTunings(currentKP, ki, currentKD)

		// Reset integral term when ki changes to prevent windup
		ctrlInstance.pid.ResetIntegral()

		p.GetLogger().Info("Updated controller integral gain",
			zap.String("controller", controllerName),
			zap.Float64("ki", ki))
		return nil

	case "kd":
		// Find the controller by name
		controllerName := getControllerName(patch.TargetProcessorName)
		if controllerName == "" {
			return fmt.Errorf("invalid target processor name: %s", patch.TargetProcessorName.String())
		}

		// Find controller in config
		configIdx, found := findControllerIndex(controllerName)
		if !found {
			return fmt.Errorf("controller not found in config: %s", controllerName)
		}

		// Find the runtime controller instance
		var ctrlInstance *controller
		for _, ctrl := range p.controllers {
			if ctrl.config.Name == controllerName {
				ctrlInstance = ctrl
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("controller runtime instance not found: %s", controllerName)
		}

		// Convert to float64 using type converter
		kd, err := typeconv.ToFloat64(patch.NewValue)
		if err != nil {
			return fmt.Errorf("invalid value for kd: %v", err)
		}

		// Get current tuning values to update only kd
		currentKP, currentKI, _ := ctrlInstance.pid.GetTunings()

		// Update the controller configuration
		p.config.Controllers[configIdx].KD = kd

		// Update the PID controller's tuning
		ctrlInstance.pid.SetTunings(currentKP, currentKI, kd)

		p.GetLogger().Info("Updated controller derivative gain",
			zap.String("controller", controllerName),
			zap.Float64("kd", kd))
		return nil

	default:
		return fmt.Errorf("unsupported parameter: %s", patch.ParameterPath)
	}
}

// GetName returns the processor name for identification
func (p *pidProcessor) GetName() string {
	return "adaptive_pid"
}

// GetConfigStatus implements the UpdateableProcessor interface
func (p *pidProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.RLock()
	defer p.RUnlock()

	// Determine if any controllers are enabled
	enabled := false
	for _, ctrl := range p.config.Controllers {
		if ctrl.Enabled {
			enabled = true
			break
		}
	}

	// Convert controllers to a map for the status
	controllers := make([]map[string]interface{}, len(p.config.Controllers))
	for i, ctrl := range p.config.Controllers {
		controllers[i] = map[string]interface{}{
			"name":             ctrl.Name,
			"enabled":          ctrl.Enabled,
			"kpi_metric_name":  ctrl.KPIMetricName,
			"kpi_target_value": ctrl.KPITargetValue,
		}
	}

	return interfaces.ConfigStatus{
		Parameters: map[string]interface{}{
			"controllers": controllers,
		},
		Enabled: enabled,
	}, nil
}
