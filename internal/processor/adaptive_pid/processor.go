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
		pidController, err := pid.NewController(cfg.KP, cfg.KI, cfg.KD, cfg.KPITargetValue)
		if err != nil {
			return nil, fmt.Errorf("create PID controller: %w", err)
		}

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

	parts := strings.Split(patch.TargetProcessorName.String(), "/")
	controllerName := parts[len(parts)-1]

	// Locate controller
	var (
		idx  int
		ctrl *controller
	)
	for i, c := range p.controllers {
		if c.config.Name == controllerName {
			idx = i
			ctrl = c
			break
		}
	}
	if ctrl == nil {
		return fmt.Errorf("controller not found: %s", patch.TargetProcessorName.String())
	}

	switch patch.ParameterPath {
	case "enabled":
		enabled, ok := patch.NewValue.(bool)
		if !ok {
			return fmt.Errorf("invalid value type for enabled: %T", patch.NewValue)
		}
		p.config.Controllers[idx].Enabled = enabled
		ctrl.config.Enabled = enabled
		return nil

	case "kpi_target_value":
		targetValue, ok := patch.NewValue.(float64)
		if !ok {
			return fmt.Errorf("invalid value type for kpi_target_value: %T", patch.NewValue)
		}
		p.config.Controllers[idx].KPITargetValue = targetValue
		ctrl.config.KPITargetValue = targetValue
		ctrl.pid.SetSetpoint(targetValue)
		return nil

	case "kp":
		val, ok := patch.NewValue.(float64)
		if !ok {
			return fmt.Errorf("invalid value type for kp: %T", patch.NewValue)
		}
		p.config.Controllers[idx].KP = val
		ctrl.config.KP = val
		ctrl.pid.SetTunings(val, ctrl.config.KI, ctrl.config.KD)
		return nil

	case "ki":
		val, ok := patch.NewValue.(float64)
		if !ok {
			return fmt.Errorf("invalid value type for ki: %T", patch.NewValue)
		}
		p.config.Controllers[idx].KI = val
		ctrl.config.KI = val
		ctrl.pid.SetTunings(ctrl.config.KP, val, ctrl.config.KD)
		return nil

	case "kd":
		val, ok := patch.NewValue.(float64)
		if !ok {
			return fmt.Errorf("invalid value type for kd: %T", patch.NewValue)
		}
		p.config.Controllers[idx].KD = val
		ctrl.config.KD = val
		ctrl.pid.SetTunings(ctrl.config.KP, ctrl.config.KI, val)
		return nil

	case "integral_windup_limit":
		val, ok := patch.NewValue.(float64)
		if !ok {
			return fmt.Errorf("invalid value type for integral_windup_limit: %T", patch.NewValue)
		}
		p.config.Controllers[idx].IntegralWindupLimit = val
		ctrl.config.IntegralWindupLimit = val
		ctrl.pid.SetIntegralLimit(val)
		return nil

	case "hysteresis_percent":
		val, ok := patch.NewValue.(float64)
		if !ok {
			return fmt.Errorf("invalid value type for hysteresis_percent: %T", patch.NewValue)
		}
		p.config.Controllers[idx].HysteresisPercent = val
		ctrl.config.HysteresisPercent = val
		return nil

	case "kpi_metric_name":
		name, ok := patch.NewValue.(string)
		if !ok {
			return fmt.Errorf("invalid value type for kpi_metric_name: %T", patch.NewValue)
		}
		p.config.Controllers[idx].KPIMetricName = name
		ctrl.config.KPIMetricName = name
		return nil

	case "use_bayesian":
		v, ok := patch.NewValue.(bool)
		if !ok {
			return fmt.Errorf("invalid value type for use_bayesian: %T", patch.NewValue)
		}
		p.config.Controllers[idx].UseBayesian = v
		ctrl.config.UseBayesian = v
		if v && ctrl.optimizer == nil {
			bounds := make([][2]float64, len(ctrl.config.OutputConfigPatches))
			for i, pch := range ctrl.config.OutputConfigPatches {
				bounds[i] = [2]float64{pch.MinValue, pch.MaxValue}
			}
			ctrl.optimizer = bayesian.NewOptimizer(bounds)
		}
		if !v {
			ctrl.optimizer = nil
		}
		return nil

	case "stall_threshold":
		v, ok := patch.NewValue.(int)
		if !ok {
			return fmt.Errorf("invalid value type for stall_threshold: %T", patch.NewValue)
		}
		p.config.Controllers[idx].StallThreshold = v
		ctrl.config.StallThreshold = v
		return nil
	}

	return fmt.Errorf("unsupported parameter: %s", patch.ParameterPath)
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
