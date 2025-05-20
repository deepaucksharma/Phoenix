package pic_connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap/zaptest"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

type mockPicControl struct {
	patches []interfaces.ConfigPatch
}

func (m *mockPicControl) Start(context.Context, component.Host) error { return nil }
func (m *mockPicControl) Shutdown(context.Context) error              { return nil }
func (m *mockPicControl) SubmitConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	m.patches = append(m.patches, patch)
	return nil
}

func TestStartRetrievesPicControlExtension(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	exp, err := newExporter(&Config{}, exporter.Settings{TelemetrySettings: component.TelemetrySettings{Logger: logger}})
	require.NoError(t, err)

	host := testutils.NewTestHost()
	mock := &mockPicControl{}
	host.AddExtension(component.NewID(component.MustNewType("pic_control")), mock)

	err = exp.Start(ctx, host)
	require.NoError(t, err)
	assert.Equal(t, mock, exp.picControl)
}

func TestExtractConfigPatchesAndConfigPatchFromDataPoint(t *testing.T) {
	patchesIn := []interfaces.ConfigPatch{
		{PatchID: "p1", ParameterPath: "k_int", NewValue: 7},
		{PatchID: "p2", ParameterPath: "k_double", NewValue: 1.5},
		{PatchID: "p3", ParameterPath: "k_str", NewValue: "val"},
		{PatchID: "p4", ParameterPath: "k_bool", NewValue: true},
	}

	combined := pmetric.NewMetrics()

	for _, p := range patchesIn {
		md := testutils.GeneratePatchMetric(p)
		md.ResourceMetrics().MoveAndAppendTo(combined.ResourceMetrics())

		dp := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0)
		out := configPatchFromDataPoint(dp)
		require.NotNil(t, out)
		assert.Equal(t, p.PatchID, out.PatchID)
		assert.Equal(t, p.ParameterPath, out.ParameterPath)
		assert.Equal(t, p.NewValue, out.NewValue)
	}

	outPatches := extractConfigPatches(combined)
	require.Len(t, outPatches, len(patchesIn))
	for i, out := range outPatches {
		exp := patchesIn[i]
		assert.Equal(t, exp.PatchID, out.PatchID)
		assert.Equal(t, exp.ParameterPath, out.ParameterPath)
		assert.Equal(t, exp.NewValue, out.NewValue)
		assert.Equal(t, 300, out.TTLSeconds)
		assert.NotEqual(t, "", out.TargetProcessorName.String())
		assert.NotZero(t, out.Timestamp)
	}
}

func TestConsumeMetricsSubmitsPatches(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	exp, err := newExporter(&Config{}, exporter.Settings{TelemetrySettings: component.TelemetrySettings{Logger: logger}})
	require.NoError(t, err)

	host := testutils.NewTestHost()
	mock := &mockPicControl{}
	host.AddExtension(component.NewID(component.MustNewType("pic_control")), mock)

	require.NoError(t, exp.Start(ctx, host))

	patch := interfaces.ConfigPatch{PatchID: "p1", ParameterPath: "k_value", NewValue: 99}
	md := testutils.GeneratePatchMetric(patch)
	require.NoError(t, exp.ConsumeMetrics(ctx, md))

	require.Len(t, mock.patches, 1)
	assert.Equal(t, patch.ParameterPath, mock.patches[0].ParameterPath)
	assert.Equal(t, patch.NewValue, mock.patches[0].NewValue)

	extra := interfaces.ConfigPatch{PatchID: "p2", ParameterPath: "k_value", NewValue: 100}
	md2 := testutils.GeneratePatchMetric(extra)
	combined := pmetric.NewMetrics()
	md.ResourceMetrics().MoveAndAppendTo(combined.ResourceMetrics())
	md2.ResourceMetrics().MoveAndAppendTo(combined.ResourceMetrics())
	require.NoError(t, exp.ConsumeMetrics(ctx, combined))
	require.Len(t, mock.patches, 3)
	assert.Equal(t, "p2", mock.patches[2].PatchID)
}
