// Package metrics provides standard methods for components to emit metrics about themselves.
package metrics

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

// MetricsEmitter provides a standardized way to emit self-metrics
type MetricsEmitter struct {
	meter       metric.Meter
	component   string
	commonAttrs []attribute.KeyValue
}

// NewMetricsEmitter creates a new MetricsEmitter for a component
func NewMetricsEmitter(meter metric.Meter, componentType string, componentName component.ID) *MetricsEmitter {
	return &MetricsEmitter{
		meter:     meter,
		component: componentType,
		commonAttrs: []attribute.KeyValue{
			attribute.String("component.type", componentType),
			attribute.String("component.name", componentName.String()),
		},
	}
}

// RegisterCounter creates and returns a new counter metric
func (e *MetricsEmitter) RegisterCounter(name string, description string) (metric.Int64Counter, error) {
	return e.meter.Int64Counter(
		"aemf_"+e.component+"_"+name,
		metric.WithDescription(description),
	)
}

// RegisterGauge creates and returns a new gauge metric
func (e *MetricsEmitter) RegisterGauge(name string, description string) (metric.Float64Gauge, error) {
	return e.meter.Float64Gauge(
		"aemf_"+e.component+"_"+name,
		metric.WithDescription(description),
	)
}

// CreatePatchMetric creates an OTLP metric for a ConfigPatch
func CreatePatchMetric(patch *interfaces.ConfigPatch) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("phoenix.pid_decider")

	m := sm.Metrics().AppendEmpty()
	m.SetName("aemf_ctrl_proposed_patch")
	m.SetDescription("Proposed configuration patch")

	dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(1.0) // Always 1.0 to indicate a patch
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(patch.Timestamp, 0)))

	// Add attributes from the patch
	dp.Attributes().PutStr("patch_id", patch.PatchID)
	dp.Attributes().PutStr("target_processor_name", patch.TargetProcessorName.String())
	dp.Attributes().PutStr("parameter_path", patch.ParameterPath)
	dp.Attributes().PutStr("reason", patch.Reason)
	dp.Attributes().PutStr("severity", patch.Severity)
	dp.Attributes().PutStr("source", patch.Source)

	// Add the new value as string, regardless of its actual type
	switch v := patch.NewValue.(type) {
	case int:
		dp.Attributes().PutInt("new_value_int", int64(v))
	case float64:
		dp.Attributes().PutDouble("new_value_double", v)
	case string:
		dp.Attributes().PutStr("new_value_string", v)
	case bool:
		dp.Attributes().PutBool("new_value_bool", v)
	default:
		// Try to convert to string as fallback
		dp.Attributes().PutStr("new_value_string", fmt.Sprintf("%v", v))
	}

	return metrics
}
