package otlp

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processortest"

	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
	"github.com/deepaucksharma/Phoenix/internal/processor/histogram_aggregator"
	"github.com/deepaucksharma/Phoenix/internal/processor/others_rollup"
	"github.com/deepaucksharma/Phoenix/internal/processor/priority_tagger"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

func TestNewRelicProcessPipeline(t *testing.T) {
	// Skip when not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}

	ctx := context.Background()
	
	// Create test sink to capture metrics
	sink := &testutils.MetricsSink{}
	
	// Create exportertest CreateSettings
	exporterSettings := exportertest.NewNopSettings()
	
	// Create processortest CreateSettings
	processorSettings := processortest.NewNopCreateSettings()
	
	// Create a pipeline of processors that match our New Relic configuration
	ptConfig := &priority_tagger.Config{
		Rules: []priority_tagger.Rule{
			{Match: "process.executable.name=~/java|nginx/", Priority: "high"},
			{Match: ".*", Priority: "low"},
		},
	}
	
	priorityTagger, err := priority_tagger.NewFactory().CreateMetricsProcessor(
		ctx, processorSettings, ptConfig, sink)
	require.NoError(t, err)
	
	topkConfig := &adaptive_topk.Config{
		KValue:        10,
		KMin:          5,
		KMax:          20,
		ResourceField: "process.executable.name",
		CounterField:  "process.cpu.utilization",
	}
	
	topkProcessor, err := adaptive_topk.NewFactory().CreateMetricsProcessor(
		ctx, processorSettings, topkConfig, priorityTagger)
	require.NoError(t, err)
	
	rollupConfig := &others_rollup.Config{
		Enabled:           true,
		PriorityThreshold: "low",
		Strategy:          "sum",
	}
	
	rollupProcessor, err := others_rollup.NewFactory().CreateMetricsProcessor(
		ctx, processorSettings, rollupConfig, topkProcessor)
	require.NoError(t, err)
	
	histConfig := &histogram_aggregator.Config{
		MaxBuckets: 10,
		CustomBoundaries: map[string][]float64{
			"process.memory.usage": {10000000, 50000000, 100000000, 500000000, 1000000000},
		},
	}
	
	histProcessor, err := histogram_aggregator.NewFactory().CreateMetricsProcessor(
		ctx, processorSettings, histConfig, rollupProcessor)
	require.NoError(t, err)
	
	// Start all processors
	err = histProcessor.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	
	t.Cleanup(func() {
		assert.NoError(t, histProcessor.Shutdown(ctx))
	})
	
	// Generate test metrics that resemble process metrics
	metrics := generateProcessMetrics()
	
	// Process metrics through the pipeline
	err = histProcessor.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)
	
	// Verify the output
	assert.NotEmpty(t, sink.AllMetrics())
	
	// Check specific aspects
	processedMetrics := sink.AllMetrics()[0]
	
	// Check cardinality
	checkCardinality(t, processedMetrics)
	
	// Verify histogram buckets have been optimized
	checkHistogramOptimization(t, processedMetrics)
	
	// Verify priority-based filtering
	checkPriorityFiltering(t, processedMetrics)
}

func generateProcessMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	
	// Create 30 process resources to simulate a realistic host
	for i := 0; i < 30; i++ {
		rm := md.ResourceMetrics().AppendEmpty()
		resource := rm.Resource()
		
		// Set process attributes based on index
		switch {
		case i < 5:
			// High priority resources
			resource.Attributes().PutStr("process.executable.name", "java")
			resource.Attributes().PutInt("process.pid", int64(1000+i))
			resource.Attributes().PutStr("process.owner", "app")
		case i < 10:
			// More high priority resources
			resource.Attributes().PutStr("process.executable.name", "nginx")
			resource.Attributes().PutInt("process.pid", int64(2000+i))
			resource.Attributes().PutStr("process.owner", "www-data")
		default:
			// Low priority resources
			resource.Attributes().PutStr("process.executable.name", "process-"+string(rune('a'+i%26)))
			resource.Attributes().PutInt("process.pid", int64(3000+i))
			resource.Attributes().PutStr("process.owner", "user")
		}
		
		// Add command line that should be filtered out
		resource.Attributes().PutStr("process.command_line", "sample command line argument --flag")
		
		// Create scope and metrics
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("hostmetrics")
		
		// Add CPU usage gauge
		cpuMetric := sm.Metrics().AppendEmpty()
		cpuMetric.SetName("process.cpu.utilization")
		cpuMetric.SetEmptyGauge()
		cpuDP := cpuMetric.Gauge().DataPoints().AppendEmpty()
		cpuDP.SetDoubleValue(float64(i) * 0.05) // 0.0 to 1.45
		cpuDP.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		
		// Add Memory usage gauge
		memMetric := sm.Metrics().AppendEmpty()
		memMetric.SetName("process.memory.usage")
		memMetric.SetEmptyHistogram()
		memDP := memMetric.Histogram().DataPoints().AppendEmpty()
		memDP.SetCount(100)
		memDP.SetSum(float64(i * 10000000))
		memDP.SetExplicitBounds([]float64{1000000, 5000000, 10000000, 50000000, 100000000, 
			200000000, 300000000, 400000000, 500000000, 1000000000, 2000000000, 5000000000})
		// Set some bucket counts
		memDP.BucketCounts().FromRaw([]uint64{10, 15, 20, 15, 10, 8, 7, 5, 4, 3, 2, 1})
		memDP.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}
	
	return md
}

func checkCardinality(t *testing.T, metrics pmetric.Metrics) {
	// Count unique resource + metric combinations (cardinality)
	uniqueMetrics := make(map[string]struct{})
	
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		resourceName := "unknown"
		
		if val, ok := rm.Resource().Attributes().Get("process.executable.name"); ok {
			resourceName = val.Str()
		}
		
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				key := resourceName + ":" + m.Name()
				uniqueMetrics[key] = struct{}{}
			}
		}
	}
	
	// We expect cardinality to be reduced significantly
	// from 30 processes * 2 metrics = 60 potential metrics
	// to around top 10-15 processes * 2 metrics plus rollup metrics
	assert.Less(t, len(uniqueMetrics), 40, "Cardinality should be reduced")
	t.Logf("Final metric cardinality: %d unique metrics", len(uniqueMetrics))
}

func checkHistogramOptimization(t *testing.T, metrics pmetric.Metrics) {
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				
				// Check only histograms
				if m.Type() == pmetric.MetricTypeHistogram {
					for dp := 0; dp < m.Histogram().DataPoints().Len(); dp++ {
						datapoint := m.Histogram().DataPoints().At(dp)
						
						// Check that bucket count is reduced
						bucketCount := datapoint.BucketCounts().Len()
						assert.LessOrEqual(t, bucketCount, 11, 
							"Histogram %s should have max 10 buckets + 1 overflow", m.Name())
						
						if m.Name() == "process.memory.usage" {
							// Check if custom boundaries were applied
							bounds := datapoint.ExplicitBounds().AsRaw()
							// We expect exactly the custom boundaries we configured
							expectedLen := 5 // This should match our config (length of custom boundaries)
							
							if len(bounds) > 0 {
								assert.LessOrEqual(t, len(bounds), expectedLen, 
									"Memory histogram should use custom boundaries")
								
								// Check for expected pattern in first two boundaries (if available)
								if len(bounds) >= 2 {
									// First boundary should be 10MB
									assert.InDelta(t, 10000000, bounds[0], 1, "First memory boundary should be ~10MB")
									
									// Second boundary should be 50MB
									assert.InDelta(t, 50000000, bounds[1], 1, "Second memory boundary should be ~50MB")
								}
							}
						}
					}
				}
			}
		}
	}
}

func checkPriorityFiltering(t *testing.T, metrics pmetric.Metrics) {
	hasJava := false
	hasNginx := false
	hasOthers := false
	
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		if val, ok := rm.Resource().Attributes().Get("process.executable.name"); ok {
			switch val.Str() {
			case "java":
				hasJava = true
			case "nginx":
				hasNginx = true
			case "others":
				hasOthers = true
			}
		}
	}
	
	// High priority processes should remain individual
	assert.True(t, hasJava, "Java process should be preserved")
	assert.True(t, hasNginx, "Nginx process should be preserved")
	
	// Low priority processes should be rolled up
	assert.True(t, hasOthers, "Others rollup should be present")
}