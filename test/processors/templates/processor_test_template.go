// Package processors provides a standardized test framework for processors.
package processors

import (
	"context"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	interfacetests "github.com/deepaucksharma/Phoenix/test/interfaces"
)

// ProcessorTestCase defines a standardized test case for processors
type ProcessorTestCase struct {
	Name           string
	InputMetrics   pmetric.Metrics
	ExpectedOutput func(pmetric.Metrics) bool
	ConfigPatches  []interfaces.ConfigPatch
}

// RunProcessorTests executes a standardized set of tests for any processor
func RunProcessorTests(t *testing.T, factory processor.Factory, defaultConfig component.Config, testCases []ProcessorTestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Setup
			next := new(consumertest.MetricsSink)
			processor, err := factory.CreateMetricsProcessor(
				context.Background(),
				processor.CreateSettings{},
				defaultConfig,
				next,
			)
			require.NoError(t, err)
			
			// Start the processor with a no-op host
			err = processor.Start(context.Background(), nil)
			require.NoError(t, err)
			
			// Test updateable interface if implemented
			if upProc, ok := processor.(interfaces.UpdateableProcessor); ok {
				// Verify it implements the interface correctly
				interfacetests.TestUpdateableProcessor(t, upProc)
				
				// Apply any test-specific config patches
				for _, patch := range tc.ConfigPatches {
					err = upProc.OnConfigPatch(context.Background(), patch)
					require.NoError(t, err, "Failed to apply config patch: %v", err)
				}
			}
			
			// Process input metrics
			err = processor.ConsumeMetrics(context.Background(), tc.InputMetrics)
			require.NoError(t, err, "Failed to consume metrics: %v", err)
			
			// Verify output
			allMetrics := next.AllMetrics()
			require.NotEmpty(t, allMetrics, "No metrics were produced")
			assert.True(t, tc.ExpectedOutput(allMetrics[0]), "Output metrics did not meet expectations")
			
			// Shutdown
			err = processor.Shutdown(context.Background())
			require.NoError(t, err, "Failed to shut down processor: %v", err)
		})
	}
}

// GenerateTestMetrics creates a set of test metrics for processor testing
func GenerateTestMetrics(processNames []string) pmetric.Metrics {
	md := pmetric.NewMetrics()
	
	for i, procName := range processNames {
		// Add resource metrics
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("process.name", procName)
		rm.Resource().Attributes().PutStr("process.pid", string(1000+i))
		
		// Add scope metrics
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("host")
		
		// Add CPU metric
		cpuMetric := sm.Metrics().AppendEmpty()
		cpuMetric.SetName("cpu.usage")
		cpuMetric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(float64(i) * 0.1)
		
		// Add memory metric
		memMetric := sm.Metrics().AppendEmpty()
		memMetric.SetName("memory.usage")
		memMetric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(float64(i * 100))
	}
	
	return md
}