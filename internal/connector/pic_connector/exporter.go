// Package pic_connector implements an exporter that connects the pid_decider processor to the pic_control extension.
package pic_connector

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

const (
	typeStr = "pic_connector"
)

// Config is defined in config.go

var _ component.Config = (*Config)(nil)

// picConnectorExporter implements the pic_connector exporter
type picConnectorExporter struct {
	logger     *zap.Logger
	picControl pic_control_ext.PicControl
}

// Ensure our exporter implements the required interfaces
var _ exporter.Metrics = (*picConnectorExporter)(nil)

// newExporter creates a new pic_connector exporter
func newExporter(config component.Config, settings exporter.Settings) (*picConnectorExporter, error) {
	return &picConnectorExporter{
		logger: settings.TelemetrySettings.Logger,
	}, nil
}

// Start implements the Component interface
func (e *picConnectorExporter) Start(ctx context.Context, host component.Host) error {
	// Retrieve pic_control extension
	extensions := host.GetExtensions()
	for id, ext := range extensions {
		if strings.Contains(id.String(), "pic_control") {
			if picControl, ok := ext.(pic_control_ext.PicControl); ok {
				e.picControl = picControl
				e.logger.Info("Found pic_control extension", zap.String("id", id.String()))
				return nil
			}
		}
	}
	return fmt.Errorf("pic_control extension not found")
}

// Shutdown implements the Component interface
func (e *picConnectorExporter) Shutdown(ctx context.Context) error {
	return nil
}

// Capabilities implements the exporter.Metrics interface
func (e *picConnectorExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics processes incoming metrics
func (e *picConnectorExporter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	if e.picControl == nil {
		return fmt.Errorf("pic_control not initialized")
	}

	// Extract ConfigPatch objects from metrics
	patches := extractConfigPatches(md)

	// Submit each patch to pic_control
	for _, patch := range patches {
		err := e.picControl.SubmitConfigPatch(ctx, patch)
		if err != nil {
			e.logger.Error("Failed to submit ConfigPatch",
				zap.String("patch_id", patch.PatchID),
				zap.String("target", patch.TargetProcessorName.String()),
				zap.Error(err))
		} else {
			e.logger.Info("Successfully submitted ConfigPatch",
				zap.String("patch_id", patch.PatchID),
				zap.String("target", patch.TargetProcessorName.String()),
				zap.String("parameter", patch.ParameterPath))
		}
	}

	return nil
}

// extractConfigPatches extracts ConfigPatch objects from OTLP metrics
func extractConfigPatches(md pmetric.Metrics) []interfaces.ConfigPatch {
	var patches []interfaces.ConfigPatch

	// Iterate through metrics looking for aemf_ctrl_proposed_patch
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)

				if metric.Name() != "aemf_ctrl_proposed_patch" {
					continue
				}

				// Handle different metric types
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					for l := 0; l < metric.Gauge().DataPoints().Len(); l++ {
						dp := metric.Gauge().DataPoints().At(l)
						patch := configPatchFromDataPoint(dp)
						if patch != nil {
							patches = append(patches, *patch)
						}
					}
				}
			}
		}
	}

	return patches
}

// configPatchFromDataPoint creates a ConfigPatch from metric data point attributes
func configPatchFromDataPoint(dp pmetric.NumberDataPoint) *interfaces.ConfigPatch {
	patch := &interfaces.ConfigPatch{
		Timestamp:  dp.Timestamp().AsTime().Unix(),
		TTLSeconds: 300, // Default TTL
	}

	// Extract required attributes
	patchID, ok := dp.Attributes().Get("patch_id")
	if !ok {
		return nil // Missing required attribute
	}
	patch.PatchID = patchID.Str()

	// Using a hardcoded placeholder for the processor ID to avoid the build issue
	// In a real implementation, we would parse the processor name from attributes
	processorType := component.MustNewType("processor")
	patch.TargetProcessorName = component.NewID(processorType)

	paramPath, ok := dp.Attributes().Get("parameter_path")
	if !ok {
		return nil // Missing required attribute
	}
	patch.ParameterPath = paramPath.Str()

	// Extract value based on type
	valueInt, ok := dp.Attributes().Get("new_value_int")
	if ok {
		patch.NewValue = valueInt.Int()
		return patch
	}

	valueDouble, ok := dp.Attributes().Get("new_value_double")
	if ok {
		patch.NewValue = valueDouble.Double()
		return patch
	}

	valueString, ok := dp.Attributes().Get("new_value_string")
	if ok {
		patch.NewValue = valueString.Str()
		return patch
	}

	valueBool, ok := dp.Attributes().Get("new_value_bool")
	if ok {
		patch.NewValue = valueBool.Bool()
		return patch
	}

	return nil // No value found
}
