package testutils

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// MetricsSink is a simple metrics consumer used for testing.
// It stores all metrics batches that are consumed.
type MetricsSink struct {
	mu      sync.Mutex
	batches []pmetric.Metrics
}

// Capabilities returns the consumer capabilities.
func (s *MetricsSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics records the provided metrics batch.
func (s *MetricsSink) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := pmetric.NewMetrics()
	md.CopyTo(cp)
	s.batches = append(s.batches, cp)
	return nil
}

// AllMetrics returns all metrics batches that were consumed.
func (s *MetricsSink) AllMetrics() []pmetric.Metrics {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]pmetric.Metrics, len(s.batches))
	for i, m := range s.batches {
		out[i] = m
	}
	return out
}
