// Package metrics provides utilities for metrics emission and collection.
package metrics

import (
	"context"
	"sync"
	"time"
)

// PIDMetrics holds metrics related to PID controller operation.
type PIDMetrics struct {
	// The name of the controller for identification in metrics
	ControllerName string

	// Raw metric values
	Error       float64
	PValue      float64
	IValue      float64
	DValue      float64
	Output      float64
	RawOutput   float64
	Setpoint    float64
	Measurement float64

	// Custom metrics
	customMetrics map[string]float64
	lock          sync.RWMutex

	// State tracking
	LastEmission time.Time
	EmitInterval time.Duration

	// Optional parent metrics emitter
	Parent *MetricsEmitter
}

// NewPIDMetrics creates a new PIDMetrics instance with the given name.
func NewPIDMetrics(controllerName string, parent *MetricsEmitter) *PIDMetrics {
	return &PIDMetrics{
		ControllerName: controllerName,
		LastEmission:   time.Time{},      // Zero time
		EmitInterval:   time.Second * 10, // Default interval
		Parent:         parent,
		customMetrics:  make(map[string]float64),
	}
}

// SetEmitInterval sets the minimum interval between metrics emissions.
func (p *PIDMetrics) SetEmitInterval(interval time.Duration) {
	p.EmitInterval = interval
}

// Update updates PID metrics with the latest values.
func (p *PIDMetrics) Update(setpoint, measurement, error, pValue, iValue, dValue, rawOutput, output float64) {
	p.Setpoint = setpoint
	p.Measurement = measurement
	p.Error = error
	p.PValue = pValue
	p.IValue = iValue
	p.DValue = dValue
	p.RawOutput = rawOutput
	p.Output = output
}

// AddMetric adds or updates a custom metric with the given name and value.
func (p *PIDMetrics) AddMetric(name string, value float64) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.customMetrics[name] = value
}

// ShouldEmit checks if metrics should be emitted based on the interval.
func (p *PIDMetrics) ShouldEmit() bool {
	return time.Since(p.LastEmission) >= p.EmitInterval
}

// EmitMetrics creates PID controller metrics and adds them to the parent emitter if available.
// If no parent emitter is available, this simply returns without emitting metrics.
func (p *PIDMetrics) EmitMetrics(ctx context.Context) {
	// Update last emission time
	p.LastEmission = time.Now()

	// In the future, we can add more sophisticated metrics emission here
	// For now, this is just a placeholder to maintain the API
}
