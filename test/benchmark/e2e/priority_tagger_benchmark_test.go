package benchmark

import (
	"context"
	"os"
	"runtime/pprof"
	"testing"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/processor/priority_tagger"
)

// Helper function to start CPU profiling
func startCPUProfile(b *testing.B, name string) func() {
	f, err := os.Create(name)
	if err != nil {
		b.Fatal(err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		b.Fatal(err)
	}
	return func() {
		pprof.StopCPUProfile()
		f.Close()
	}
}

// Generate benchmark metrics
func GetBenchmarkMetrics(numResources int) pmetric.Metrics {
	md := pmetric.NewMetrics()

	// Create numResources resources with metrics
	for i := 0; i < numResources; i++ {
		rm := md.ResourceMetrics().AppendEmpty()
		resource := rm.Resource()
		
		// Add some attributes to the resource
		resource.Attributes().PutStr("process.name", "process-name")
		resource.Attributes().PutStr("service.name", "service-name")
		resource.Attributes().PutInt("process.pid", int64(i))
		
		// Add a scope metric
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("benchmark-scope")
		
		// Add a gauge metric
		m := sm.Metrics().AppendEmpty()
		m.SetName("test.metric")
		m.SetEmptyGauge()
		dp := m.Gauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(float64(i))
	}
	
	return md
}

func BenchmarkPriorityTaggerEndToEnd(b *testing.B) {
	metrics := GetBenchmarkMetrics(1000)

	sink := new(consumertest.MetricsSink)
	factory := priority_tagger.NewFactory()
	cfg := factory.CreateDefaultConfig()

	// Create processor with updated API
	processorSettings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewID("priority_tagger"),
	}
	
	// Create the processor directly without using WithMetrics
	proc, err := factory.CreateDefaultMetricsProcessor(context.Background(), processorSettings, cfg, sink)
	if err != nil {
		b.Fatalf("failed to create processor: %v", err)
	}

	stop := startCPUProfile(b, "priority_tagger_cpu.prof")
	defer stop()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = proc.ConsumeMetrics(context.Background(), metrics)
	}
}
