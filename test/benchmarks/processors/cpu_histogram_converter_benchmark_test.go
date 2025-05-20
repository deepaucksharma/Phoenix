package benchmarks

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/processor/cpu_histogram_converter"
)

func BenchmarkCPUHistogramConverter(b *testing.B) {
	// Setup
	benchmarks := []struct {
		name         string
		processCount int
	}{
		{
			name:         "Small_10Processes",
			processCount: 10,
		},
		{
			name:         "Medium_100Processes",
			processCount: 100,
		},
		{
			name:         "Large_1000Processes",
			processCount: 1000,
		},
	}

	// Run benchmarks
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Create processor
			factory := cpu_histogram_converter.NewFactory()
			cfg := factory.CreateDefaultConfig().(*cpu_histogram_converter.Config)
			cfg.Enabled = true
			cfg.InputMetricName = "process.cpu.time"
			cfg.OutputMetricName = "process.cpu.utilization.histogram"
			cfg.CollectionIntervalSeconds = 10
			cfg.HostCPUCount = 8
			cfg.MaxProcessesInMemory = 10000
			
			sink := new(consumertest.MetricsSink)
			
			ctx := context.Background()
			settings := processor.Settings{
				TelemetrySettings: component.TelemetrySettings{
					Logger: zap.NewNop(),
				},
				ID: component.NewIDWithName(component.MustNewType("cpu_histogram_converter"), ""),
			}
			
			proc, _ := factory.CreateMetricsProcessor(ctx, settings, cfg, sink)
			proc.Start(ctx, nil)
			
			// Create initial metrics batch to establish baseline values
			initialMetrics := createCPUBenchmarkMetrics(bm.processCount, 10.0)
			_ = proc.ConsumeMetrics(ctx, initialMetrics)
			
			// Wait a moment to simulate time passing
			time.Sleep(10 * time.Millisecond)
			
			// Create second metrics batch with increased values
			incrementedMetrics := createCPUBenchmarkMetrics(bm.processCount, 15.0)
			
			// Reset benchmark timer to exclude setup time
			b.ResetTimer()
			
			// Run benchmark
			for i := 0; i < b.N; i++ {
				_ = proc.ConsumeMetrics(ctx, incrementedMetrics)
			}
			
			// Stop timer and clean up
			b.StopTimer()
			proc.Shutdown(ctx)
		})
	}
}

// createCPUBenchmarkMetrics creates test metrics with CPU time metrics for multiple processes
func createCPUBenchmarkMetrics(processCount int, baseCPUTime float64) pmetric.Metrics {
	md := pmetric.NewMetrics()
	
	for i := 0; i < processCount; i++ {
		rm := md.ResourceMetrics().AppendEmpty()
		
		// Add process attributes
		rm.Resource().Attributes().PutStr("process.executable.name", "bench-process-"+string(rune(i%20)))
		rm.Resource().Attributes().PutStr("process.pid", string(rune(1000+i)))
		rm.Resource().Attributes().PutStr("host.name", "bench-host-"+string(rune(i%10)))
		
		// Add aemf.filter.included for half the processes to test top-k filtering
		if i%2 == 0 {
			rm.Resource().Attributes().PutStr("aemf.filter.included", "true")
		}
		
		// Add scope metrics
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("bench-scope")
		
		// Add CPU time metric
		metric := sm.Metrics().AppendEmpty()
		metric.SetName("process.cpu.time")
		
		sum := metric.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		
		dp := sum.DataPoints().AppendEmpty()
		dp.SetDoubleValue(baseCPUTime + float64(i)*0.1)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		dp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-time.Minute)))
		
		// Add some attributes to the datapoint
		dp.Attributes().PutStr("cpu", "all")
		dp.Attributes().PutStr("state", "user")
	}
	
	return md
}