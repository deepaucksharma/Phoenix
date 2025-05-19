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

	"github.com/yourorg/sa-omf/internal/interfaces"
	"github.com/yourorg/sa-omf/internal/processor/priority_tagger"
	iftest "github.com/yourorg/sa-omf/test/interfaces"
)

func TestPriorityTaggerProcessor(t *testing.T) {
	// Create a factory
	factory := prioritytagger.NewFactory()
	assert.NotNil(t, factory)

	// Create a default configuration
	cfg := factory.CreateDefaultConfig().(*prioritytagger.Config)
	
	// Add test rules
	cfg.Rules = []prioritytagger.Rule{
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
	settings := processor.CreateSettings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewID("priority_tagger"),
	}

	proc, err := factory.CreateMetricsProcessor(ctx, settings, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Ensure it implements the UpdateableProcessor interface
	updateableProc, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok, "Processor does not implement UpdateableProcessor")

	// Start the processor
	err = proc.Start(ctx, nil)
	require.NoError(t, err)

	// Run the standard UpdateableProcessor tests
	suite := iftest.UpdateableProcessorSuite{
		Processor: updateableProc,
		ValidPatches: []iftest.TestPatch{
			{
				Name: "ChangeEnabled",
				Patch: interfaces.ConfigPatch{
					PatchID:             "test-enable",
					TargetProcessorName: component.NewID("priority_tagger"),
					ParameterPath:       "enabled",
					NewValue:            false,
				},
				ExpectedValue: false,
			},
			{
				Name: "UpdateRules",
				Patch: interfaces.ConfigPatch{
					PatchID:             "test-rules",
					TargetProcessorName: component.NewID("priority_tagger"),
					ParameterPath:       "rules",
					NewValue: []prioritytagger.Rule{
						{
							Match:    "apache.*",
							Priority: "high",
						},
						{
							Match:    "postgres.*",
							Priority: "critical",
						},
					},
				},
				// We don't check ExpectedValue here as the rules are stored as a complex structure
			},
		},
		InvalidPatches: []iftest.TestPatch{
			{
				Name: "InvalidRegex",
				Patch: interfaces.ConfigPatch{
					PatchID:             "test-invalid-regex",
					TargetProcessorName: component.NewID("priority_tagger"),
					ParameterPath:       "rules",
					NewValue: []prioritytagger.Rule{
						{
							Match:    "[invalid regex",
							Priority: "high",
						},
					},
				},
			},
		},
	}
	iftest.RunUpdateableProcessorTests(t, suite)

	// Test actual metric processing functionality
	t.Run("ProcessMetrics", func(t *testing.T) {
		// Create test metrics
		metrics := generateTestMetrics()
		
		// Re-enable the processor for testing
		enablePatch := interfaces.ConfigPatch{
			PatchID:             "test-enable-for-processing",
			TargetProcessorName: component.NewID("priority_tagger"),
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