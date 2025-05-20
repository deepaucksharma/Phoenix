// Package metrics provides utilities for metrics emission and collection.
package metrics

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
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
		LastEmission:   time.Time{}, // Zero time
		EmitInterval:   time.Second * 10, // Default interval
		Parent:         parent,
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

// ShouldEmit checks if metrics should be emitted based on the interval.
func (p *PIDMetrics) ShouldEmit() bool {
	return time.Since(p.LastEmission) >= p.EmitInterval
}

// EmitMetrics creates PID controller metrics and adds them to the parent emitter if available.
// If no parent emitter is available, this simply returns the metrics without emitting them.
func (p *PIDMetrics) EmitMetrics(ctx context.Context) pmetric.Metrics {
	// Create metrics
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	
	// Add resource attributes
	resourceMetrics.Resource().Attributes().PutStr("controller.name", p.ControllerName)
	resourceMetrics.Resource().Attributes().PutStr("controller.type", "pid")
	
	// Create scope metrics
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
	scopeMetrics.Scope().SetName("aemf.controller.pid")
	
	// Create metrics
	createPIDMetrics(scopeMetrics.Metrics(), p)
	
	// Update last emission time
	p.LastEmission = time.Now()
	
	// If parent emitter exists, add these metrics to its queue
	if p.Parent != nil {
		p.Parent.AddMetrics(metrics)
	}
	
	return metrics
}

// createPIDMetrics adds PID controller metrics to the metrics collection.
func createPIDMetrics(metrics pmetric.MetricSlice, p *PIDMetrics) {
	now := pcommon.NewTimestampFromTime(time.Now())
	
	// Error metric
	errorMetric := metrics.AppendEmpty()
	errorMetric.SetName("aemf.controller.pid.error")
	errorMetric.SetEmptyGauge()
	dp := errorMetric.Gauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(now)
	dp.SetDoubleValue(p.Error)
	
	// P term metric
	pTermMetric := metrics.AppendEmpty()
	pTermMetric.SetName("aemf.controller.pid.p_term")
	pTermMetric.SetEmptyGauge()
	dp = pTermMetric.Gauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(now)
	dp.SetDoubleValue(p.PValue)
	
	// I term metric
	iTermMetric := metrics.AppendEmpty()
	iTermMetric.SetName("aemf.controller.pid.i_term")
	iTermMetric.SetEmptyGauge()
	dp = iTermMetric.Gauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(now)
	dp.SetDoubleValue(p.IValue)
	
	// D term metric
	dTermMetric := metrics.AppendEmpty()
	dTermMetric.SetName("aemf.controller.pid.d_term")
	dTermMetric.SetEmptyGauge()
	dp = dTermMetric.Gauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(now)
	dp.SetDoubleValue(p.DValue)
	
	// Raw output metric
	rawOutputMetric := metrics.AppendEmpty()
	rawOutputMetric.SetName("aemf.controller.pid.raw_output")
	rawOutputMetric.SetEmptyGauge()
	dp = rawOutputMetric.Gauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(now)
	dp.SetDoubleValue(p.RawOutput)
	
	// Final output metric
	outputMetric := metrics.AppendEmpty()
	outputMetric.SetName("aemf.controller.pid.output")
	outputMetric.SetEmptyGauge()
	dp = outputMetric.Gauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(now)
	dp.SetDoubleValue(p.Output)
	
	// Setpoint metric
	setpointMetric := metrics.AppendEmpty()
	setpointMetric.SetName("aemf.controller.pid.setpoint")
	setpointMetric.SetEmptyGauge()
	dp = setpointMetric.Gauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(now)
	dp.SetDoubleValue(p.Setpoint)
	
	// Measurement metric
	measurementMetric := metrics.AppendEmpty()
	measurementMetric.SetName("aemf.controller.pid.measurement")
	measurementMetric.SetEmptyGauge()
	dp = measurementMetric.Gauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(now)
	dp.SetDoubleValue(p.Measurement)
}