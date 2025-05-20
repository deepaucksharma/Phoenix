package pic_control_ext_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/deepaucksharma/Phoenix/internal/connector/pic_connector"
	piccontrol "github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	pic_control_ext "github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
	"github.com/deepaucksharma/Phoenix/pkg/policy"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// testHost implements component.Host for unit testing.
type testHost struct {
	exts map[component.ID]component.Component
}

func (h *testHost) GetExtensions() map[component.ID]component.Component { return h.exts }

//go:linkname enterSafeMode github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext.(*Extension).enterSafeMode
func enterSafeMode(*piccontrol.Extension) error

//go:linkname exitSafeMode github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext.(*Extension).exitSafeMode
func exitSafeMode(*piccontrol.Extension) error

//go:linkname loadPolicyBytes github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext.(*Extension).loadPolicyBytes
func loadPolicyBytes(*piccontrol.Extension, []byte) error

// mockProcessor implements interfaces.UpdateableProcessor for testing.
type mockProcessor struct {
	params  map[string]any
	enabled bool
	patches []interfaces.ConfigPatch
}

func getProcessors(e *piccontrol.Extension) map[component.ID]interfaces.UpdateableProcessor {
	v := reflect.ValueOf(e).Elem().FieldByName("processors")
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Interface().(map[component.ID]interfaces.UpdateableProcessor)
}

func getPatchHistory(e *piccontrol.Extension) []interfaces.ConfigPatch {
	v := reflect.ValueOf(e).Elem().FieldByName("patchHistory")
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Interface().([]interfaces.ConfigPatch)
}

func getPolicy(e *piccontrol.Extension) *policy.Policy {
	v := reflect.ValueOf(e).Elem().FieldByName("policy")
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Interface().(*policy.Policy)
}

func getConfig(e *piccontrol.Extension) *piccontrol.Config {
	v := reflect.ValueOf(e).Elem().FieldByName("config")
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Interface().(*piccontrol.Config)
}

func isSafeMode(e *piccontrol.Extension) bool {
	v := reflect.ValueOf(e).Elem().FieldByName("safeMode")
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Interface().(bool)
}

func newMockProcessor() *mockProcessor {
	return &mockProcessor{params: make(map[string]any), enabled: true}
}

func (m *mockProcessor) Start(context.Context, component.Host) error { return nil }
func (m *mockProcessor) Shutdown(context.Context) error              { return nil }

func (m *mockProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	m.patches = append(m.patches, patch)
	if patch.ParameterPath == "enabled" {
		v, ok := patch.NewValue.(bool)
		if !ok {
			return assert.AnError
		}
		m.enabled = v
		return nil
	}
	m.params[patch.ParameterPath] = patch.NewValue
	return nil
}

func (m *mockProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	cp := make(map[string]any)
	for k, v := range m.params {
		cp[k] = v
	}
	return interfaces.ConfigStatus{Parameters: cp, Enabled: m.enabled}, nil
}

const minimalPolicy = `
global_settings:
  autonomy_level: shadow
  collector_cpu_safety_limit_mcores: 400
  collector_rss_safety_limit_mib: 350
processors_config:
  priority_tagger:
    enabled: true
  adaptive_topk:
    enabled: true
    k_value: 30
    k_min: 10
    k_max: 60
  cardinality_guardian:
    enabled: false
    max_unique: 1000
  reservoir_sampler:
    enabled: false
    reservoir_size: 100
  others_rollup:
    enabled: false
pid_decider_config:
  controllers:
    - name: test
      enabled: false
      kpi_metric_name: test_metric
      kpi_target_value: 1
      output_config_patches: []
pic_control_config:
  policy_file_path: /etc/sa-omf/policy.yaml
  max_patches_per_minute: 10
  patch_cooldown_seconds: 1
  safe_mode_processor_configs:
    adaptive_topk:
      k_value: 10
`

// createExtension with a single mock processor registered
func createExtension(t *testing.T) (*piccontrol.Extension, component.ID, *mockProcessor) {
	cfg := &piccontrol.Config{
		MaxPatchesPerMinute:  2,
		PatchCooldownSeconds: 1,
		SafeModeConfigs: map[string]any{
			"adaptive_topk": map[string]any{"k_value": 10},
		},
	}
	ext, err := piccontrol.NewExtension(cfg, zaptest.NewLogger(t))
	require.NoError(t, err)
	procID := component.NewID(component.MustNewType("adaptive_topk"))
	mp := newMockProcessor()
	getProcessors(ext)[procID] = mp
	return ext, procID, mp
}

// createStartedExtension starts the extension with a temporary policy file.
func createStartedExtension(t *testing.T) (*piccontrol.Extension, component.ID, *mockProcessor) {
	dir := t.TempDir()
	policyFile := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(policyFile, []byte(minimalPolicy), 0o644))

	cfg := &piccontrol.Config{
		PolicyFilePath:       policyFile,
		MaxPatchesPerMinute:  2,
		PatchCooldownSeconds: 1,
		SafeModeConfigs: map[string]any{
			"adaptive_topk": map[string]any{"k_value": 10},
		},
	}
	ext, err := piccontrol.NewExtension(cfg, zaptest.NewLogger(t))
	require.NoError(t, err)
	procID := component.NewID(component.MustNewType("adaptive_topk"))
	mp := newMockProcessor()
	getProcessors(ext)[procID] = mp

	ctx := context.Background()
	require.NoError(t, ext.Start(ctx, mockHost{}))
	t.Cleanup(func() { _ = ext.Shutdown(ctx) })
	return ext, procID, mp
}

type mockHost struct{}

func (mockHost) GetExtensions() map[component.ID]component.Component { return nil }

func TestLoadPolicyAppliesConfig(t *testing.T) {
	ext, id, proc := createStartedExtension(t)

	assert.Equal(t, 30, proc.params["k_value"])
	assert.Equal(t, 10, proc.params["k_min"])
	assert.Equal(t, 60, proc.params["k_max"])
	// ensure policy stored
	require.NotNil(t, getPolicy(ext))
	// verify patch history length (initial patches not recorded)
	assert.Empty(t, getPatchHistory(ext))
	_ = id
}

func TestSubmitConfigPatch(t *testing.T) {
	ext, id, proc := createExtension(t)
	ctx := context.Background()

	patch := interfaces.ConfigPatch{
		PatchID:             "p1",
		TargetProcessorName: id,
		ParameterPath:       "k_value",
		NewValue:            55,
	}
	err := ext.SubmitConfigPatch(ctx, patch)
	require.NoError(t, err)
	assert.Equal(t, 55, proc.params["k_value"])
	assert.Len(t, getPatchHistory(ext), 1)
}

func TestSubmitConfigPatchTTLExpired(t *testing.T) {
	ext, id, _ := createExtension(t)
	ctx := context.Background()

	patch := interfaces.ConfigPatch{
		PatchID:             "exp",
		TargetProcessorName: id,
		ParameterPath:       "k_value",
		NewValue:            40,
		Timestamp:           time.Now().Add(-2 * time.Second).Unix(),
		TTLSeconds:          1,
	}
	err := ext.SubmitConfigPatch(ctx, patch)
	assert.Error(t, err)
}

func TestSubmitConfigPatchRateLimit(t *testing.T) {
	ext, id, _ := createExtension(t)
	ctx := context.Background()
	getConfig(ext).MaxPatchesPerMinute = 1
	getConfig(ext).PatchCooldownSeconds = 0

	patch := interfaces.ConfigPatch{PatchID: "p1", TargetProcessorName: id, ParameterPath: "k_value", NewValue: 1}
	require.NoError(t, ext.SubmitConfigPatch(ctx, patch))
	patch2 := interfaces.ConfigPatch{PatchID: "p2", TargetProcessorName: id, ParameterPath: "k_value", NewValue: 2}
	err := ext.SubmitConfigPatch(ctx, patch2)
	assert.Error(t, err)
}

func TestSubmitConfigPatchCooldown(t *testing.T) {
	ext, id, _ := createExtension(t)
	ctx := context.Background()
	getConfig(ext).MaxPatchesPerMinute = 10
	getConfig(ext).PatchCooldownSeconds = 5

	require.NoError(t, ext.SubmitConfigPatch(ctx, interfaces.ConfigPatch{PatchID: "p1", TargetProcessorName: id, ParameterPath: "k_value", NewValue: 1}))
	err := ext.SubmitConfigPatch(ctx, interfaces.ConfigPatch{PatchID: "p2", TargetProcessorName: id, ParameterPath: "k_value", NewValue: 2})
	assert.Error(t, err)
}

func TestSafeModeBehavior(t *testing.T) {
	ext, id, proc := createStartedExtension(t)
	// enter safe mode
	require.NoError(t, enterSafeMode(ext))
	assert.True(t, isSafeMode(ext))
	assert.Equal(t, 10, proc.params["k_value"])

	ctx := context.Background()
	err := ext.SubmitConfigPatch(ctx, interfaces.ConfigPatch{PatchID: "p1", TargetProcessorName: id, ParameterPath: "k_value", NewValue: 20})
	assert.Error(t, err)

	// exit safe mode
	require.NoError(t, exitSafeMode(ext))
	assert.False(t, isSafeMode(ext))
	// policy re-applied
	assert.Equal(t, 30, proc.params["k_value"])
}

func TestPicControlExtension(t *testing.T) {
	ctx := context.Background()

	// Create adaptive_topk processor
	procFactory := adaptive_topk.NewFactory()
	procCfg := procFactory.CreateDefaultConfig().(*adaptive_topk.Config)
	procSettings := processor.Settings{
		ID:                component.NewIDWithName(component.MustNewType("adaptive_topk"), ""),
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
	}
	sink := new(consumertest.MetricsSink)
	proc, err := procFactory.CreateMetrics(ctx, procSettings, procCfg, sink)
	require.NoError(t, err)
	upProc, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok)

	patchID := component.NewID(component.MustNewType("processor"))

	// Create pic_control extension without starting watchers
	extCfg := &pic_control_ext.Config{MaxPatchesPerMinute: 10, PatchCooldownSeconds: 0, SafeModeConfigs: map[string]interface{}{}}
	ext, err := pic_control_ext.NewExtension(extCfg, zap.NewNop())
	require.NoError(t, err)

	// Register processor with extension
	ext.RegisterProcessor(patchID, upProc)

	// Create host exposing the extension
	host := &testHost{exts: map[component.ID]component.Component{component.NewIDWithName(component.MustNewType("pic_control"), ""): ext}}

	// Create connector exporter and start it so it discovers the extension
	connFactory := pic_connector.NewFactory()
	connCfg := connFactory.CreateDefaultConfig()
	connSettings := exporter.Settings{ID: component.NewIDWithName(component.MustNewType("pic_connector"), ""), TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}}
	conn, err := connFactory.CreateMetrics(ctx, connSettings, connCfg)
	require.NoError(t, err)
	require.NoError(t, conn.Start(ctx, host))

	// Generate a config patch using testutils
	patch := interfaces.ConfigPatch{
		PatchID:             "patch1",
		TargetProcessorName: patchID,
		ParameterPath:       "k_value",
		NewValue:            20,
		Source:              "pid_decider",
	}
	patchMetrics := testutils.GeneratePatchMetric(patch)

	// Apply patch through the connector
	require.NoError(t, conn.ConsumeMetrics(ctx, patchMetrics))

	status, err := upProc.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, 20, status.Parameters["k_value"])

	assert.NoError(t, conn.Shutdown(ctx))
}
