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

// This const is defined in factory.go
// var typeStr = "pid_decider"

// Removed duplicate Config, ControllerConfig, and OutputConfigPatch definitions
// These are already defined in config.go

var _ component.Config = (*Config)(nil)

// pidProcessor implements the pid_decider processor
type pidProcessor struct {
	logger       *zap.Logger
	nextConsumer consumer.Metrics
	config       *Config
	controllers  []*controller
	lock         sync.RWMutex
	metrics      *metrics.MetricsEmitter
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
func newProcessor(config *Config, settings component.TelemetrySettings, nextConsumer consumer.Metrics, id component.ID) (*pidProcessor, error) {
	p := &pidProcessor{
		logger:       settings.Logger,
		nextConsumer: nextConsumer,
		config:       config,
		controllers:  make([]*controller, 0, len(config.Controllers)),
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
func NewProcessor(config *Config, settings component.TelemetrySettings, nextConsumer consumer.Metrics, id component.ID) (*pidProcessor, error) {
	return newProcessor(config, settings, nextConsumer, id)
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

// ConsumeMetrics processes incoming metrics
func (p *pidProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Extract KPI values from metrics
	kpiValues := extractKPIValues(md)

	// Process each controller
	for _, ctrl := range p.controllers {
		// Get current KPI value
		kpiValue, found := kpiValues[ctrl.config.KPIMetricName]
		if !found {
			// KPI not found in this batch, try next controller
			continue
		}

		// Save current value
		ctrl.lastValues[ctrl.config.KPIMetricName] = kpiValue

		// Update PID controller and get output
		output := ctrl.pid.Compute(kpiValue)

		if ctrl.optimizer != nil {
			if math.Abs(output-ctrl.lastPIDOut) < 1e-3 {
				ctrl.stallCount++
			} else {
				ctrl.stallCount = 0
			}
			ctrl.lastPIDOut = output

			// Add sample for optimizer based on current outputs
			sample := make([]float64, len(ctrl.config.OutputConfigPatches))
			for i, pch := range ctrl.config.OutputConfigPatches {
				sample[i] = ctrl.lastOutputs[pch.ParameterPath]
			}
			score := -math.Abs(ctrl.config.KPITargetValue - kpiValue)
			ctrl.optimizer.AddSample(sample, score)

			if ctrl.stallCount >= ctrl.config.StallThreshold {
				suggestion := ctrl.optimizer.Suggest()
				for i, outConfig := range ctrl.config.OutputConfigPatches {
					newValue := suggestion[i]
					if newValue < outConfig.MinValue {
						newValue = outConfig.MinValue
					} else if newValue > outConfig.MaxValue {
						newValue = outConfig.MaxValue
					}
					ctrl.lastOutputs[outConfig.ParameterPath] = newValue
					patch := interfaces.ConfigPatch{
						PatchID:             uuid.New().String(),
						TargetProcessorName: component.NewID(component.MustNewType(typeStr)),
						ParameterPath:       outConfig.ParameterPath,
						NewValue:            newValue,
						Reason:              "bayesian_fallback",
						Severity:            "normal",
						Source:              "pid_decider",
						Timestamp:           time.Now().Unix(),
						TTLSeconds:          300,
					}
					p.logger.Info("Bayesian patch", zap.String("controller", ctrl.config.Name), zap.String("patch_id", patch.PatchID))
				}
				ctrl.stallCount = 0
				continue
			}
		}

		// Generate ConfigPatch for each output parameter
		for _, outConfig := range ctrl.config.OutputConfigPatches {
			// Apply scaling factor to the output
			scaledDelta := output * outConfig.ChangeScaleFactor

			// Get last output value for this parameter
			lastValue := ctrl.lastOutputs[outConfig.ParameterPath]

			// Calculate new value
			newValue := lastValue + scaledDelta

			// Clamp to min/max
			if newValue < outConfig.MinValue {
				newValue = outConfig.MinValue
			} else if newValue > outConfig.MaxValue {
				newValue = outConfig.MaxValue
			}

			// Check against hysteresis threshold
			if lastValue != 0 {
				changePercent := (newValue - lastValue) / lastValue * 100
				if math.Abs(changePercent) < ctrl.config.HysteresisPercent {
					// Change too small, skip
					continue
				}
			}

			// If we get here, the change is significant enough to emit a patch

			// Generate patch
			patch := interfaces.ConfigPatch{
				PatchID:             uuid.New().String(),
				TargetProcessorName: component.NewID(component.MustNewType(typeStr)),
				ParameterPath:       outConfig.ParameterPath,
				NewValue:            newValue,
				Reason:              generateReason(ctrl.config.Name, ctrl.config.KPITargetValue-kpiValue, output),
				Severity:            "normal",
				Source:              "pid_decider",
				Timestamp:           time.Now().Unix(),
				TTLSeconds:          300, // 5 minute TTL
			}

			// Emit as metric with attributes
			// Note: In a full implementation, we'd add this patch to the output metrics
			// For now, we'll just log it
			p.logger.Info("Generated patch",
				zap.String("controller", ctrl.config.Name),
				zap.String("patch_id", patch.PatchID),
				zap.String("target", typeStr),
				zap.String("parameter", outConfig.ParameterPath),
				zap.Float64("new_value", newValue),
				zap.Float64("error", ctrl.config.KPITargetValue-kpiValue),
				zap.Float64("raw_output", output))

			// In a real implementation, we would create a metric for this patch
			// and add it to the output metrics. For now, we'll create a stub.

			// Update last output value
			ctrl.lastOutputs[outConfig.ParameterPath] = newValue
		}
	}

	// Forward metrics to next consumer
	return p.nextConsumer.ConsumeMetrics(ctx, md)
}

// extractKPIValues extracts KPI values from metrics
func extractKPIValues(md pmetric.Metrics) map[string]float64 {
	kpiValues := make(map[string]float64)

	// Iterate through all metrics
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)

				// Check metric name
				name := metric.Name()

				// Skip metrics that don't start with aemf_impact
				if !strings.HasPrefix(name, "aemf_impact") {
					continue
				}

				// Extract value based on metric type
				var value float64
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					if metric.Gauge().DataPoints().Len() > 0 {
						dp := metric.Gauge().DataPoints().At(metric.Gauge().DataPoints().Len() - 1)
						value = dp.DoubleValue()
						kpiValues[name] = value
					}
				case pmetric.MetricTypeSum:
					if metric.Sum().DataPoints().Len() > 0 {
						dp := metric.Sum().DataPoints().At(metric.Sum().DataPoints().Len() - 1)
						value = dp.DoubleValue()
						kpiValues[name] = value
					}
				}
			}
		}
	}

	return kpiValues
}

// generateReason creates a human-readable reason for a patch
func generateReason(controllerName string, error float64, output float64) string {
	direction := "increase"
	if output < 0 {
		direction = "decrease"
	}

	return fmt.Sprintf("%s: Adjusting parameter to %s coverage (error: %.3f, pid_output: %.3f)",
		controllerName, direction, error, output)
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
