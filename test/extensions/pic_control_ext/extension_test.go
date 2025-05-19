package pic_control_ext_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.uber.org/zap"

	piccontrol "github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

// mockHost implements component.Host for tests.
type mockHost struct{}

func (h *mockHost) GetExtensions() map[component.ID]component.Component { return nil }

func newMockHost() *mockHost { return &mockHost{} }

// mockProcessor implements interfaces.UpdateableProcessor for tests.
type mockProcessor struct {
	patches []interfaces.ConfigPatch
	status  interfaces.ConfigStatus
}

func newMockProcessor() *mockProcessor {
	return &mockProcessor{status: interfaces.ConfigStatus{Parameters: map[string]any{"test": 0}, Enabled: true}}
}

func (m *mockProcessor) Start(context.Context, component.Host) error { return nil }
func (m *mockProcessor) Shutdown(context.Context) error              { return nil }
func (m *mockProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	m.patches = append(m.patches, patch)
	if m.status.Parameters == nil {
		m.status.Parameters = make(map[string]any)
	}
	m.status.Parameters[patch.ParameterPath] = patch.NewValue
	return nil
}
func (m *mockProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	return m.status, nil
}

func TestPicControlExtension(t *testing.T) {
	// Create temporary policy file from testing config
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "policy.yaml")
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "configs", "testing", "policy.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(policyPath, data, 0o644))

	factory := piccontrol.NewFactory()
	cfg := factory.CreateDefaultConfig().(*piccontrol.Config)
	cfg.PolicyFilePath = policyPath
	cfg.MaxPatchesPerMinute = 2
	cfg.PatchCooldownSeconds = 0
	cfg.SafeModeConfigs = map[string]any{"processor": map[string]any{"test": 99}}

	settings := extensiontest.NewNopSettings(component.MustNewType("pic_control"))
	settings.TelemetrySettings.Logger = zap.NewNop()

	ext, err := factory.Create(context.Background(), settings, cfg)
	require.NoError(t, err)

	host := newMockHost()
	require.NoError(t, ext.Start(context.Background(), host))
	defer func() { require.NoError(t, ext.Shutdown(context.Background())) }()

	picExt := ext.(*piccontrol.Extension)
	procID := component.NewID(component.MustNewType("processor"))
	proc := newMockProcessor()
	processorsField := reflect.ValueOf(picExt).Elem().FieldByName("processors")
	processorsPtr := reflect.NewAt(processorsField.Type(), unsafe.Pointer(processorsField.UnsafeAddr())).Elem()
	processorsPtr.SetMapIndex(reflect.ValueOf(procID), reflect.ValueOf(proc))

	picCtrl := ext.(piccontrol.PicControl)
	patch := interfaces.ConfigPatch{
		PatchID:             "initial",
		TargetProcessorName: procID,
		ParameterPath:       "test",
		NewValue:            42,
	}

	// No processors registered in map before adding -> expect error
	processorsPtr.SetMapIndex(reflect.ValueOf(procID), reflect.Value{})
	err = picCtrl.SubmitConfigPatch(context.Background(), patch)
	assert.Error(t, err)

	// Add processor and submit patch successfully
	processorsPtr.SetMapIndex(reflect.ValueOf(procID), reflect.ValueOf(proc))
	err = picCtrl.SubmitConfigPatch(context.Background(), patch)
	require.NoError(t, err)
	require.Len(t, proc.patches, 1)
	status, _ := proc.GetConfigStatus(context.Background())
	assert.Equal(t, 42, status.Parameters["test"])

	// Rate limiting
	for i := 0; i < 3; i++ {
		patch.PatchID = fmt.Sprintf("p-%d", i)
		err = picCtrl.SubmitConfigPatch(context.Background(), patch)
		if i >= cfg.MaxPatchesPerMinute-1 {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
	assert.Len(t, proc.patches, 2)

	// Policy reload
	newData := bytes.ReplaceAll(data, []byte("shadow"), []byte("active"))
	require.NoError(t, os.WriteFile(policyPath, newData, 0o644))
	time.Sleep(200 * time.Millisecond)
	policyField := reflect.ValueOf(picExt).Elem().FieldByName("policy")
	policyPtr := reflect.NewAt(policyField.Type(), unsafe.Pointer(policyField.UnsafeAddr())).Elem()
	autonomy := policyPtr.Elem().FieldByName("GlobalSettings").FieldByName("AutonomyLevel").String()
	assert.Equal(t, "active", autonomy)

	// Safe mode
	safeField := reflect.ValueOf(picExt).Elem().FieldByName("safeMode")
	reflect.NewAt(safeField.Type(), unsafe.Pointer(safeField.UnsafeAddr())).Elem().SetBool(true)
	err = picCtrl.SubmitConfigPatch(context.Background(), patch)
	assert.Error(t, err)
	assert.Len(t, proc.patches, 2)
}
