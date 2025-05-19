package pic_control_ext

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

// mockHost implements a minimal component.Host for testing
type mockHost struct {
	processors map[component.ID]component.Component
	extensions map[component.ID]component.Component
}

func newMockHost() *mockHost {
	return &mockHost{
		processors: make(map[component.ID]component.Component),
		extensions: make(map[component.ID]component.Component),
	}
}

func (h *mockHost) ReportFatalError(err error) {
	// Do nothing in tests
}

func (h *mockHost) GetFactory(kind component.Kind, id component.ID) component.Factory {
	return nil
}

func (h *mockHost) GetExtensions() map[component.ID]component.Extension {
	extensions := make(map[component.ID]component.Extension)
	for id, ext := range h.extensions {
		extensions[id] = ext.(component.Extension)
	}
	return extensions
}

func (h *mockHost) GetExporters() map[component.DataType]map[component.ID]component.Component {
	return nil
}

func (h *mockHost) GetProcessors() map[component.ID]component.Processor {
	processors := make(map[component.ID]component.Processor)
	for id, proc := range h.processors {
		processors[id] = proc.(component.Processor)
	}
	return processors
}

func (h *mockHost) AddProcessor(id component.ID, processor component.Component) {
	h.processors[id] = processor
}

func (h *mockHost) AddExtension(id component.ID, extension component.Component) {
	h.extensions[id] = extension
}

// mockProcessor implements a minimal UpdateableProcessor for testing
type mockProcessor struct {
	component.Component
	interfaces.UpdateableProcessor
	
	enabled       bool
	parameters    map[string]interface{}
	patchApplied  bool
	lastPatch     interfaces.ConfigPatch
}

func newMockProcessor() *mockProcessor {
	return &mockProcessor{
		enabled:      true,
		parameters:   make(map[string]interface{}),
		patchApplied: false,
	}
}

func (p *mockProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (p *mockProcessor) Shutdown(ctx context.Context) error {
	return nil
}

func (p *mockProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.patchApplied = true
	p.lastPatch = patch
	
	if patch.ParameterPath == "enabled" {
		if enabled, ok := patch.NewValue.(bool); ok {
			p.enabled = enabled
		}
	} else {
		p.parameters[patch.ParameterPath] = patch.NewValue
	}
	
	return nil
}

func (p *mockProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	return interfaces.ConfigStatus{
		Parameters: p.parameters,
		Enabled:    p.enabled,
	}, nil
}

func TestPicControlExtension(t *testing.T) {
	// Create a temporary directory for policy files
	tmpDir, err := os.MkdirTemp("", "piccontrol-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	
	// Create a temporary policy file
	policyPath := filepath.Join(tmpDir, "policy.yaml")
	initialPolicy := `
global_settings:
  autonomy_level: shadow
  collector_cpu_safety_limit_mcores: 300
  collector_rss_safety_limit_mib: 300

processors_config:
  priority_tagger:
    enabled: true
    rules:
    - match: "nginx.*"
      priority: "high"
  
  adaptive_topk:
    enabled: true
    k_value: 30
    k_min: 10
    k_max: 60

pic_control_config:
  policy_file_path: ` + policyPath + `
  max_patches_per_minute: 3
  patch_cooldown_seconds: 1
`
	err = os.WriteFile(policyPath, []byte(initialPolicy), 0644)
	require.NoError(t, err)
	
	// Create the extension factory
	factory := piccontrolext.NewFactory()
	require.NotNil(t, factory)
	
	// Create a default configuration
	cfg := factory.CreateDefaultConfig().(*piccontrolext.Config)
	cfg.PolicyFilePath = policyPath
	cfg.MaxPatchesPerMinute = 5
	cfg.PatchCooldownSeconds = 1
	
	// Create the extension
	ctx := context.Background()
	settings := extension.CreateSettings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewID("pic_control"),
	}
	
	ext, err := factory.CreateExtension(ctx, settings, cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)
	
	// Ensure it implements the PicControl interface
	picControl, ok := ext.(pic_control_ext.PicControl)
	require.True(t, ok, "Extension does not implement PicControl")
	
	// Create a mock host
	host := newMockHost()
	
	// Add a mock processor
	procID := component.NewID("priority_tagger")
	mockProc := newMockProcessor()
	host.AddProcessor(procID, mockProc)
	
	// Start the extension
	err = ext.Start(ctx, host)
	require.NoError(t, err)
	
	// Test submitting a config patch
	patch := interfaces.ConfigPatch{
		PatchID:             "test-patch",
		TargetProcessorName: procID,
		ParameterPath:       "test_param",
		NewValue:            42,
		Reason:              "Unit test",
		Severity:            "normal",
		Source:              "test",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}
	
	err = picControl.SubmitConfigPatch(ctx, patch)
	require.NoError(t, err)
	
	// Verify the patch was applied to the processor
	assert.True(t, mockProc.patchApplied)
	assert.Equal(t, patch.PatchID, mockProc.lastPatch.PatchID)
	assert.Equal(t, 42, mockProc.parameters["test_param"])
	
	// Test policy file watching by changing the policy
	updatedPolicy := `
global_settings:
  autonomy_level: active
  collector_cpu_safety_limit_mcores: 300
  collector_rss_safety_limit_mib: 300

processors_config:
  priority_tagger:
    enabled: true
    rules:
    - match: "apache.*"
      priority: "high"
  
  adaptive_topk:
    enabled: true
    k_value: 25
    k_min: 10
    k_max: 60

pic_control_config:
  policy_file_path: ` + policyPath + `
  max_patches_per_minute: 3
  patch_cooldown_seconds: 1
`
	err = os.WriteFile(policyPath, []byte(updatedPolicy), 0644)
	require.NoError(t, err)
	
	// Wait for file watcher to detect the change
	time.Sleep(300 * time.Millisecond)
	
	// Test rate limiting
	for i := 0; i < 10; i++ {
		patch := interfaces.ConfigPatch{
			PatchID:             "test-patch-ratelimit-" + fmt.Sprintf("%d", i),
			TargetProcessorName: procID,
			ParameterPath:       "test_param_" + fmt.Sprintf("%d", i),
			NewValue:            i,
			Reason:              "Unit test rate limit",
			Severity:            "normal",
			Source:              "test",
			Timestamp:           time.Now().Unix(),
			TTLSeconds:          300,
		}
		
		err = picControl.SubmitConfigPatch(ctx, patch)
		if i >= cfg.MaxPatchesPerMinute {
			assert.Error(t, err, "Patch should be rate limited")
		}
	}
	
	// Test processor not found
	notFoundPatch := interfaces.ConfigPatch{
		PatchID:             "test-patch-not-found",
		TargetProcessorName: component.NewID("non_existent"),
		ParameterPath:       "test_param",
		NewValue:            42,
	}
	
	err = picControl.SubmitConfigPatch(ctx, notFoundPatch)
	assert.Error(t, err)
	
	// Shutdown the extension
	err = ext.Shutdown(ctx)
	require.NoError(t, err)
}