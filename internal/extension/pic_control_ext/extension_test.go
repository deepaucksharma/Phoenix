package pic_control_ext

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap/zaptest"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

// mockProcessor used for internal tests
type mockProcessor struct {
	params  map[string]any
	enabled bool
}

func newMock() *mockProcessor {
	return &mockProcessor{params: map[string]any{"k_value": 0}, enabled: true}
}

func (m *mockProcessor) Start(context.Context, component.Host) error { return nil }
func (m *mockProcessor) Shutdown(context.Context) error              { return nil }
func (m *mockProcessor) OnConfigPatch(ctx context.Context, p interfaces.ConfigPatch) error {
	m.params[p.ParameterPath] = p.NewValue
	return nil
}
func (m *mockProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	return interfaces.ConfigStatus{Parameters: m.params, Enabled: m.enabled}, nil
}

func TestInternalPolicyAndSafeMode(t *testing.T) {
	cfg := &Config{MaxPatchesPerMinute: 2, PatchCooldownSeconds: 1, SafeModeConfigs: map[string]any{"adaptive_topk": map[string]any{"k_value": 5}}}
	ext, err := NewExtension(cfg, zaptest.NewLogger(t))
	if err != nil {
		t.Fatal(err)
	}
	pid := component.NewID(component.MustNewType("adaptive_topk"))
	mp := newMock()
	ext.processors[pid] = mp

	policyYaml := []byte(`global_settings:
  autonomy_level: shadow
  collector_cpu_safety_limit_mcores: 100
  collector_rss_safety_limit_mib: 200
processors_config:
  priority_tagger:
    enabled: true
  adaptive_topk:
    enabled: true
    k_value: 20
    k_min: 1
    k_max: 40
  cardinality_guardian:
    enabled: false
    max_unique: 100
  reservoir_sampler:
    enabled: false
    reservoir_size: 50
  others_rollup:
    enabled: false
pid_decider_config:
  controllers:
    - name: test
      enabled: false
      kpi_metric_name: a
      kpi_target_value: 1
      output_config_patches: []
pic_control_config:
  policy_file_path: x
  max_patches_per_minute: 2
  patch_cooldown_seconds: 1
  safe_mode_processor_configs:
    adaptive_topk:
      k_value: 5
`)
	if err := ext.loadPolicyBytes(policyYaml); err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if mp.params["k_value"] != 20 {
		t.Fatalf("policy not applied")
	}
	if err := ext.enterSafeMode(); err != nil {
		t.Fatal(err)
	}
	if mp.params["k_value"] != 5 {
		t.Fatalf("safe mode not applied")
	}
	if err := ext.exitSafeMode(); err != nil {
		t.Fatal(err)
	}
	if mp.params["k_value"] != 20 {
		t.Fatalf("policy reapplied")
	}
}

func TestInternalPatchRateLimit(t *testing.T) {
	cfg := &Config{MaxPatchesPerMinute: 1, PatchCooldownSeconds: 1}
	ext, _ := NewExtension(cfg, zaptest.NewLogger(t))
	pid := component.NewID(component.MustNewType("adaptive_topk"))
	mp := newMock()
	ext.processors[pid] = mp
	ctx := context.Background()
	if err := ext.SubmitConfigPatch(ctx, interfaces.ConfigPatch{PatchID: "1", TargetProcessorName: pid, ParameterPath: "k_value", NewValue: 1}); err != nil {
		t.Fatal(err)
	}
	if err := ext.SubmitConfigPatch(ctx, interfaces.ConfigPatch{PatchID: "2", TargetProcessorName: pid, ParameterPath: "k_value", NewValue: 2}); err == nil {
		t.Fatalf("expected rate limit")
	}
	time.Sleep(time.Second)
	if err := ext.SubmitConfigPatch(ctx, interfaces.ConfigPatch{PatchID: "3", TargetProcessorName: pid, ParameterPath: "k_value", NewValue: 3}); err == nil {
		t.Fatalf("expected rate limit")
	}
}
