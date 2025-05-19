package control_chain

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
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	pic_connector "github.com/deepaucksharma/Phoenix/internal/connector/pic_connector"
	pic_control_ext "github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	adaptive_topk "github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
	priority_tagger "github.com/deepaucksharma/Phoenix/internal/processor/priority_tagger"
	"github.com/deepaucksharma/Phoenix/test/testutils"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type multiConsumer struct{ consumers []consumer.Metrics }

func (m multiConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (m multiConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	for _, c := range m.consumers {
		if err := c.ConsumeMetrics(ctx, md); err != nil {
			return err
		}
	}
	return nil
}

func TestControlChainPatchFlow(t *testing.T) {
	ctx := context.Background()
	host := testutils.NewTestHost()
	policyFile := filepath.Join(t.TempDir(), "policy.yaml")
	os.WriteFile(policyFile, []byte(
		"global_settings:\n"+
			"  autonomy_level: shadow\n"+
			"  collector_cpu_safety_limit_mcores: 100\n"+
			"  collector_rss_safety_limit_mib: 100\n"+
			"processors_config:\n"+
			"  priority_tagger:\n"+
			"    enabled: true\n"+
			"  adaptive_topk:\n"+
			"    enabled: true\n"+
			"    k_value: 30\n"+
			"  cardinality_guardian:\n"+
			"    enabled: true\n"+
			"    max_unique: 100\n"+
			"  reservoir_sampler:\n"+
			"    enabled: true\n"+
			"    reservoir_size: 10\n"+
			"  others_rollup:\n"+
			"    enabled: true\n"+
			"pid_decider_config:\n"+
			"  controllers: []\n"+
			"pic_control_config:\n"+
			"  policy_file_path: \"\"\n"+
			"  max_patches_per_minute: 5\n"+
			"  patch_cooldown_seconds: 0\n"+
			"  safe_mode_processor_configs: {}\n"), 0o600)

	// Create the pic_control extension
	extCfg := pic_control_ext.NewFactory().CreateDefaultConfig().(*pic_control_ext.Config)
	extCfg.PolicyFilePath = policyFile
	// Disable OpAMP client to avoid network calls during tests
	extCfg.OpAMPConfig = nil
	extSettings := extension.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewID(component.MustNewType("pic_control")),
	}
	extImpl, err := pic_control_ext.NewExtension(extCfg, zap.NewNop())
	require.NoError(t, err)
	require.NoError(t, extImpl.Start(ctx, host))
	defer extImpl.Shutdown(ctx)
	host.AddExtension(extSettings.ID, extImpl)

	// Create the pic_connector exporter
	connFactory := pic_connector.NewFactory()
	connCfg := connFactory.CreateDefaultConfig()
	connSettings := exporter.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewID(component.MustNewType("pic_connector")),
	}
	connector, err := connFactory.CreateMetrics(ctx, connSettings, connCfg)
	require.NoError(t, err)
	require.NoError(t, connector.Start(ctx, host))
	defer connector.Shutdown(ctx)

	// Sink to capture metrics after priority tagging
	sink := new(consumertest.MetricsSink)

	// Create priority_tagger processor
	tagFactory := priority_tagger.NewFactory()
	tagCfg := tagFactory.CreateDefaultConfig().(*priority_tagger.Config)
	tagSettings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewID(component.MustNewType("priority_tagger")),
	}
	fan := multiConsumer{consumers: []consumer.Metrics{connector, sink}}
	tagProc, err := tagFactory.CreateMetrics(ctx, tagSettings, tagCfg, fan)
	require.NoError(t, err)
	require.NoError(t, tagProc.Start(ctx, host))
	defer tagProc.Shutdown(ctx)

	// Create adaptive_topk processor
	topkFactory := adaptive_topk.NewFactory()
	topkCfg := topkFactory.CreateDefaultConfig().(*adaptive_topk.Config)
	topkCfg.BaseConfig.SetEnabled(false)
	topkSettings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		ID:                component.NewID(component.MustNewType("adaptive_topk")),
	}
	topkProc, err := topkFactory.CreateMetrics(ctx, topkSettings, topkCfg, tagProc)
	require.NoError(t, err)
	require.NoError(t, topkProc.Start(ctx, host))
	defer topkProc.Shutdown(ctx)

	// Register processor with extension via reflection
	field := reflect.ValueOf(extImpl).Elem().FieldByName("processors")
	ptr := unsafe.Pointer(field.UnsafeAddr())
	procMap := reflect.NewAt(field.Type(), ptr).Elem()
	procMap.SetMapIndex(reflect.ValueOf(topkSettings.ID), reflect.ValueOf(topkProc.(interfaces.UpdateableProcessor)))
	// Also register a generic "processor" ID to match pic_connector output
	genericID := component.NewID(component.MustNewType("processor"))
	procMap.SetMapIndex(reflect.ValueOf(genericID), reflect.ValueOf(topkProc.(interfaces.UpdateableProcessor)))

	// Prepare patch metric from adaptive_pid
	patch := interfaces.ConfigPatch{
		PatchID:             "patch1",
		TargetProcessorName: genericID,
		ParameterPath:       "enabled",
		NewValue:            true,
		Reason:              "test",
		Severity:            "normal",
		Source:              "adaptive_pid",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}
	patchMetrics := testutils.GeneratePatchMetric(patch)

	// Send patch metric directly via connector to avoid locking conflicts
	require.NoError(t, connector.ConsumeMetrics(ctx, patchMetrics))

	// Process regular metrics through the pipeline
	metrics := testutils.GenerateMetrics()
	err = topkProc.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	// Verify processor was enabled via patch
	status, err := topkProc.(interfaces.UpdateableProcessor).GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.True(t, status.Enabled)
}
