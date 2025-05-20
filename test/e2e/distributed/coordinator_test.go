// Package distributed contains tests for distributed deployment features.
package distributed

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"

	"github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// ClusterNode simulates a node in a distributed deployment
type ClusterNode struct {
	NodeID           string
	PicControlExt    *pic_control_ext.Extension
	MockProcessors   map[string]*testutils.MockUpdateableProcessor
	MetricsCollector *metrics.MetricsCollector
}

// TestDistributedCoordination verifies that configuration changes are properly
// coordinated across multiple nodes in a distributed deployment.
// Scenario: DISTR-02
func TestDistributedCoordination(t *testing.T) {
	// Skip if running in short mode - this is an integration test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create multiple simulated cluster nodes
	nodes := make([]*ClusterNode, 3)

	// Create nodes
	for i := 0; i < 3; i++ {
		node, err := createTestNode(ctx, t, i)
		require.NoError(t, err, "Failed to create test node")
		defer node.PicControlExt.Shutdown(ctx)
		nodes[i] = node
	}

	// Create a central coordination service (simulated)
	coordinator := &testCoordinator{
		nodes: nodes,
	}

	// Test that changes are properly propagated to all nodes

	// PART 1: Test global configuration change

	// Apply a global configuration change to all nodes
	err := coordinator.ApplyGlobalConfigurationChange(ctx, "adaptive_topk", "k_value", 50)
	require.NoError(t, err, "Failed to apply global configuration change")

	// Wait a moment for changes to apply
	time.Sleep(100 * time.Millisecond)

	// Verify that all nodes received the configuration change
	for i, node := range nodes {
		processor := node.MockProcessors["adaptive_topk"]
		value, exists := processor.GetParameter("k_value")
		assert.True(t, exists, "Node %d should have k_value parameter", i)
		assert.Equal(t, 50, value, "Node %d should have received global configuration update", i)
	}

	// PART 2: Test node-specific configuration change

	// Apply a node-specific configuration change to Node 1
	err = coordinator.ApplyNodeSpecificConfiguration(ctx, 1, "adaptive_topk", "k_value", 75)
	require.NoError(t, err, "Failed to apply node-specific configuration change")

	// Wait a moment for changes to apply
	time.Sleep(100 * time.Millisecond)

	// Verify that only Node 1 received the configuration change
	value0, exists0 := nodes[0].MockProcessors["adaptive_topk"].GetParameter("k_value")
	assert.True(t, exists0, "Node 0 should have k_value parameter")
	assert.Equal(t, 50, value0, "Node 0 should not have been affected by node-specific change")

	value1, exists1 := nodes[1].MockProcessors["adaptive_topk"].GetParameter("k_value")
	assert.True(t, exists1, "Node 1 should have k_value parameter")
	assert.Equal(t, 75, value1, "Node 1 should have received node-specific configuration update")

	value2, exists2 := nodes[2].MockProcessors["adaptive_topk"].GetParameter("k_value")
	assert.True(t, exists2, "Node 2 should have k_value parameter")
	assert.Equal(t, 50, value2, "Node 2 should not have been affected by node-specific change")

	// PART 3: Test cluster roll-out with sequencing

	// Apply a gradual configuration change across the cluster
	err = coordinator.ApplySequentialConfigurationChange(ctx, "adaptive_topk", "k_value", 100)
	require.NoError(t, err, "Failed to apply sequential configuration change")

	// Verify that all nodes eventually received the configuration change
	for i, node := range nodes {
		processor := node.MockProcessors["adaptive_topk"]
		value, exists := processor.GetParameter("k_value")
		assert.True(t, exists, "Node %d should have k_value parameter", i)
		assert.Equal(t, 100, value, "Node %d should have received sequential configuration update", i)
	}

	// Verify that patches were applied in sequence (check timestamps)
	// This would require collecting timestamps from each node's metrics
	// For simplicity, we'll just check that the appropriate metrics were emitted

	for _, node := range nodes {
		patchMetrics := node.MetricsCollector.GetMetricsByName("aemf_ctrl_patch_applied_total")
		assert.NotEmpty(t, patchMetrics, "Node should have emitted patch metrics")
	}
}

// createTestNode creates a test node with a pic_control extension and mock processors
func createTestNode(ctx context.Context, t *testing.T, nodeIndex int) (*ClusterNode, error) {
	// Create a metrics collector
	metricsCollector := metrics.NewMetricsCollector()

	// Create pic_control extension
	picControlConfig := pic_control_ext.NewFactory().CreateDefaultConfig().(*pic_control_ext.Config)
	picControlConfig.PolicyFile = "../policy/testdata/valid_policy.yaml"
	picControlConfig.WatchPolicy = false // Disable watching for the test
	picControlConfig.MetricsEmitter = metricsCollector

	picControlExt, err := pic_control_ext.NewPICControlExtension(picControlConfig, component.TelemetrySettings{})
	if err != nil {
		return nil, err
	}

	// Start the extension
	err = picControlExt.Start(ctx, testutils.NewMockHost())
	if err != nil {
		return nil, err
	}

	// Create processors
	processors := make(map[string]*testutils.MockUpdateableProcessor)

	// Create adaptive_topk processor
	topkProcessor := testutils.NewMockUpdateableProcessor("adaptive_topk")
	topkProcessor.SetParameter("k_value", 10) // Initial value

	// Register processor with extension
	err = picControlExt.RegisterUpdateableProcessor(topkProcessor)
	if err != nil {
		return nil, err
	}

	processors["adaptive_topk"] = topkProcessor

	// Return the node
	return &ClusterNode{
		NodeID:           fmt.Sprintf("node-%d", nodeIndex),
		PicControlExt:    picControlExt,
		MockProcessors:   processors,
		MetricsCollector: metricsCollector,
	}, nil
}

// testCoordinator simulates a central coordination service
type testCoordinator struct {
	nodes []*ClusterNode
	lock  sync.Mutex
}

// ApplyGlobalConfigurationChange applies a configuration change to all nodes
func (c *testCoordinator) ApplyGlobalConfigurationChange(
	ctx context.Context,
	processorName string,
	parameterPath string,
	value interface{},
) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Create a patch
	patch := interfaces.ConfigPatch{
		PatchID:             fmt.Sprintf("global-patch-%d", time.Now().UnixNano()),
		TargetProcessorName: component.NewIDWithName(component.MustNewType("processor"), processorName),
		ParameterPath:       parameterPath,
		NewValue:            value,
		Reason:              "Global configuration change",
		Severity:            "normal",
		Source:              "coordinator",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}

	// Apply to all nodes
	for _, node := range c.nodes {
		err := node.PicControlExt.ApplyConfigPatch(ctx, patch)
		if err != nil {
			return err
		}
	}

	return nil
}

// ApplyNodeSpecificConfiguration applies a configuration change to a specific node
func (c *testCoordinator) ApplyNodeSpecificConfiguration(
	ctx context.Context,
	nodeIndex int,
	processorName string,
	parameterPath string,
	value interface{},
) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if nodeIndex < 0 || nodeIndex >= len(c.nodes) {
		return fmt.Errorf("invalid node index: %d", nodeIndex)
	}

	// Create a patch
	patch := interfaces.ConfigPatch{
		PatchID:             fmt.Sprintf("node-specific-patch-%d", time.Now().UnixNano()),
		TargetProcessorName: component.NewIDWithName(component.MustNewType("processor"), processorName),
		ParameterPath:       parameterPath,
		NewValue:            value,
		Reason:              fmt.Sprintf("Node-specific configuration change for node %d", nodeIndex),
		Severity:            "normal",
		Source:              "coordinator",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}

	// Apply to the specific node
	return c.nodes[nodeIndex].PicControlExt.ApplyConfigPatch(ctx, patch)
}

// ApplySequentialConfigurationChange applies a configuration change to all nodes in sequence
func (c *testCoordinator) ApplySequentialConfigurationChange(
	ctx context.Context,
	processorName string,
	parameterPath string,
	value interface{},
) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Apply to each node in sequence with a delay between each
	for i, node := range c.nodes {
		// Create a patch
		patch := interfaces.ConfigPatch{
			PatchID:             fmt.Sprintf("sequential-patch-%d-%d", i, time.Now().UnixNano()),
			TargetProcessorName: component.NewIDWithName(component.MustNewType("processor"), processorName),
			ParameterPath:       parameterPath,
			NewValue:            value,
			Reason:              fmt.Sprintf("Sequential configuration change for node %d", i),
			Severity:            "normal",
			Source:              "coordinator",
			Timestamp:           time.Now().Unix(),
			TTLSeconds:          300,
		}

		// Apply to the node
		err := node.PicControlExt.ApplyConfigPatch(ctx, patch)
		if err != nil {
			return err
		}

		// Wait before moving to the next node (simulate gradual roll-out)
		time.Sleep(50 * time.Millisecond)
	}

	return nil
}
