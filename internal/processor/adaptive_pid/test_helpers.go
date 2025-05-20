package adaptive_pid

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

// Helper function to create a configuration patch for testing
func createConfigPatch(targetProcessor, paramPath string, value float64, controllerName string) interfaces.ConfigPatch {
	return interfaces.ConfigPatch{
		PatchID:             uuid.New().String(),
		TargetProcessorName: component.NewIDWithName("processor", targetProcessor),
		ParameterPath:       paramPath,
		NewValue:            value,
		Reason:              fmt.Sprintf("Adjustment from %s controller", controllerName),
		Severity:            "normal",
		Source:              "pid_decider",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300, // 5 minute TTL
	}
}

// Helper function to generate a reason string for patches
func generateReason(controllerName string, error, output float64) string {
	return fmt.Sprintf("PID controller %s adjustment: error=%.3f, output=%.3f", 
		controllerName, error, output)
}

// ProcessMetricsForTest is a test helper that processes metrics and returns generated patches.
func (p *pidProcessor) ProcessMetricsForTest(ctx context.Context, md pmetric.Metrics) ([]interfaces.ConfigPatch, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	
	// Extract KPI values from metrics
	kpiValues := extractKPIValues(md)
	
	// Store patches to return
	patches := make([]interfaces.ConfigPatch, 0)
	
	// Process each controller
	for _, ctrl := range p.controllers {
		// Get current KPI value
		kpiValue, found := kpiValues[ctrl.config.KPIMetricName]
		if !found {
			// KPI not found in this batch, emit metric and continue
			p.logger.Warn("KPI not found in metrics batch", 
				zap.String("controller", ctrl.config.Name),
				zap.String("kpi", ctrl.config.KPIMetricName))
			
			// Emit metric if available
			if p.metrics != nil {
				p.metrics.AddMetric("aemf_pid_decider_kpi_missing_total", 1)
			}
			continue
		}
		
		// Save current value
		ctrl.lastValues[ctrl.config.KPIMetricName] = kpiValue
		
		// Update PID controller and get output
		output := ctrl.pid.Compute(kpiValue)
		
		// Generate ConfigPatch for each output parameter
		for _, outConfig := range ctrl.config.OutputConfigPatches {
			// Apply scaling factor to the output
			scaledDelta := output * outConfig.ChangeScaleFactor
			
			// Get last output value for this parameter
			lastValue := ctrl.lastOutputs[outConfig.ParameterPath]
			
			// Calculate new value
			newValue := lastValue + scaledDelta
			
			// Raw value before clamping (for metrics)
			rawValue := newValue
			
			// Clamp to min/max
			if newValue < outConfig.MinValue {
				if p.metrics != nil {
					p.metrics.AddMetric("aemf_pid_output_clamped_total", 1)
				}
				newValue = outConfig.MinValue
			} else if newValue > outConfig.MaxValue {
				if p.metrics != nil {
					p.metrics.AddMetric("aemf_pid_output_clamped_total", 1)
				}
				newValue = outConfig.MaxValue
			}
			
			// Check against hysteresis threshold
			if lastValue != 0 {
				changePercent := (newValue - lastValue) / lastValue * 100
				if ctrl.config.HysteresisPercent > 0 && 
				   changePercent < ctrl.config.HysteresisPercent {
					// Change too small, skip
					continue
				}
			}
			
			// Generate patch
			patch := interfaces.ConfigPatch{
				PatchID:             uuid.New().String(),
				TargetProcessorName: component.NewIDWithName("processor", outConfig.TargetProcessorName),
				ParameterPath:       outConfig.ParameterPath,
				NewValue:            newValue,
				Reason:              generateReason(ctrl.config.Name, ctrl.config.KPITargetValue-kpiValue, output),
				Severity:            "normal",
				Source:              "pid_decider",
				Timestamp:           time.Now().Unix(),
				TTLSeconds:          300, // 5 minute TTL
			}
			
			// Add patch to result
			patches = append(patches, patch)
			
			// Update last output value
			ctrl.lastOutputs[outConfig.ParameterPath] = newValue
		}
	}
	
	return patches, nil
}