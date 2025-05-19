// Package adaptivepid implements the pid_decider processor which generates configuration
// patches using PID control loops to maintain KPI targets.
package adaptivepid

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/yourorg/sa-omf/internal/control/pid"
	"github.com/yourorg/sa-omf/internal/interfaces"
	"github.com/yourorg/sa-omf/pkg/metrics"
)

// processorImpl is the implementation of the pid_decider processor.
type processorImpl struct {
	config      *Config
	logger      *zap.Logger
	next        consumer.Metrics
	controllers []*controller
	lock        sync.RWMutex
	lastValues  map[string]float64      // Last observed values for each KPI
	patchTimes  map[string]time.Time    // Last patch time for each parameter
	metricsEmitter *metrics.MetricsEmitter
}

// controller encapsulates a single PID control loop.
type controller struct {
	config        ControllerConfig
	pidController *pid.Controller
	lastOutputs   map[string]float64 // Last output for each parameter
}

// Ensure processorImpl implements the required interfaces.
var _ processor.Metrics = (*processorImpl)(nil)

// newProcessor creates a new pid_decider processor.
func newProcessor(cfg *Config, settings processor.CreateSettings, nextConsumer consumer.Metrics) (*processorImpl, error) {
	p := &processorImpl{
		config:     cfg,
		logger:     settings.Logger,
		next:       nextConsumer,
		lastValues: make(map[string]float64),
		patchTimes: make(map[string]time.Time),
	}

	// Initialize controllers
	p.controllers = make([]*controller, 0, len(cfg.Controllers))
	for _, ctrlConfig := range cfg.Controllers {
		if !ctrlConfig.Enabled {
			continue
		}

		// Create PID controller
		pidCtrl := pid.NewController(
			ctrlConfig.KP,
			ctrlConfig.KI,
			ctrlConfig.KD,
			ctrlConfig.KPITargetValue,
		)
		
		// Set integral windup limit if configured
		if ctrlConfig.IntegralWindupLimit > 0 {
			pidCtrl.SetIntegralLimit(ctrlConfig.IntegralWindupLimit)
		}

		// Create controller
		ctrl := &controller{
			config:        ctrlConfig,
			pidController: pidCtrl,
			lastOutputs:   make(map[string]float64),
		}

		// Initialize lastOutputs with reasonable default values
		for _, patch := range ctrlConfig.OutputConfigPatches {
			paramKey := fmt.Sprintf("%s.%s", patch.TargetProcessorName, patch.ParameterPath)
			ctrl.lastOutputs[paramKey] = patch.MinValue + (patch.MaxValue-patch.MinValue)/2
		}

		p.controllers = append(p.controllers, ctrl)
	}

	return p, nil
}

// Start implements the Component interface.
func (p *processorImpl) Start(ctx context.Context, host component.Host) error {
	// Set up metrics if provider available
	return nil
}

// Shutdown implements the Component interface.
func (p *processorImpl) Shutdown(ctx context.Context) error {
	return nil
}

// Capabilities implements the processor.Metrics interface.
func (p *processorImpl) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics implements the consumer.Metrics interface.
func (p *processorImpl) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Extract KPI values from incoming metrics
	kpiValues := p.extractKPIValues(md)

	// Process each controller
	for _, ctrl := range p.controllers {
		// Get current KPI value
		kpiValue, found := kpiValues[ctrl.config.KPIMetricName]
		if !found {
			// KPI metric not found in this batch
			continue
		}

		// Calculate error (target - actual)
		error := ctrl.config.KPITargetValue - kpiValue

		// Update PID controller
		output := ctrl.pidController.Compute(kpiValue)

		// Generate ConfigPatch for each output parameter
		for _, outConfig := range ctrl.config.OutputConfigPatches {
			paramKey := fmt.Sprintf("%s.%s", outConfig.TargetProcessorName, outConfig.ParameterPath)
			
			// Apply scaling factor to the output
			scaledOutput := output * outConfig.ChangeScaleFactor

			// Check last output against hysteresis
			lastOutput, exists := ctrl.lastOutputs[paramKey]
			if !exists {
				// Initialize with middle of range for first run
				lastOutput = outConfig.MinValue + (outConfig.MaxValue-outConfig.MinValue)/2
				ctrl.lastOutputs[paramKey] = lastOutput
			}

			// Calculate percentage change
			var changePercent float64
			if lastOutput != 0 {
				changePercent = (scaledOutput - lastOutput) / math.Abs(lastOutput) * 100
			} else {
				changePercent = 100 // Treat as 100% change if lastOutput is 0
			}

			// Skip if change is too small (hysteresis)
			if math.Abs(changePercent) < ctrl.config.HysteresisPercent {
				continue
			}

			// Calculate new value
			newValue := lastOutput + scaledOutput

			// Clamp the value to configured limits
			if newValue < outConfig.MinValue {
				newValue = outConfig.MinValue
			} else if newValue > outConfig.MaxValue {
				newValue = outConfig.MaxValue
			}

			// Generate patch
			patch := interfaces.ConfigPatch{
				PatchID:             uuid.New().String(),
				TargetProcessorName: component.MustNewIDFromString(outConfig.TargetProcessorName),
				ParameterPath:       outConfig.ParameterPath,
				NewValue:            newValue,
				Reason:              p.generateReason(ctrl.config.Name, error, output),
				Severity:            "normal",
				Source:              "pid_decider",
				Timestamp:           time.Now().Unix(),
				TTLSeconds:          300, // 5 minute TTL
			}

			// Emit as metric with attributes
			p.emitPatchAsMetric(ctx, patch)

			// Update last output value
			ctrl.lastOutputs[paramKey] = newValue
		}
	}

	// Pass the metrics to the next consumer
	return p.next.ConsumeMetrics(ctx, md)
}

// extractKPIValues extracts KPI values from incoming metrics.
func (p *processorImpl) extractKPIValues(md pmetric.Metrics) map[string]float64 {
	kpiValues := make(map[string]float64)

	// Iterate through all metrics
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				
				// Check if this metric is one of our KPIs
				isKPI := false
				for _, ctrl := range p.controllers {
					if metric.Name() == ctrl.config.KPIMetricName {
						isKPI = true
						break
					}
				}
				
				if !isKPI {
					continue
				}
				
				// Extract value based on metric type
				var value float64
				
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					if metric.Gauge().DataPoints().Len() > 0 {
						dp := metric.Gauge().DataPoints().At(0)
						value = dp.DoubleValue()
					}
					
				case pmetric.MetricTypeSum:
					if metric.Sum().DataPoints().Len() > 0 {
						dp := metric.Sum().DataPoints().At(0)
						value = dp.DoubleValue()
					}
					
				default:
					// Unsupported metric type for KPI
					continue
				}
				
				kpiValues[metric.Name()] = value
				
				// Update last seen value
				p.lastValues[metric.Name()] = value
			}
		}
	}
	
	return kpiValues
}

// generateReason creates a human-readable reason for the patch.
func (p *processorImpl) generateReason(controllerName string, error float64, output float64) string {
	direction := "increase"
	if output < 0 {
		direction = "decrease"
	}
	
	return fmt.Sprintf(
		"Controller '%s' detected KPI error of %.3f, recommending %s with magnitude %.3f",
		controllerName,
		error,
		direction,
		math.Abs(output),
	)
}

// emitPatchAsMetric emits a ConfigPatch as a metric for the pic_connector.
func (p *processorImpl) emitPatchAsMetric(ctx context.Context, patch interfaces.ConfigPatch) {
	// If we have a metrics emitter, use it to emit the patch
	if p.metricsEmitter != nil {
		// Ideally use the metrics emitter, but that requires more setup...
	}
	
	// Log the patch for now
	p.logger.Info("Generated config patch",
		zap.String("patch_id", patch.PatchID),
		zap.String("target", patch.TargetProcessorName.String()),
		zap.String("parameter", patch.ParameterPath),
		zap.Any("new_value", patch.NewValue),
		zap.String("reason", patch.Reason),
	)
}