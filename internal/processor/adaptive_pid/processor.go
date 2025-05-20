// Package adaptive_pid implements a processor that uses PID control for adaptive configuration.
package adaptive_pid

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/control/pid"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
	"github.com/deepaucksharma/Phoenix/pkg/util/bayesian"
	"github.com/deepaucksharma/Phoenix/pkg/util/typeconv"
)

var _ component.Config = (*Config)(nil)

// Interface for pic_control extension
type picControl interface {
	SubmitConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error
}

// pidProcessor implements the pid_decider processor
type pidProcessor struct {
	logger       *zap.Logger
	nextConsumer consumer.Metrics
	config       *Config
	controllers  []*controller
	lock         sync.RWMutex
	metrics      *metrics.MetricsEmitter
	picControl   picControl // Interface to pic_control_ext
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
func newProcessor(config *Config, settings component.TelemetrySettings, picControlExt picControl, id component.ID) (*pidProcessor, error) {
	p := &pidProcessor{
		logger:       settings.Logger,
		nextConsumer: nil, // Not used directly, patches are submitted to pic_control
		config:       config,
		controllers:  make([]*controller, 0, len(config.Controllers)),
		picControl:   picControlExt,
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
func NewProcessor(config *Config, settings component.TelemetrySettings, picControlExt interface{}, id component.ID) (*pidProcessor, error) {
	// Handle case where picControlExt could be nil or not implement picControl
	var picCtrl picControl
	if picControlExt != nil {
		if pc, ok := picControlExt.(picControl); ok {
			picCtrl = pc
		}
	}
	
	return newProcessor(config, settings, picCtrl, id)
}

// Start implements the Component interface
func (p *pidProcessor) Start(ctx context.Context, host component.Host) error {
	// No initialization required for now
	return nil
}

// Shutdown implements the Component interface
func (p *pidProcessor) Shutdown(ctx context.Context) error {
	return nil
}

// Capabilities implements the processor.Metrics interface
func (p *pidProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics processes incoming metrics and submits patches to pic_control
func (p *pidProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Process metrics and generate patches
	patches, err := p.ProcessMetricsForTest(ctx, md)
	if err != nil {
		return err
	}
	
	// Submit patches to pic_control if available
	if p.picControl != nil {
		for _, patch := range patches {
			err := p.picControl.SubmitConfigPatch(ctx, patch)
			if err != nil {
				p.logger.Warn("Failed to submit config patch", 
					zap.String("patch_id", patch.PatchID),
					zap.Error(err))
			}
		}
	}
	
	// Forward metrics to next consumer if one is configured
	if p.nextConsumer != nil {
		return p.nextConsumer.ConsumeMetrics(ctx, md)
	}
	
	return nil
}

// OnConfigPatch implements the UpdateableProcessor interface
func (p *pidProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.lock.Lock()
	defer p.lock.Unlock()

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
		p.logger.Info("Updated controller enabled state",
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
		var ctrlInstanceIdx int
		found = false
		for i, ctrl := range p.controllers {
			if ctrl.config.Name == controllerName {
				ctrlInstance = ctrl
				ctrlInstanceIdx = i
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

		p.logger.Info("Updated controller KPI target value",
			zap.String("controller", controllerName),
			zap.Float64("target_value", targetValue))
		return nil

	default:
		return fmt.Errorf("unsupported parameter: %s", patch.ParameterPath)
	}
}

// GetConfigStatus implements the UpdateableProcessor interface
func (p *pidProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

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
