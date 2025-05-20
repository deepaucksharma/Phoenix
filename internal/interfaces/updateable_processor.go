// Package interfaces provides core interfaces for the Phoenix SA-OMF system.
package interfaces

import (
	"context"
	"time"
)

// ConfigPatch represents a requested change to a processor's configuration.
type ConfigPatch struct {
	// PatchID is a unique identifier for this configuration patch
	PatchID string `json:"patch_id,omitempty"`

	// TargetProcessorName is the name of the processor to be patched
	TargetProcessorName TargetID `json:"target_processor_name"`

	// ParameterPath is the path to the parameter to be changed
	ParameterPath string `json:"parameter_path"`

	// NewValue is the new value for the parameter
	NewValue interface{} `json:"new_value"`

	// Reason is a human-readable explanation for the patch
	Reason string `json:"reason,omitempty"`

	// Severity indicates the importance of this patch
	Severity string `json:"severity,omitempty"`

	// Source identifies what component generated this patch
	Source string `json:"source,omitempty"`

	// Timestamp is the Unix timestamp when the patch was created
	Timestamp int64 `json:"timestamp,omitempty"`

	// TTLSeconds is the number of seconds this patch is considered valid
	TTLSeconds int64 `json:"ttl_seconds,omitempty"`

	// SafetyOverride indicates whether this patch can override safety measures
	SafetyOverride bool `json:"safety_override,omitempty"`
}

// TargetID represents a component identifier
type TargetID interface {
	String() string
	Type() string
	Name() string
}

// ConfigStatus represents the current configuration status of a processor.
type ConfigStatus struct {
	// Parameters is a map of parameter names to their current values
	Parameters map[string]interface{} `json:"parameters"`

	// Enabled indicates whether the processor is currently enabled
	Enabled bool `json:"enabled"`
}

// UpdateableProcessor is an interface for processors that support dynamic
// configuration updates. These processors can receive configuration patches
// at runtime and adapt their behavior accordingly.
type UpdateableProcessor interface {
	// OnConfigPatch applies a configuration patch to the processor.
	// It returns an error if the patch could not be applied.
	OnConfigPatch(ctx context.Context, patch ConfigPatch) error

	// GetConfigStatus returns the current configuration status of the processor.
	GetConfigStatus(ctx context.Context) (ConfigStatus, error)

	// GetName returns a string identifier for the processor.
	GetName() string
}

// PicControl defines the interface for the policy-in-code control extension.
// This extension manages configuration patch application and policy enforcement.
type PicControl interface {
	// ApplyConfigPatch applies a configuration patch to a processor.
	// Returns an error if the patch cannot be applied.
	ApplyConfigPatch(ctx context.Context, patch ConfigPatch) error

	// RegisterUpdateableProcessor registers a processor that can have its
	// configuration updated dynamically.
	RegisterUpdateableProcessor(processor UpdateableProcessor) error

	// IsInSafeMode returns whether the control system is currently in safe mode.
	IsInSafeMode() bool

	// SetSafeMode sets the safe mode state.
	SetSafeMode(safeMode bool)

	// RegisterSafetyMonitor registers a safety monitor that can trigger safe mode.
	RegisterSafetyMonitor(monitor SafetyMonitor)
}

// SafetyMonitor defines the interface for components that monitor system
// resource usage and can trigger safe mode to prevent resource exhaustion.
type SafetyMonitor interface {
	// IsInSafeMode returns whether the system is currently in safe mode.
	IsInSafeMode() bool

	// GetSafeModeEnterTime returns when safe mode was entered, or nil if not in safe mode.
	GetSafeModeEnterTime() *time.Time

	// GetCurrentCPUThreshold returns the current CPU threshold in millicores.
	GetCurrentCPUThreshold() int64
}
