// Package interfaces defines the core interfaces for updateable processors in the SA-OMF.
package interfaces

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

// ConfigPatch defines a proposed change to a processor's configuration
type ConfigPatch struct {
	PatchID             string       `json:"patch_id"`              // Unique ID for this patch attempt
	TargetProcessorName component.ID  `json:"target_processor_name"` // Name of the processor to update
	ParameterPath       string       `json:"parameter_path"`        // Dot-separated path to the parameter
	NewValue            any          `json:"new_value"`             // The new value for the parameter
	PrevValue           any          `json:"prev_value"`            // Previous value (for rollback)
	Reason              string       `json:"reason"`                // Why this patch is proposed
	Severity            string       `json:"severity"`              // normal|urgent|safety
	Source              string       `json:"source"`                // pid_decider|opamp|manual
	Timestamp           int64        `json:"timestamp"`             // When this patch was created
	TTLSeconds          int          `json:"ttl_seconds"`           // Time-to-live for this patch
}

// ConfigStatus provides current operational parameters of an UpdateableProcessor
type ConfigStatus struct {
	Parameters map[string]any `json:"parameters"` // Current values of tunable parameters
	Enabled    bool           `json:"enabled"`    // Whether the processor is currently enabled
}

// UpdateableProcessor defines the interface for processors that can be dynamically reconfigured
type UpdateableProcessor interface {
	component.Component // Embed standard component interface

	// OnConfigPatch applies a configuration change.
	// Returns error if patch cannot be applied.
	OnConfigPatch(ctx context.Context, patch ConfigPatch) error

	// GetConfigStatus returns the current effective configuration.
	GetConfigStatus(ctx context.Context) (ConfigStatus, error)
}
