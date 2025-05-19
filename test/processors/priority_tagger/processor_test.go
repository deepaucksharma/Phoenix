package priority_tagger

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/priority_tagger"
)

func TestPriorityTaggerProcessor(t *testing.T) {
	// Create a factory
	factory := priority_tagger.NewFactory()
	assert.NotNil(t, factory)

	// Create a default configuration
	cfg := factory.CreateDefaultConfig().(*priority_tagger.Config)
	
	// Add test rules
	cfg.Rules = []priority_tagger.Rule{
		{
			Match:    "nginx.*",
			Priority: "high",
		},
		{
			Match:    ".*mysql.*",
			Priority: "critical",
		},
		{
			Match:    "background.*",
			Priority: "low",
		},
	}
	cfg.Enabled = true

	// Create a test sink for output metrics
	sink := new(consumertest.MetricsSink)

	// Create the processor
	ctx := context.Background()
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewIDWithName(component.MustNewType("priority_tagger"), ""),
	}

	proc, err := factory.CreateMetrics(ctx, settings, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Ensure it implements the UpdateableProcessor interface
	updateableProc, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok, "Processor does not implement UpdateableProcessor")

	// Start the processor
	err = proc.Start(ctx, nil)
	require.NoError(t, err)

	// Test the interface methods directly
	// Test the OnConfigPatch method
	
	// Test enabled flag
	enablePatch := interfaces.ConfigPatch{
		PatchID:             "test-enable",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("priority_tagger"), ""),
		ParameterPath:       "enabled",
		NewValue:            false,
	}
	err = updateableProc.OnConfigPatch(ctx, enablePatch)
	require.NoError(t, err, "Failed to apply enable patch")
	
	status, err := updateableProc.GetConfigStatus(ctx)
	require.NoError(t, err, "Failed to get config status")
	assert.False(t, status.Enabled, "Processor should be disabled")
	
	// Test updating rules
	rulesPatch := interfaces.ConfigPatch{
		PatchID:             "test-rules",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("priority_tagger"), ""),
		ParameterPath:       "rules",
		NewValue: []priority_tagger.Rule{
			{
				Match:    "apache.*",
				Priority: "high",
			},
			{
				Match:    "postgres.*",
				Priority: "critical",
			},
		},
	}
	err = updateableProc.OnConfigPatch(ctx, rulesPatch)
	require.NoError(t, err, "Failed to apply rules patch")
	
	// Test invalid regex
	invalidPatch := interfaces.ConfigPatch{
		PatchID:             "test-invalid-regex",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("priority_tagger"), ""),
		ParameterPath:       "rules",
		NewValue: []priority_tagger.Rule{
			{
				Match:    "[invalid regex",
				Priority: "high",
			},
		},
	}
	err = updateableProc.OnConfigPatch(ctx, invalidPatch)
	assert.Error(t, err, "Should fail with invalid regex")

	// Test actual metric processing functionality
	t.Run("ProcessMetrics", func(t *testing.T) {
		// Create test metrics
		metrics := generateTestMetrics()
		
		// First, ensure we have the original test rules (they may have been changed by earlier tests)
		rulesPatch := interfaces.ConfigPatch{
			PatchID:             "test-restore-rules",
			TargetProcessorName: component.NewIDWithName(component.MustNewType("priority_tagger"), ""),
			ParameterPath:       "rules",
			NewValue: []priority_tagger.Rule{
				{
					Match:    "nginx.*",
					Priority: "high",
				},
				{
					Match:    ".*mysql.*",
					Priority: "critical",
				},
				{
					Match:    "background.*",
					Priority: "low",
				},
			},
		}
		err = updateableProc.OnConfigPatch(ctx, rulesPatch)
		require.NoError(t, err)
		
		// Re-enable the processor for testing
		enablePatch := interfaces.ConfigPatch{
			PatchID:             "test-enable-for-processing",
			TargetProcessorName: component.NewIDWithName(component.MustNewType("priority_tagger"), ""),
			ParameterPath:       "enabled",
			NewValue:            true,
		}
		err = updateableProc.OnConfigPatch(ctx, enablePatch)
		require.NoError(t, err)
		
		// Process metrics
		err = proc.ConsumeMetrics(ctx, metrics)
		require.NoError(t, err)
		
		// Verify output
		processedMetrics := sink.AllMetrics()
		require.NotEmpty(t, processedMetrics)
		
		// Check that priorities were assigned correctly
		for i := 0; i < processedMetrics[0].ResourceMetrics().Len(); i++ {
			rm := processedMetrics[0].ResourceMetrics().At(i)
			resource := rm.Resource()
			
			processName, ok := resource.Attributes().Get("process.name")
			require.True(t, ok, "process.name attribute missing")
			
			// Check if priority was correctly assigned based on process name
			priorityAttr, ok := resource.Attributes().Get("aemf.process.priority")
			
			// nginx process should have high priority
			if processName.Str() == "nginx-worker" {
				require.True(t, ok, "Priority attribute missing for nginx process")
				assert.Equal(t, "high", priorityAttr.Str())
			}
			
			// mysql process should have critical priority
			if processName.Str() == "mysql-server" {
				require.True(t, ok, "Priority attribute missing for mysql process")
				assert.Equal(t, "critical", priorityAttr.Str())
			}
			
			// background process should have low priority
			if processName.Str() == "background-worker" {
				require.True(t, ok, "Priority attribute missing for background process")
				assert.Equal(t, "low", priorityAttr.Str())
			}
			
			// other process should not have a priority assigned
			if processName.Str() == "other-process" {
				assert.False(t, ok, "Priority incorrectly assigned to non-matching process")
			}
		}
	})
	
	// Shutdown the processor
	err = proc.Shutdown(ctx)
	require.NoError(t, err)
}

// generateTestMetrics creates test metrics with different process names
func generateTestMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	// Create metrics with 4 resources, each with a different process name
	processNames := []string{"nginx-worker", "mysql-server", "background-worker", "other-process"}
	
	for _, name := range processNames {
		rm := metrics.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("process.name", name)
		
		// Add a metric
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test.scope")
		
		metric := sm.Metrics().AppendEmpty()
		metric.SetName("test.metric")
		metric.SetEmptyGauge()
		dp := metric.Gauge().DataPoints().AppendEmpty()
		dp.SetIntValue(100)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(testNow))
	}
	
	return metrics
}

// Test time value to use for consistency
var testNow = time.Now()