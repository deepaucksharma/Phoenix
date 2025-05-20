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

	pic_control_ext "github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// TestPicControlExtension starts the extension, submits a patch, and shuts it down.
func TestPicControlExtension(t *testing.T) {
	// Create temporary policy file from example policy
	srcPolicy := filepath.Join("..", "..", "..", "configs", "default", "policy.yaml")
	data, err := os.ReadFile(srcPolicy)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "policy.yaml")
	require.NoError(t, os.WriteFile(policyPath, data, 0o644))

	cfg := &pic_control_ext.Config{
		PolicyFilePath:       policyPath,
		MaxPatchesPerMinute:  5,
		PatchCooldownSeconds: 0,
		SafeModeConfigs:      map[string]interface{}{},
		OpAMPConfig:          nil,
	}

	ext, err := pic_control_ext.NewExtension(cfg, zap.NewNop())
	require.NoError(t, err)

	host := testutils.NewTestHost()
	require.NoError(t, ext.Start(context.Background(), host))

	patch := interfaces.ConfigPatch{
		PatchID:             "test-patch",
		TargetProcessorName: component.MustNewID("dummy"),
		ParameterPath:       "enabled",
		NewValue:            true,
		Reason:              "test",
		Severity:            "normal",
		Source:              "test",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          60,
	}

	err = ext.SubmitConfigPatch(context.Background(), patch)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target processor not found")

	require.NoError(t, ext.Shutdown(context.Background()))
}
