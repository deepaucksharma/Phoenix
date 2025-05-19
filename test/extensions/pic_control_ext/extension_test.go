package pic_control_ext_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"

	pic "github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

func minimalPolicy(path string) string {
	return `global_settings:
  autonomy_level: shadow
  collector_cpu_safety_limit_mcores: 400
  collector_rss_safety_limit_mib: 350
processors_config:
  priority_tagger:
    enabled: true
    rules:
      - match: ".*"
        priority: low
  adaptive_topk:
    enabled: true
    k_value: 10
  cardinality_guardian:
    enabled: false
    max_unique: 100
  reservoir_sampler:
    enabled: false
    reservoir_size: 100
  others_rollup:
    enabled: false
pid_decider_config:
  controllers: []
pic_control_config:
  policy_file_path: ` + path + `
  max_patches_per_minute: 10
  patch_cooldown_seconds: 0
  safe_mode_processor_configs: {}
service: {}`
}

func TestPicControlExtension(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")
	require.NoError(t, os.WriteFile(policyFile, []byte(minimalPolicy(policyFile)), 0644))

	cfg := &pic.Config{PolicyFilePath: policyFile, MaxPatchesPerMinute: 10, PatchCooldownSeconds: 0}
	ext, err := pic.NewExtension(cfg, zap.NewNop())
	require.NoError(t, err)

	host := testutils.NewTestHost()
	require.NoError(t, ext.Start(ctx, host))
	defer ext.Shutdown(ctx)

	// Trigger policy watcher
	require.NoError(t, os.WriteFile(policyFile, []byte(minimalPolicy(policyFile)), 0644))
	time.Sleep(100 * time.Millisecond)

	patch := interfaces.ConfigPatch{
		PatchID:             "test",
		TargetProcessorName: component.NewID(component.MustNewType("priority_tagger")),
		ParameterPath:       "enabled",
		NewValue:            false,
		Timestamp:           time.Now().Unix(),
	}
	err = ext.SubmitConfigPatch(ctx, patch)
	assert.Error(t, err)
}
