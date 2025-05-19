package benchmark

import (
	"sync"

	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/deepaucksharma/Phoenix/test/testutils"
)

var metricsCache sync.Map // map[int]pmetric.Metrics

// GetBenchmarkMetrics returns cached metrics for the given process count.
// Metrics are generated once using testutils.GenerateTestMetrics and then
// reused across benchmark runs to avoid allocation overhead.
func GetBenchmarkMetrics(count int) pmetric.Metrics {
	if m, ok := metricsCache.Load(count); ok {
		return m.(pmetric.Metrics)
	}
	m := testutils.GenerateTestMetrics(count)
	metricsCache.Store(count, m)
	return m
}
