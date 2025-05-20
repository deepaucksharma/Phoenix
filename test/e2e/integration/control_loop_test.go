// Package e2e contains end-to-end tests for the SA-OMF collector.
package e2e

import (
	"context"
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap/zaptest"

	"github.com/deepaucksharma/Phoenix/internal/connector/pic_connector"
	pic_control_ext "github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_pid"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// TestControlLoop verifies the basic closed-loop operation of the SA-OMF system.
func TestControlLoop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	host := testutils.NewTestHost()

	// Create pic_control extension
	extCfg := &pic_control_ext.Config{MaxPatchesPerMinute: 10, PatchCooldownSeconds: 0, SafeModeConfigs: map[string]any{}}
	ext, err := pic_control_ext.NewExtension(extCfg, logger)
	require.NoError(t, err)
	host.AddExtension(component.NewID(component.MustNewType("pic_control")), ext)
	require.NoError(t, ext.Start(ctx, host))
	defer func() { _ = ext.Shutdown(ctx) }()

	// Create mock adaptive processor and register with extension
	targetID := component.NewID(component.MustNewType("adaptive_topk"))
	mockProc := newMockProcessor()
	registerProcessor(ext, targetID, mockProc)

	// Create pic_connector exporter and start it
	connFactory := pic_connector.NewFactory()
	connCfg := connFactory.CreateDefaultConfig()
	conn, err := connFactory.CreateMetricsExporter(ctx, exporter.CreateSettings{
		ID:                component.NewID(component.MustNewType("pic_connector")),
		TelemetrySettings: component.TelemetrySettings{Logger: logger},
	}, connCfg)
	require.NoError(t, err)
	require.NoError(t, conn.Start(ctx, host))
	defer func() { _ = conn.Shutdown(ctx) }()

	// Create adaptive_pid processor with the connector as next consumer
	pidFactory := adaptive_pid.NewFactory()
	pidCfg := pidFactory.CreateDefaultConfig().(*adaptive_pid.Config)
	pidCfg.Controllers[0].KPIMetricName = "aemf_impact_adaptive_topk_resource_coverage_percent"
	pidCfg.Controllers[0].KPITargetValue = 0.9

	pidProc, err := pidFactory.CreateMetricsProcessor(ctx, processor.CreateSettings{
		TelemetrySettings: component.TelemetrySettings{Logger: logger},
		ID:                component.NewID(component.MustNewType("adaptive_pid")),
	}, pidCfg, conn)
	require.NoError(t, err)
	require.NoError(t, pidProc.Start(ctx, host))
	defer func() { _ = pidProc.Shutdown(ctx) }()

	// Send metrics below the KPI target
	metrics := createTestMetricsWithCoverage(0.7)
	require.NoError(t, pidProc.ConsumeMetrics(ctx, metrics))

	// Verify a patch was applied through the extension
	status, err := mockProc.GetConfigStatus(ctx)
	require.NoError(t, err)
	val, ok := status.Parameters["k_value"].(float64)
	require.True(t, ok)
	assert.Greater(t, val, float64(30))
}

// registerProcessor adds a processor to the extension's internal map using reflection
func registerProcessor(ext *pic_control_ext.Extension, id component.ID, proc interfaces.UpdateableProcessor) {
	v := reflect.ValueOf(ext).Elem().FieldByName("processors")
	m := reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
	m.SetMapIndex(reflect.ValueOf(id), reflect.ValueOf(proc))
}

// mockProcessor implements interfaces.UpdateableProcessor for testing
type mockProcessor struct {
	params map[string]any
}

func newMockProcessor() *mockProcessor {
	return &mockProcessor{params: make(map[string]any)}
}

func (m *mockProcessor) Start(context.Context, component.Host) error { return nil }
func (m *mockProcessor) Shutdown(context.Context) error              { return nil }

func (m *mockProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	m.params[patch.ParameterPath] = patch.NewValue
	return nil
}

func (m *mockProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	return interfaces.ConfigStatus{Parameters: m.params, Enabled: true}, nil
}

// createTestMetricsWithCoverage creates a metric with the specified coverage value
func createTestMetricsWithCoverage(coverage float64) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("adaptive_topk")
	m := sm.Metrics().AppendEmpty()
	m.SetName("aemf_impact_adaptive_topk_resource_coverage_percent")
	m.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(coverage)
	return md
}
