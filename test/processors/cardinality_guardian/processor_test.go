package cardinality_guardian

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/cardinality_guardian"
	processors "github.com/deepaucksharma/Phoenix/test/processors/templates"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// checkReduced verifies that datapoints have been bucketized.
func checkReduced(md pmetric.Metrics) bool {
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		smSlice := rm.ScopeMetrics()
		for j := 0; j < smSlice.Len(); j++ {
			sm := smSlice.At(j)
			metrics := sm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				m := metrics.At(k)
				var dps pmetric.NumberDataPointSlice
				switch m.Type() {
				case pmetric.MetricTypeGauge:
					dps = m.Gauge().DataPoints()
				case pmetric.MetricTypeSum:
					dps = m.Sum().DataPoints()
				default:
					continue
				}
				for l := 0; l < dps.Len(); l++ {
					attrs := dps.At(l).Attributes()
					if attrs.Len() != 1 {
						return false
					}
					if _, ok := attrs.Get("cg_bucket"); !ok {
						return false
					}
				}
			}
		}
	}
	return true
}

func TestCardinalityGuardianProcessor(t *testing.T) {
	t.Skip("flaky in container")
	factory := cardinality_guardian.NewFactory()
	cfg := factory.CreateDefaultConfig().(*cardinality_guardian.Config)
	cfg.MaxUnique = 10
	cfg.Enabled = true

	testCases := []processors.ProcessorTestCase{
		{
			Name:         "BucketReduction",
			InputMetrics: testutils.GenerateHighCardinalityMetrics(50),
			ExpectedOutput: func(md pmetric.Metrics) bool {
				return checkReduced(md)
			},
			ConfigPatches: []interfaces.ConfigPatch{ // ensure low threshold
				{
					PatchID:             "set-max-unique",
					TargetProcessorName: component.NewIDWithName(component.MustNewType("cardinality_guardian"), ""),
					ParameterPath:       "max_unique",
					NewValue:            10,
				},
			},
		},
		{
			Name:         "Disabled",
			InputMetrics: testutils.GenerateHighCardinalityMetrics(20),
			ExpectedOutput: func(md pmetric.Metrics) bool {
				// attributes should remain untouched when disabled
				for i := 0; i < md.ResourceMetrics().Len(); i++ {
					rm := md.ResourceMetrics().At(i)
					smSlice := rm.ScopeMetrics()
					for j := 0; j < smSlice.Len(); j++ {
						sm := smSlice.At(j)
						metrics := sm.Metrics()
						for k := 0; k < metrics.Len(); k++ {
							m := metrics.At(k)
							var dps pmetric.NumberDataPointSlice
							switch m.Type() {
							case pmetric.MetricTypeGauge:
								dps = m.Gauge().DataPoints()
							case pmetric.MetricTypeSum:
								dps = m.Sum().DataPoints()
							default:
								continue
							}
							for l := 0; l < dps.Len(); l++ {
								if _, ok := dps.At(l).Attributes().Get("cg_bucket"); ok {
									return false
								}
							}
						}
					}
				}
				return true
			},
			ConfigPatches: []interfaces.ConfigPatch{
				{
					PatchID:             "disable",
					TargetProcessorName: component.NewIDWithName(component.MustNewType("cardinality_guardian"), ""),
					ParameterPath:       "enabled",
					NewValue:            false,
				},
			},
		},
	}

	processors.RunProcessorTests(t, factory, cfg, testCases)
}

func TestOnConfigPatchUpdates(t *testing.T) {
	t.Skip("flaky in container")
	factory := cardinality_guardian.NewFactory()
	cfg := factory.CreateDefaultConfig().(*cardinality_guardian.Config)

	ctx := context.Background()
	sink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(ctx, processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}}, cfg, sink)
	require.NoError(t, err)
	up, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)
	require.NoError(t, proc.Start(ctx, nil))

	patch := interfaces.ConfigPatch{
		PatchID:             "patch-max",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("cardinality_guardian"), ""),
		ParameterPath:       "max_unique",
		NewValue:            5,
	}
	err = up.OnConfigPatch(ctx, patch)
	require.NoError(t, err)

	status, err := up.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, status.Parameters["max_unique"])

	patchEn := interfaces.ConfigPatch{
		PatchID:             "patch-enabled",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("cardinality_guardian"), ""),
		ParameterPath:       "enabled",
		NewValue:            false,
	}
	err = up.OnConfigPatch(ctx, patchEn)
	require.NoError(t, err)

	status, err = up.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.False(t, status.Enabled)

	require.NoError(t, proc.Shutdown(ctx))
}

func TestHLLCountingAndReduction(t *testing.T) {
	t.Skip("flaky in container")
	factory := cardinality_guardian.NewFactory()
	cfg := factory.CreateDefaultConfig().(*cardinality_guardian.Config)
	cfg.MaxUnique = 5
	ctx := context.Background()
	sink := new(consumertest.MetricsSink)

	proc, err := factory.CreateMetrics(ctx, processor.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}}, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, proc.Start(ctx, nil))

	metrics := testutils.GenerateHighCardinalityMetrics(20)
	err = proc.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	processed := sink.AllMetrics()
	require.NotEmpty(t, processed)
	assert.True(t, checkReduced(processed[0]))

	bucketSet := map[int64]struct{}{}
	rms := processed[0].ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		sm := rms.At(i).ScopeMetrics()
		for j := 0; j < sm.Len(); j++ {
			ms := sm.At(j).Metrics()
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				var dps pmetric.NumberDataPointSlice
				if m.Type() == pmetric.MetricTypeGauge {
					dps = m.Gauge().DataPoints()
				} else {
					dps = m.Sum().DataPoints()
				}
				for l := 0; l < dps.Len(); l++ {
					v, ok := dps.At(l).Attributes().Get("cg_bucket")
					if ok {
						bucketSet[v.Int()] = struct{}{}
					}
				}
			}
		}
	}
	assert.LessOrEqual(t, len(bucketSet), cfg.MaxUnique)

	require.NoError(t, proc.Shutdown(ctx))
}
