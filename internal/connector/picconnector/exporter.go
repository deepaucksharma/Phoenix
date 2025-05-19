// Package picconnector implements an exporter that forwards configuration 
// patches from pid_decider to pic_control.
package picconnector

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/yourorg/sa-omf/internal/extension/piccontrolext"
	"github.com/yourorg/sa-omf/internal/interfaces"
)

// ensureInterface checks that the pic_connector implements the required interfaces.
var _ exporter.Metrics = (*picConnectorExporter)(nil)

// picConnectorExporter is the implementation of the pic_connector exporter.
type picConnectorExporter struct {
	logger     *zap.Logger
	picControl piccontrolext.PicControl
}

// newExporter creates a new pic_connector exporter.
func newExporter(set exporter.CreateSettings) (*picConnectorExporter, error) {
	return &picConnectorExporter{
		logger: set.Logger,
	}, nil
}

// Start implements the Component interface.
func (e *picConnectorExporter) Start(ctx context.Context, host component.Host) error {
	// Retrieve pic_control extension
	extensions := host.GetExtensions()
	for id, ext := range extensions {
		if id.Type() == "pic_control" {
			if pc, ok := ext.(piccontrolext.PicControl); ok {
				e.picControl = pc
				e.logger.Info("Successfully connected to pic_control extension")
				return nil
			}
		}
	}
	return fmt.Errorf("pic_control extension not found")
}

// Shutdown implements the Component interface.
func (e *picConnectorExporter) Shutdown(ctx context.Context) error {
	return nil
}

// Capabilities implements the exporter.Metrics interface.
func (e *picConnectorExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics implements the consumer.Metrics interface.
func (e *picConnectorExporter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	if e.picControl == nil {
		return fmt.Errorf("pic_control not initialized")
	}

	// Extract ConfigPatch objects from metrics
	patches := e.extractConfigPatches(md)
	if len(patches) == 0 {
		// No patches found, nothing to do
		return nil
	}

	// Submit each patch to pic_control
	for _, patch := range patches {
		err := e.picControl.SubmitConfigPatch(ctx, patch)
		if err != nil {
			// Log error but continue with other patches
			e.logger.Error("Failed to submit ConfigPatch",
				zap.String("patch_id", patch.PatchID),
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

// extractConfigPatches extracts ConfigPatch objects from OTLP metrics.
func (e *picConnectorExporter) extractConfigPatches(md pmetric.Metrics) []interfaces.ConfigPatch {
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

				// Handle gauge metrics
				if metric.Type() == pmetric.MetricTypeGauge {
					for l := 0; l < metric.Gauge().DataPoints().Len(); l++ {
						dp := metric.Gauge().DataPoints().At(l)
						patch := e.configPatchFromDataPoint(dp)
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

// configPatchFromDataPoint creates a ConfigPatch from metric data point attributes.
func (e *picConnectorExporter) configPatchFromDataPoint(dp pmetric.NumberDataPoint) *interfaces.ConfigPatch {
	// Extract required attributes
	var patchID, targetProcessorName, parameterPath, reason, severity, source string
	var timestamp int64
	var ttlSeconds int
	var newValue interface{}

	// Extract string attributes
	attrs := dp.Attributes()
	
	// Get PatchID - required
	if val, exists := attrs.Get("patch_id"); exists {
		patchID = val.AsString()
	} else {
		e.logger.Warn("ConfigPatch metric missing patch_id attribute")
		return nil
	}

	// Get TargetProcessorName - required
	if val, exists := attrs.Get("target_processor_name"); exists {
		targetProcessorName = val.AsString()
	} else {
		e.logger.Warn("ConfigPatch metric missing target_processor_name attribute",
			zap.String("patch_id", patchID))
		return nil
	}

	// Get ParameterPath - required
	if val, exists := attrs.Get("parameter_path"); exists {
		parameterPath = val.AsString()
	} else {
		e.logger.Warn("ConfigPatch metric missing parameter_path attribute",
			zap.String("patch_id", patchID))
		return nil
	}

	// Get Reason - optional
	if val, exists := attrs.Get("reason"); exists {
		reason = val.AsString()
	}

	// Get Severity - optional
	if val, exists := attrs.Get("severity"); exists {
		severity = val.AsString()
	} else {
		severity = "normal" // Default severity
	}

	// Get Source - optional
	if val, exists := attrs.Get("source"); exists {
		source = val.AsString()
	} else {
		source = "unknown" // Default source
	}

	// Get Timestamp - optional
	if val, exists := attrs.Get("timestamp"); exists {
		timestamp = val.AsInt()
	} else {
		// Use the datapoint timestamp if available
		timestamp = dp.Timestamp().AsTime().Unix()
	}

	// Get TTLSeconds - optional
	if val, exists := attrs.Get("ttl_seconds"); exists {
		ttlSeconds = int(val.AsInt())
	} else {
		ttlSeconds = 300 // Default 5 minutes TTL
	}

	// Extract new value based on type
	// Try different attribute names for different types
	if val, exists := attrs.Get("new_value_int"); exists {
		newValue = val.AsInt()
	} else if val, exists := attrs.Get("new_value_double"); exists {
		newValue = val.AsDouble()
	} else if val, exists := attrs.Get("new_value_string"); exists {
		newValue = val.AsString()
	} else if val, exists := attrs.Get("new_value_bool"); exists {
		newValue = val.AsBool()
	} else {
		e.logger.Warn("ConfigPatch metric missing new_value attribute",
			zap.String("patch_id", patchID))
		return nil
	}

	// Create and return the ConfigPatch
	return &interfaces.ConfigPatch{
		PatchID:             patchID,
		TargetProcessorName: component.MustNewIDFromString(targetProcessorName),
		ParameterPath:       parameterPath,
		NewValue:            newValue,
		Reason:              reason,
		Severity:            severity,
		Source:              source,
		Timestamp:           timestamp,
		TTLSeconds:          ttlSeconds,
	}
}