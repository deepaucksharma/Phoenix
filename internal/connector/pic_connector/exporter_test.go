package pic_connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// fakePicControl implements pic_control_ext.PicControl for testing.
type fakePicControl struct {
	received []interfaces.ConfigPatch
}

func (f *fakePicControl) SubmitConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	f.received = append(f.received, patch)
	return nil
}

func (f *fakePicControl) Start(ctx context.Context, host component.Host) error { return nil }
func (f *fakePicControl) Shutdown(ctx context.Context) error                   { return nil }

// mockHost implements component.Host and allows registering extensions.
type mockHost struct {
	extensions map[component.ID]component.Component
}

func newMockHost() *mockHost {
	return &mockHost{extensions: make(map[component.ID]component.Component)}
}

func (m *mockHost) ReportFatalError(err error) {}
func (m *mockHost) GetFactory(kind component.Kind, componentType component.Type) component.Factory {
	return nil
}
func (m *mockHost) GetExtensions() map[component.ID]component.Component                   { return m.extensions }
func (m *mockHost) GetExporters() map[component.Type]map[component.ID]component.Component { return nil }
func (m *mockHost) GetProcessors() map[component.ID]component.Component                   { return nil }

// AddExtension registers an extension with the host.
func (m *mockHost) AddExtension(id component.ID, ext component.Component) {
	m.extensions[id] = ext
}

func newTestExporter(t *testing.T) *picConnectorExporter {
	exp, err := newExporter(&Config{}, exporter.Settings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}})
	require.NoError(t, err)
	return exp
}

func TestStartWithoutExtension(t *testing.T) {
	exp := newTestExporter(t)
	host := newMockHost()

	err := exp.Start(context.Background(), host)
	require.Error(t, err, "expected error when pic_control extension is missing")
}

func TestStartWithExtension(t *testing.T) {
	exp := newTestExporter(t)
	host := newMockHost()
	pc := &fakePicControl{}
	host.AddExtension(component.MustNewID("pic_control"), pc)

	err := exp.Start(context.Background(), host)
	require.NoError(t, err, "start should succeed when extension is present")
	require.NotNil(t, exp.picControl)
}

func TestConsumeMetrics(t *testing.T) {
	exp := newTestExporter(t)
	pc := &fakePicControl{}
	exp.picControl = pc

	patch := interfaces.ConfigPatch{
		PatchID:             "patch1",
		TargetProcessorName: component.NewID(component.MustNewType("processor")),
		ParameterPath:       "param",
		NewValue:            42,
		Reason:              "test",
		Severity:            "normal",
		Source:              "test",
	}
	md := testutils.GeneratePatchMetric(patch)

	err := exp.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)
	require.Len(t, pc.received, 1)
	require.Equal(t, patch.PatchID, pc.received[0].PatchID)
	require.Equal(t, patch.ParameterPath, pc.received[0].ParameterPath)
	require.EqualValues(t, patch.NewValue, pc.received[0].NewValue)
}

func TestExtractConfigPatches(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{"int", 5},
		{"double", 3.14},
		{"string", "foo"},
		{"bool", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			patch := interfaces.ConfigPatch{
				PatchID:             "patch",
				TargetProcessorName: component.NewID(component.MustNewType("processor")),
				ParameterPath:       "param",
				NewValue:            tc.value,
				Reason:              "reason",
				Severity:            "normal",
				Source:              "test",
			}
			md := testutils.GeneratePatchMetric(patch)

			patches := extractConfigPatches(md)
			require.Len(t, patches, 1)
			got := patches[0]
			require.Equal(t, patch.PatchID, got.PatchID)
			require.Equal(t, patch.ParameterPath, got.ParameterPath)
			require.EqualValues(t, tc.value, got.NewValue)
		})
	}
}
