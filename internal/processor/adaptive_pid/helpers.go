package adaptive_pid

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// extractKPIValues extracts KPI values from metrics
func extractKPIValues(md pmetric.Metrics) map[string]float64 {
	result := make(map[string]float64)

	// Iterate through all metrics
	resourceMetricsSlice := md.ResourceMetrics()
	for i := 0; i < resourceMetricsSlice.Len(); i++ {
		resourceMetrics := resourceMetricsSlice.At(i)

		scopeMetricsSlice := resourceMetrics.ScopeMetrics()
		for j := 0; j < scopeMetricsSlice.Len(); j++ {
			scopeMetrics := scopeMetricsSlice.At(j)

			metricsSlice := scopeMetrics.Metrics()
			for k := 0; k < metricsSlice.Len(); k++ {
				metric := metricsSlice.At(k)

				// We're just looking for gauges and sums for KPIs
				var value float64
				var found bool

				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					gauge := metric.Gauge()
					if gauge.DataPoints().Len() > 0 {
						point := gauge.DataPoints().At(0)
						value = point.DoubleValue()
						found = true
					}
				case pmetric.MetricTypeSum:
					sum := metric.Sum()
					if sum.DataPoints().Len() > 0 {
						point := sum.DataPoints().At(0)
						value = point.DoubleValue()
						found = true
					}
				}

				if found {
					result[metric.Name()] = value
				}
			}
		}
	}

	return result
}

// createMetric creates a simple metric for testing purposes
func createMetric(name string, value float64) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()

	m.SetName(name)
	m.SetEmptyGauge()
	dp := m.Gauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetDoubleValue(value)

	return metrics
}
