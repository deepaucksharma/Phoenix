package benchmark

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/processor/priority_tagger"
)

func BenchmarkPriorityTaggerEndToEnd(b *testing.B) {
	metrics := GetBenchmarkMetrics(1000)

	sink := new(consumertest.MetricsSink)
	factory := priority_tagger.NewFactory()
	cfg := factory.CreateDefaultConfig()

	proc, err := factory.CreateMetricsProcessor(
		context.Background(),
		processor.CreateSettings{
			TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
			ID:                component.NewID("priority_tagger"),
		},
		cfg,
		sink,
	)
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
