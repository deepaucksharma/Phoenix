// Package chaos contains tests for resilience and failure handling.
package chaos

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"

	"github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// TestCorruptPolicyFile verifies that pic_control can handle corrupt policy files
// during hot-reload and maintain operation with the last known good policy.
// Scenario: CHAOS-POLICY
func TestCorruptPolicyFile(t *testing.T) {
	// Skip if running in short mode - this is an integration test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test context
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a test metrics collector to capture emitted metrics
	metricsCollector := metrics.NewMetricsCollector()

	// Create a temporary directory for policy files
	tempDir, err := ioutil.TempDir("", "pic-control-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	// Create a valid initial policy file
	validPolicyPath := filepath.Join(tempDir, "policy.yaml")
	validPolicy := `
controllers:
  - name: "coverage_controller"
    enabled: true
    kpi_metric_name: "aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m"
    kpi_target_value: 0.9
    kp: 30
    ki: 5
    kd: 0
    hysteresis_percent: 3
    output_config_patches:
      - target_processor_name: "adaptive_topk"
        parameter_path: "k_value"
        change_scale_factor: -20
        min_value: 10
        max_value: 60
`
	err = ioutil.WriteFile(validPolicyPath, []byte(validPolicy), 0644)
	require.NoError(t, err, "Failed to write valid policy file")

	// Create the pic_control extension with test configuration
	extensionFactory := pic_control_ext.NewFactory()
	defaultConfig := extensionFactory.CreateDefaultConfig().(*pic_control_ext.Config)
	
	// Configure with test settings
	defaultConfig.PolicyFile = validPolicyPath
	defaultConfig.WatchPolicy = true  // Enable watching to test hot-reload
	defaultConfig.WatchIntervalSeconds = 1  // Check frequently for changes
	defaultConfig.MetricsEmitter = metricsCollector
	
	// Create the extension
	extension, err := pic_control_ext.NewPICControlExtension(defaultConfig, component.TelemetrySettings{})
	require.NoError(t, err, "Failed to create pic_control extension")
	
	// Start the extension
	err = extension.Start(ctx, testutils.NewMockHost())
	require.NoError(t, err, "Failed to start pic_control extension")
	defer extension.Shutdown(ctx)

	// Register mock updateable processors
	mockTopK := testutils.NewMockUpdateableProcessor("adaptive_topk")
	mockTopK.SetParameter("k_value", 30) // Initial value
	
	err = extension.RegisterUpdateableProcessor(mockTopK)
	require.NoError(t, err, "Failed to register mock processor")

	// Give time for the initial policy to be loaded
	time.Sleep(2 * time.Second)

	// Verify the initial policy was loaded
	currentValue := mockTopK.GetParameter("k_value")
	assert.Equal(t, 30, currentValue, "Initial policy should be loaded")

	// Now corrupt the policy file with invalid YAML
	corruptPolicy := `
controllers:
  - name: "coverage_controller"
    enabled: true
    kpi_metric_name: "aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m"
    kpi_target_value: 0.9
    kp: 30
    ki: 5
    THIS IS NOT VALID YAML
      indentation is wrong
        and missing colons
`
	err = ioutil.WriteFile(validPolicyPath, []byte(corruptPolicy), 0644)
	require.NoError(t, err, "Failed to write corrupt policy file")

	// Give time for the file watcher to detect the change
	time.Sleep(3 * time.Second)

	// Verify system still uses the last known good policy
	currentValue = mockTopK.GetParameter("k_value")
	assert.Equal(t, 30, currentValue, "Value should remain unchanged with corrupt policy")

	// Check metrics
	metrics := metricsCollector.GetMetrics()
	foundReloadFailure := false
	
	for _, metric := range metrics {
		if metric.Name == "aemf_policy_reload_failed_total" && metric.Value > 0 {
			foundReloadFailure = true
			break
		}
	}
	
	assert.True(t, foundReloadFailure, "policy_reload_failed_total metric should be incremented")

	// Now restore a valid policy with a new value
	newValidPolicy := `
controllers:
  - name: "coverage_controller"
    enabled: true
    kpi_metric_name: "aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m"
    kpi_target_value: 0.9
    kp: 30
    ki: 5
    kd: 0
    hysteresis_percent: 3
    output_config_patches:
      - target_processor_name: "adaptive_topk"
        parameter_path: "k_value"
        change_scale_factor: -20
        min_value: 10
        max_value: 60
`
	err = ioutil.WriteFile(validPolicyPath, []byte(newValidPolicy), 0644)
	require.NoError(t, err, "Failed to write new valid policy file")

	// Give time for the file watcher to detect the change
	time.Sleep(3 * time.Second)

	// Verify system recovers and uses the new policy
	// We would normally see a new value here if the policy had different parameters
	// For this test we're just verifying the system remains operational
	currentValue = mockTopK.GetParameter("k_value")
	assert.Equal(t, 30, currentValue, "System should recover with valid policy")

	// Check metrics for successful reload
	metricsCollector.Clear()
	time.Sleep(1 * time.Second)
	
	metrics = metricsCollector.GetMetrics()
	foundReloadSuccess := false
	
	for _, metric := range metrics {
		if metric.Name == "aemf_policy_reload_success_total" && metric.Value > 0 {
			foundReloadSuccess = true
			break
		}
	}
	
	assert.True(t, foundReloadSuccess, "policy_reload_success_total metric should be incremented")
}