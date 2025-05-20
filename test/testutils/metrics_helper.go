package testutils

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// GenerateMetrics creates a standardized set of metrics for testing
func GenerateMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()

	// Create resource metrics for each test process
	processes := []struct {
		Name  string
		Attrs map[string]interface{}
	}{
		{
			Name: "nginx-worker",
			Attrs: map[string]interface{}{
				"service.name":           "web-frontend",
				"deployment.environment": "production",
			},
		},
		{
			Name: "mysql-server",
			Attrs: map[string]interface{}{
				"service.name":           "database",
				"deployment.environment": "production",
			},
		},
		{
			Name: "background-worker",
			Attrs: map[string]interface{}{
				"service.name":           "batch-processor",
				"deployment.environment": "production",
			},
		},
		{
			Name: "other-process",
			Attrs: map[string]interface{}{
				"service.name":           "misc",
				"deployment.environment": "production",
			},
		},
	}

	for _, process := range processes {
		rm := metrics.ResourceMetrics().AppendEmpty()

		// Set resource attributes
		attrs := rm.Resource().Attributes()
		attrs.PutStr("process.name", process.Name)

		for k, v := range process.Attrs {
			switch val := v.(type) {
			case string:
				attrs.PutStr(k, val)
			case int:
				attrs.PutInt(k, int64(val))
			case float64:
				attrs.PutDouble(k, val)
			case bool:
				attrs.PutBool(k, val)
			}
		}

		// Add metric data
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test.scope")
		sm.Scope().SetVersion("1.0")

		// Add CPU metric
		metricCPU := sm.Metrics().AppendEmpty()
		metricCPU.SetName("process.cpu.utilization")
		metricCPU.SetEmptyGauge()
		dpCPU := metricCPU.Gauge().DataPoints().AppendEmpty()
		dpCPU.SetDoubleValue(0.45) // 45% CPU usage
		dpCPU.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

		// Add memory metric
		metricMem := sm.Metrics().AppendEmpty()
		metricMem.SetName("process.memory.usage")
		metricMem.SetEmptyGauge()
		dpMem := metricMem.Gauge().DataPoints().AppendEmpty()
		dpMem.SetIntValue(1024 * 1024 * 100) // 100 MB
		dpMem.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

		// Add request count metric (only for nginx)
		if process.Name == "nginx-worker" {
			metricReq := sm.Metrics().AppendEmpty()
			metricReq.SetName("http.server.request.count")
			metricReq.SetEmptySum()
			metricReq.Sum().SetIsMonotonic(true)
			metricReq.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
			dpReq := metricReq.Sum().DataPoints().AppendEmpty()
			dpReq.SetIntValue(1000)
			dpReq.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		}

		// Add database operations (only for mysql)
		if process.Name == "mysql-server" {
			metricDB := sm.Metrics().AppendEmpty()
			metricDB.SetName("db.operation.count")
			metricDB.SetEmptySum()
			metricDB.Sum().SetIsMonotonic(true)
			metricDB.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
			dpDB := metricDB.Sum().DataPoints().AppendEmpty()
			dpDB.SetIntValue(500)
			dpDB.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			dpDB.Attributes().PutStr("db.operation", "select")

			dpDB2 := metricDB.Sum().DataPoints().AppendEmpty()
			dpDB2.SetIntValue(100)
			dpDB2.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			dpDB2.Attributes().PutStr("db.operation", "insert")
		}
	}

	return metrics
}

// GenerateMetricsWithHighCardinality creates metrics with high cardinality for testing
func GenerateMetricsWithHighCardinality(cardinalityCount int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()

	// Create a single resource metric with high cardinality datapoints
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "high-cardinality-service")

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("test.scope")

	// Create a metric with high cardinality
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("test.high.cardinality")
	metric.SetEmptyGauge()

	// Add many data points with different attributes
	for i := 0; i < cardinalityCount; i++ {
		dp := metric.Gauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(float64(i))
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

		// Add unique attributes to each data point
		dp.Attributes().PutStr("id", fmt.Sprintf("id-%d", i))
		dp.Attributes().PutInt("value", int64(i))

		// Add some common dimensions for realistic scenarios
		dp.Attributes().PutStr("region", fmt.Sprintf("region-%d", i%5))
		dp.Attributes().PutStr("zone", fmt.Sprintf("zone-%d", i%10))
		dp.Attributes().PutStr("cluster", fmt.Sprintf("cluster-%d", i%3))
	}

	return metrics
}

// GenerateControlLoopMetrics creates metrics that would be used in the control loop
func GenerateControlLoopMetrics(kpiValues map[string]float64) pmetric.Metrics {
	metrics := pmetric.NewMetrics()

	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "sa-omf-collector")

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("sa-omf")

	// Add KPI metrics
	for kpiName, kpiValue := range kpiValues {
		metric := sm.Metrics().AppendEmpty()
		metric.SetName(kpiName)
		metric.SetEmptyGauge()

		dp := metric.Gauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(kpiValue)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}

	return metrics
}

// TestHost implements component.Host for testing
type TestHost struct {
	processors map[component.ID]component.Component
	extensions map[component.ID]component.Component
}

// NewTestHost creates a new test host for testing
func NewTestHost() *TestHost {
	return &TestHost{
		processors: make(map[component.ID]component.Component),
		extensions: make(map[component.ID]component.Component),
	}
}

// ReportFatalError implements component.Host
func (h *TestHost) ReportFatalError(err error) {
	// Do nothing in tests
}

// GetFactory implements component.Host
func (h *TestHost) GetFactory(kind component.Kind, componentType component.Type) component.Factory {
	return nil
}

// GetExtensions implements component.Host
func (h *TestHost) GetExtensions() map[component.ID]component.Component {
	return h.extensions
}

// GetExporters implements component.Host
func (h *TestHost) GetExporters() map[component.Type]map[component.ID]component.Component {
	return nil
}

// GetProcessors implements host.Host returning processors as generic components
func (h *TestHost) GetProcessors() map[component.ID]component.Component {
	return h.processors
}

// AddProcessor adds a processor to the host
func (h *TestHost) AddProcessor(id component.ID, processor component.Component) {
	h.processors[id] = processor
}

// AddExtension adds an extension to the host
func (h *TestHost) AddExtension(id component.ID, ext extension.Extension) {
	h.extensions[id] = ext
}
