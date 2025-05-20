package adaptive_topk

import (
	"context"
	"testing"

	"github.com/deepaucksharma/Phoenix/test/testutils"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	atp "github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
)

func BenchmarkConsumeMetrics(b *testing.B) {
	factory := atp.NewFactory()
	cfg := factory.CreateDefaultConfig().(*atp.Config)
	cfg.KValue = 20

	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(context.Background(), processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}, ID: component.NewIDWithName(component.MustNewType("adaptive_topk"), "")}, cfg, sink)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := testutils.GenerateTestMetrics(200)
		_ = proc.ConsumeMetrics(context.Background(), metrics)
		sink.Reset()
	}
}
