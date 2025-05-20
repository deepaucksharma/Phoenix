package adaptive_pid

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

// Helper function to create a configuration patch for testing
func createConfigPatch(targetProcessor, paramPath string, value float64, controllerName string) interfaces.ConfigPatch {
	return interfaces.ConfigPatch{
		PatchID:             uuid.New().String(),
		TargetProcessorName: component.NewIDWithName(component.MustNewType("processor"), targetProcessor),
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
	p.lock.RLock() // Use read lock to allow parallel processing
	defer p.lock.RUnlock()
	
	return p.processMetricsInternal(ctx, md)
}

// generatePatches creates configuration patches based on PID controller output
func (c *controller) generatePatches(ctx context.Context, pidOutput float64, kpiValue float64, oscillating bool) []interfaces.ConfigPatch {
	patches := make([]interfaces.ConfigPatch, 0, len(c.config.OutputConfigPatches))
	
	// If we're using Bayesian optimization and the system is oscillating,
	// try a new point in parameter space
	if c.config.UseBayesian && c.optimizer != nil && oscillating {
		c.stallCount++
		
		// If the system has been oscillating for too long, try a new point
		if c.stallCount >= c.config.StallThreshold {
			c.stallCount = 0
			return c.generateBayesianOptimizationPatches(ctx, kpiValue)
		}
	}
	
	// Normal PID-based patches
	for _, patchCfg := range c.config.OutputConfigPatches {
		// Get current value as baseline
		currentValue, exists := c.lastOutputs[patchCfg.ParameterPath]
		if !exists {
			// Use midpoint of range as default
			currentValue = (patchCfg.MinValue + patchCfg.MaxValue) / 2
		}
		
		// Apply scaling factor to the output
		scaledDelta := pidOutput * patchCfg.ChangeScaleFactor
		
		// Calculate new value
		newValue := currentValue + scaledDelta
		
		// Clamp to min/max
		if newValue < patchCfg.MinValue {
			newValue = patchCfg.MinValue
		} else if newValue > patchCfg.MaxValue {
			newValue = patchCfg.MaxValue
		}
		
		// Check against hysteresis threshold
		if currentValue != 0 {
			changePercent := math.Abs((newValue - currentValue) / currentValue * 100)
			if c.config.HysteresisPercent > 0 && 
			   changePercent < c.config.HysteresisPercent {
				// Change too small, skip
				continue
			}
		}
		
		// Skip if the change is negligible
		if math.Abs(newValue - currentValue) < 0.001 {
			continue
		}
		
		// Create the patch
		patch := interfaces.ConfigPatch{
			PatchID:             fmt.Sprintf("%s-%d", c.config.Name, time.Now().UnixNano()),
			TargetProcessorName: component.NewIDWithName(component.MustNewType("processor"), patchCfg.TargetProcessorName),
			ParameterPath:       patchCfg.ParameterPath,
			NewValue:            newValue,
			PrevValue:           currentValue,
			Reason:              fmt.Sprintf("PID adjustment based on KPI=%f, target=%f", kpiValue, c.config.KPITargetValue),
			Severity:            "normal",
			Source:              "adaptive_pid",
			Timestamp:           time.Now().Unix(),
			TTLSeconds:          300, // 5 minute expiry
			SafetyOverride:      false,
		}
		
		// Remember the last output for this parameter
		c.lastOutputs[patchCfg.ParameterPath] = newValue
		
		// Add to list of patches
		patches = append(patches, patch)
	}
	
	return patches
}

// generateBayesianOptimizationPatches creates patches based on Bayesian optimization
func (c *controller) generateBayesianOptimizationPatches(ctx context.Context, kpiValue float64) []interfaces.ConfigPatch {
	patches := make([]interfaces.ConfigPatch, 0, len(c.config.OutputConfigPatches))
	
	// If this is the first time, just suggest the midpoint
	if c.optimizer == nil || len(c.lastValues) == 0 {
		return patches
	}
	
	// Create a point from the current parameter values
	currentPoint := make([]float64, len(c.config.OutputConfigPatches))
	for i, patchCfg := range c.config.OutputConfigPatches {
		currentPoint[i] = c.lastOutputs[patchCfg.ParameterPath]
	}
	
	// Add the current point with its performance metric
	// For KPI metrics where higher is better:
	performance := kpiValue
	// For KPI metrics where lower is better, you might use:
	// performance = -kpiValue or performance = 1/kpiValue
	
	c.lastValues[c.config.KPIMetricName] = kpiValue
	c.optimizer.AddSample(currentPoint, performance)
	
	// Get new suggested point
	newPoint := c.optimizer.Suggest()
	
	// Generate patches for the new point
	for i, patchCfg := range c.config.OutputConfigPatches {
		currentValue := c.lastOutputs[patchCfg.ParameterPath]
		newValue := newPoint[i]
		
		// Apply limits
		if newValue < patchCfg.MinValue {
			newValue = patchCfg.MinValue
		} else if newValue > patchCfg.MaxValue {
			newValue = patchCfg.MaxValue
		}
		
		// Skip if the change is negligible
		if math.Abs(newValue - currentValue) < 0.001 {
			continue
		}
		
		// Create the patch
		patch := interfaces.ConfigPatch{
			PatchID:             fmt.Sprintf("%s-bayes-%d", c.config.Name, time.Now().UnixNano()),
			TargetProcessorName: component.NewIDWithName(component.MustNewType("processor"), patchCfg.TargetProcessorName),
			ParameterPath:       patchCfg.ParameterPath,
			NewValue:            newValue,
			PrevValue:           currentValue,
			Reason:              fmt.Sprintf("Bayesian optimization step, current KPI=%f", kpiValue),
			Severity:            "normal",
			Source:              "adaptive_pid_bayesian",
			Timestamp:           time.Now().Unix(),
			TTLSeconds:          300, // 5 minute expiry
			SafetyOverride:      false,
		}
		
		// Remember the last output for this parameter
		c.lastOutputs[patchCfg.ParameterPath] = newValue
		
		// Add to list of patches
		patches = append(patches, patch)
	}
	
	return patches
}