package prioritytagger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.uber.org/zap"

	"github.com/yourorg/sa-omf/internal/interfaces"
)

func TestPriorityTagger(t *testing.T) {
	// Create test metrics
	metrics := generateTestMetrics()
	
	// Create test config
	cfg := &Config{
		Rules: []Rule{
			{
				Match:    "^node$",
				Priority: "critical",
			},
			{
				Match:    "^java$",
				Priority: "high",
			},
		},
		Enabled: true,
	}
	
	// Create test processor
	next := new(consumertest.MetricsSink)
	factory := NewFactory()
	set := processortest.NewNopCreateSettings()
	processor, err := factory.CreateMetricsProcessor(context.Background(), set, cfg, next)
	require.NoError(t, err)
	require.NotNil(t, processor)
	
	// Start processor
	err = processor.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	
	// Process metrics
	err = processor.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)
	
	// Verify results
	processedMetrics := next.AllMetrics()
	require.Len(t, processedMetrics, 1)
	
	// Check that the priority attributes have been added
	metricData := processedMetrics[0]
	resourceMetrics := metricData.ResourceMetrics()
	
	foundNode := false
	foundJava := false
	
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		resource := rm.Resource()
		
		// Check if this resource has a process name
		var processName string
		if val, ok := resource.Attributes().Get("process.name"); ok {
			processName = val.AsString()
		}
		
		// Check if a priority was assigned
		if val, ok := resource.Attributes().Get("aemf.process.priority"); ok {
			if processName == "node" {
				assert.Equal(t, "critical", val.AsString())
				foundNode = true
			} else if processName == "java" {
				assert.Equal(t, "high", val.AsString())
				foundJava = true
			}
		}
	}
	
	assert.True(t, foundNode, "node process with critical priority not found")
	assert.True(t, foundJava, "java process with high priority not found")
	
	// Test UpdateableProcessor implementation
	updateableProc, ok := processor.(interfaces.UpdateableProcessor)
	assert.True(t, ok, "processor does not implement UpdateableProcessor")
	
	// Test GetConfigStatus
	status, err := updateableProc.GetConfigStatus(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, len(status.Parameters["rules"].([]Rule)))
	assert.True(t, status.Enabled)
	
	// Test OnConfigPatch - add a new rule
	patch := interfaces.ConfigPatch{
		PatchID:             "test-patch-001",
		TargetProcessorName: component.MustNewIDFromString("priority_tagger"),
		ParameterPath:       "rules",
		NewValue: []Rule{
			{
				Match:    "^node$",
				Priority: "critical",
			},
			{
				Match:    "^java$",
				Priority: "high",
			},
			{
				Match:    "^python$",
				Priority: "medium",
			},
		},
		Reason:   "test patch",
		Severity: "normal",
		Source:   "test",
	}
	
	err = updateableProc.OnConfigPatch(context.Background(), patch)
	require.NoError(t, err)
	
	// Verify the patch was applied
	status, err = updateableProc.GetConfigStatus(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, len(status.Parameters["rules"].([]Rule)))
	
	// Test OnConfigPatch - disable processor
	patch = interfaces.ConfigPatch{
		PatchID:             "test-patch-002",
		TargetProcessorName: component.MustNewIDFromString("priority_tagger"),
		ParameterPath:       "enabled",
		NewValue:            false,
		Reason:              "test patch",
		Severity:            "normal",
		Source:              "test",
	}
	
	err = updateableProc.OnConfigPatch(context.Background(), patch)
	require.NoError(t, err)
	
	// Verify the patch was applied
	status, err = updateableProc.GetConfigStatus(context.Background())
	require.NoError(t, err)
	assert.False(t, status.Enabled)
	
	// Shutdown processor
	err = processor.Shutdown(context.Background())
	require.NoError(t, err)
}

func generateTestMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	
	// Add resource metrics for "node" process
	rm1 := md.ResourceMetrics().AppendEmpty()
	rm1.Resource().Attributes().PutStr("process.name", "node")
	rm1.Resource().Attributes().PutStr("process.pid", "1234")
	sm1 := rm1.ScopeMetrics().AppendEmpty()
	sm1.Scope().SetName("host")
	metric1 := sm1.Metrics().AppendEmpty()
	metric1.SetName("cpu.usage")
	metric1.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(0.8)
	
	// Add resource metrics for "java" process
	rm2 := md.ResourceMetrics().AppendEmpty()
	rm2.Resource().Attributes().PutStr("process.name", "java")
	rm2.Resource().Attributes().PutStr("process.pid", "5678")
	sm2 := rm2.ScopeMetrics().AppendEmpty()
	sm2.Scope().SetName("host")
	metric2 := sm2.Metrics().AppendEmpty()
	metric2.SetName("memory.usage")
	metric2.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(1024.0)
	
	// Add resource metrics for "nginx" process (no rule match)
	rm3 := md.ResourceMetrics().AppendEmpty()
	rm3.Resource().Attributes().PutStr("process.name", "nginx")
	rm3.Resource().Attributes().PutStr("process.pid", "9012")
	sm3 := rm3.ScopeMetrics().AppendEmpty()
	sm3.Scope().SetName("host")
	metric3 := sm3.Metrics().AppendEmpty()
	metric3.SetName("network.connections")
	metric3.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(100.0)
	
	return md
}
