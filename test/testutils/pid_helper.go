package testutils

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"

	"github.com/deepaucksharma/Phoenix/internal/control/pid"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

// PIDControlHelper simplifies testing of PID controllers with the rest of the system
type PIDControlHelper struct {
	Controllers map[string]*pid.Controller
	KPIValues   map[string]float64
}

// NewPIDControlHelper creates a new helper for testing PID controllers
func NewPIDControlHelper() *PIDControlHelper {
	return &PIDControlHelper{
		Controllers: make(map[string]*pid.Controller),
		KPIValues:   make(map[string]float64),
	}
}

// AddController adds a new PID controller to the helper
func (h *PIDControlHelper) AddController(name string, kp, ki, kd, setpoint float64) {
	h.Controllers[name] = pid.NewController(kp, ki, kd, setpoint)
}

// SetKPIValue updates a KPI value for the next control cycle
func (h *PIDControlHelper) SetKPIValue(kpiName string, value float64) {
	h.KPIValues[kpiName] = value
}

// GenerateConfigPatches creates config patches from the current KPI values
func (h *PIDControlHelper) GenerateConfigPatches(ctx context.Context, controllerConfig map[string]ControllerMapping) []interfaces.ConfigPatch {
	var patches []interfaces.ConfigPatch

	for name, controller := range h.Controllers {
		mapping, exists := controllerConfig[name]
		if !exists {
			continue
		}

		// Get the KPI value
		kpiValue, exists := h.KPIValues[mapping.KPIMetricName]
		if !exists {
			continue
		}

		// Compute the output
		output := controller.Compute(kpiValue)

		// Scale output if needed
		scaledOutput := output * mapping.ScaleFactor

		// Apply min/max constraints
		if scaledOutput < mapping.MinValue {
			scaledOutput = mapping.MinValue
		} else if scaledOutput > mapping.MaxValue {
			scaledOutput = mapping.MaxValue
		}

		// Create the patch
		patch := interfaces.ConfigPatch{
			PatchID:             fmt.Sprintf("%s-patch-%d", name, time.Now().UnixNano()),
			TargetProcessorName: component.NewIDWithName(component.MustNewType(mapping.TargetProcessor), ""),
			ParameterPath:       mapping.ParameterPath,
			NewValue:            scaledOutput,
			Reason:              fmt.Sprintf("PID adjustment from %s controller", name),
			Severity:            "normal",
			Source:              "adaptive_pid",
			Timestamp:           time.Now().Unix(),
			TTLSeconds:          300,
		}

		patches = append(patches, patch)
	}

	return patches
}

// ControllerMapping defines how a controller maps to processors and parameters
type ControllerMapping struct {
	KPIMetricName   string
	TargetProcessor string
	ParameterPath   string
	ScaleFactor     float64
	MinValue        float64
	MaxValue        float64
}

// SimulateSystem simulates a system response to a control action
// This is useful for integration testing the control loop in isolation
type SimulationResponse struct {
	// Maps from controller name to KPI value
	KPIResponse map[string]float64
}

// SimpleExponentialResponse returns a simple simulation that exponentially
// approaches the target based on the controller's output
func SimpleExponentialResponse(controllerName string, currentValue, targetValue, rate float64) SimulationResponse {
	// Calculate new value that moves toward target at the given rate
	newValue := currentValue + (targetValue-currentValue)*rate

	return SimulationResponse{
		KPIResponse: map[string]float64{
			controllerName: newValue,
		},
	}
}
