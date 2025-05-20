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

	"github.com/deepaucksharma/Phoenix/internal/processor/timeseries_estimator"
)

func BenchmarkTimeseriesEstimator(b *testing.B) {
	// Setup
	benchmarks := []struct {
		name         string
		metrics      pmetric.Metrics
		estimatorType string
	}{
		{
			name:         "ExactEstimator_SmallBatch",
			metrics:      generateBenchmarkMetrics(10, 5),
			estimatorType: "exact",
		},
		{
			name:         "ExactEstimator_MediumBatch",
			metrics:      generateBenchmarkMetrics(100, 10),
			estimatorType: "exact",
		},
		{
			name:         "ExactEstimator_LargeBatch",
			metrics:      generateBenchmarkMetrics(1000, 20),
			estimatorType: "exact",
		},
		{
			name:         "HLLEstimator_SmallBatch",
			metrics:      generateBenchmarkMetrics(10, 5),
			estimatorType: "hll",
		},
		{
			name:         "HLLEstimator_MediumBatch",
			metrics:      generateBenchmarkMetrics(100, 10),
			estimatorType: "hll",
		},
		{
			name:         "HLLEstimator_LargeBatch",
			metrics:      generateBenchmarkMetrics(1000, 20),
			estimatorType: "hll",
		},
	}

	// Run benchmarks
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Create processor
			factory := timeseries_estimator.NewFactory()
			cfg := factory.CreateDefaultConfig().(*timeseries_estimator.Config)
			cfg.Enabled = true
			cfg.EstimatorType = bm.estimatorType
			cfg.HLLPrecision = 10
			cfg.MemoryLimitMB = 1000 // High limit to avoid fallback during benchmark
			cfg.RefreshInterval = time.Hour
			
			sink := new(consumertest.MetricsSink)
			
			ctx := context.Background()
			settings := processor.Settings{
				TelemetrySettings: component.TelemetrySettings{
					Logger: zap.NewNop(),
				},
				ID: component.NewIDWithName(component.MustNewType("timeseries_estimator"), ""),
			}
			
			proc, _ := factory.CreateMetricsProcessor(ctx, settings, cfg, sink)
			proc.Start(ctx, nil)
			
			// Reset benchmark timer to exclude setup time
			b.ResetTimer()
			
			// Run benchmark
			for i := 0; i < b.N; i++ {
				_ = proc.ConsumeMetrics(ctx, bm.metrics)
			}
			
			// Stop timer and clean up
			b.StopTimer()
			proc.Shutdown(ctx)
		})
	}
}

// generateBenchmarkMetrics creates test metrics with a specified number of resources and metrics per resource
func generateBenchmarkMetrics(resources, metricsPerResource int) pmetric.Metrics {
	md := pmetric.NewMetrics()
	
	for i := 0; i < resources; i++ {
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("service.name", "bench-service-"+string(rune(i%10)))
		rm.Resource().Attributes().PutStr("host.name", "bench-host-"+string(rune(i%5)))
		rm.Resource().Attributes().PutInt("instance.id", int64(i))
		
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("bench-scope")
		
		for j := 0; j < metricsPerResource; j++ {
			m := sm.Metrics().AppendEmpty()
			m.SetName("bench.metric." + string(rune(j%5)))
			
			gauge := m.SetEmptyGauge()
			dp := gauge.DataPoints().AppendEmpty()
			dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			dp.SetDoubleValue(float64(i * j))
			
			// Add some attributes for uniqueness
			dp.Attributes().PutStr("dim1", "val-"+string(rune(i%10)))
			dp.Attributes().PutStr("dim2", "val-"+string(rune(j%10)))
			dp.Attributes().PutInt("count", int64(j))
		}
	}
	
	return md
}