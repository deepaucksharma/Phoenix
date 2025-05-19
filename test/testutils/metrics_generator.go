// Package testutils provides utilities for testing SA-OMF components.
package testutils

import (
	"math/rand"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var processNames = []string{
	"java", "python", "node", "nginx", "httpd", "mysql", "postgres", 
	"redis", "mongodb", "prometheus", "grafana", "fluentd", "kibana",
	"kafka", "zookeeper", "etcd", "consul", "nomad", "vault",
}

// GenerateTestMetrics creates a set of test metrics suitable for most processor tests.
func GenerateTestMetrics(processCount int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	// Generate one ResourceMetrics per process
	for i := 0; i < processCount; i++ {
		rm := metrics.ResourceMetrics().AppendEmpty()
		
		// Set resource attributes
		procName := processNames[i%len(processNames)]
		procID := procName + "-" + strconv.Itoa(i)
		
		resource := rm.Resource()
		resource.Attributes().PutStr("process.name", procName)
		resource.Attributes().PutStr("process.id", procID)
		resource.Attributes().PutStr("host.name", "test-host-" + strconv.Itoa(i/10))
		
		// Generate metrics for each resource
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test-scope")
		
		// CPU metric
		cpuMetric := sm.Metrics().AppendEmpty()
		cpuMetric.SetName("process.cpu_seconds_total")
		cpuMetric.SetDescription("Total CPU time spent by the process")
		cpuMetric.SetUnit("s")
		
		cpuSum := cpuMetric.SetEmptySum()
		cpuSum.SetIsMonotonic(true)
		cpuSum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		
		cpuPoint := cpuSum.DataPoints().AppendEmpty()
		cpuPoint.SetTimestamp(pmetric.NewTimestampFromTime(time.Now()))
		cpuPoint.SetDoubleValue(float64(i) * 0.5)
		
		// Memory metric
		memMetric := sm.Metrics().AppendEmpty()
		memMetric.SetName("process.memory_rss")
		memMetric.SetDescription("Resident memory size used by the process")
		memMetric.SetUnit("bytes")
		
		memGauge := memMetric.SetEmptyGauge()
		memPoint := memGauge.DataPoints().AppendEmpty()
		memPoint.SetTimestamp(pmetric.NewTimestampFromTime(time.Now()))
		memPoint.SetDoubleValue(float64(1024*1024*(rand.Intn(100)+10))) // Random MB value
		
		// Counter metric
		countMetric := sm.Metrics().AppendEmpty()
		countMetric.SetName("process.requests_total")
		countMetric.SetDescription("Total number of requests processed")
		
		countSum := countMetric.SetEmptySum()
		countSum.SetIsMonotonic(true)
		countSum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		
		countPoint := countSum.DataPoints().AppendEmpty()
		countPoint.SetTimestamp(pmetric.NewTimestampFromTime(time.Now()))
		countPoint.SetDoubleValue(float64(rand.Intn(10000)))
	}
	
	return metrics
}

// GenerateHighCardinalityMetrics creates a set of test metrics with high cardinality.
func GenerateHighCardinalityMetrics(cardinalityCount int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	// Generate one ResourceMetrics per unique combination
	for i := 0; i < cardinalityCount; i++ {
		rm := metrics.ResourceMetrics().AppendEmpty()
		
		// Set highly unique resource attributes
		resource := rm.Resource()
		resource.Attributes().PutStr("process.name", "highcard-" + strconv.Itoa(i%50))
		resource.Attributes().PutStr("process.id", "proc-" + strconv.Itoa(i))
		resource.Attributes().PutStr("host.name", "host-" + strconv.Itoa(i%20))
		resource.Attributes().PutStr("service.name", "svc-" + strconv.Itoa(i%30))
		resource.Attributes().PutStr("deployment.environment", "env-" + strconv.Itoa(i%5))
		resource.Attributes().PutStr("container.id", "container-" + strconv.Itoa(i))
		
		// Add some high-cardinality attributes
		if i%2 == 0 {
			resource.Attributes().PutStr("user.id", "user-" + strconv.Itoa(i))
		}
		
		if i%3 == 0 {
			resource.Attributes().PutStr("transaction.id", "tx-" + strconv.Itoa(i))
		}
		
		if i%5 == 0 {
			resource.Attributes().PutStr("request.id", "req-" + strconv.Itoa(i))
		}
		
		// Generate a standard metric
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test-scope")
		
		// Standard metric
		metric := sm.Metrics().AppendEmpty()
		metric.SetName("test.metric")
		
		gauge := metric.SetEmptyGauge()
		point := gauge.DataPoints().AppendEmpty()
		point.SetTimestamp(pmetric.NewTimestampFromTime(time.Now()))
		point.SetDoubleValue(float64(rand.Intn(100)))
		
		// Add high-cardinality dimensions to datapoint
		point.Attributes().PutStr("dimension.id", "dim-" + strconv.Itoa(i))
		point.Attributes().PutStr("operation", "op-" + strconv.Itoa(i%10))
	}
	
	return metrics
}

// GenerateControlMetrics creates a set of metrics suitable for testing the control loop.
func GenerateControlMetrics(coverageScore float64) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	rm := metrics.ResourceMetrics().AppendEmpty()
	resource := rm.Resource()
	resource.Attributes().PutStr("service.name", "sa-omf-collector")
	
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("phoenix.impact")
	
	// Coverage score metric
	coverageMetric := sm.Metrics().AppendEmpty()
	coverageMetric.SetName("aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m")
	coverageMetric.SetDescription("Coverage score for adaptive_topk processor")
	
	coverageGauge := coverageMetric.SetEmptyGauge()
	coveragePoint := coverageGauge.DataPoints().AppendEmpty()
	coveragePoint.SetTimestamp(pmetric.NewTimestampFromTime(time.Now()))
	coveragePoint.SetDoubleValue(coverageScore)
	
	// Add some standard controller metrics
	cardinalityMetric := sm.Metrics().AppendEmpty()
	cardinalityMetric.SetName("aemf_impact_cardinality_reduction_ratio")
	cardinalityMetric.SetDescription("Cardinality reduction ratio")
	
	cardinalityGauge := cardinalityMetric.SetEmptyGauge()
	cardinalityPoint := cardinalityGauge.DataPoints().AppendEmpty()
	cardinalityPoint.SetTimestamp(pmetric.NewTimestampFromTime(time.Now()))
	cardinalityPoint.SetDoubleValue(0.85) // 85% cardinality reduction
	
	return metrics
}

// GeneratePatchMetric creates a metric that represents a configuration patch.
func GeneratePatchMetric(patch interfaces.ConfigPatch) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	rm := metrics.ResourceMetrics().AppendEmpty()
	resource := rm.Resource()
	resource.Attributes().PutStr("service.name", "sa-omf-collector")
	
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("phoenix.control")
	
	// Patch metric
	patchMetric := sm.Metrics().AppendEmpty()
	patchMetric.SetName("aemf_ctrl_proposed_patch")
	patchMetric.SetDescription("Proposed configuration patch")
	
	patchGauge := patchMetric.SetEmptyGauge()
	patchPoint := patchGauge.DataPoints().AppendEmpty()
	patchPoint.SetTimestamp(pmetric.NewTimestampFromTime(time.Now()))
	patchPoint.SetDoubleValue(1.0) // Always 1.0 to indicate a patch
	
	// Add attributes from the patch
	patchPoint.Attributes().PutStr("patch_id", patch.PatchID)
	patchPoint.Attributes().PutStr("target_processor_name", patch.TargetProcessorName.String())
	patchPoint.Attributes().PutStr("parameter_path", patch.ParameterPath)
	patchPoint.Attributes().PutStr("reason", patch.Reason)
	patchPoint.Attributes().PutStr("severity", patch.Severity)
	patchPoint.Attributes().PutStr("source", patch.Source)
	
	// Add the value as appropriate type
	switch v := patch.NewValue.(type) {
	case int:
		patchPoint.Attributes().PutInt("new_value_int", int64(v))
	case float64:
		patchPoint.Attributes().PutDouble("new_value_double", v)
	case string:
		patchPoint.Attributes().PutStr("new_value_string", v)
	case bool:
		patchPoint.Attributes().PutBool("new_value_bool", v)
	default:
		// Fall back to string
		patchPoint.Attributes().PutStr("new_value_string", fmt.Sprintf("%v", v))
	}
	
	return metrics
}