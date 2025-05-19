package pic_control_ext_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/connector/pic_connector"
	pic_control_ext "github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// testHost implements component.Host for unit testing.
type testHost struct {
	exts map[component.ID]component.Component
}

func (h *testHost) GetExtensions() map[component.ID]component.Component { return h.exts }

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
