// Package configpatch provides utilities for validating and applying configuration patches.
package configpatch

import (
	"fmt"
	"time"

	"github.com/yourorg/sa-omf/internal/interfaces"
)

// ValidationError represents an error that occurs during patch validation
type ValidationError struct {
	Message  string
	PatchID  string
	Severity string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("patch %s [%s]: %s", e.PatchID, e.Severity, e.Message)
}

// Validator defines the interface for config patch validation
type Validator interface {
	Validate(patch interfaces.ConfigPatch) error
}

// Options contains configurable options for patch validation
type Options struct {
	// MaxPatchesPerMinute limits the number of patches that can be applied in a minute
	MaxPatchesPerMinute int

	// PatchCooldownSeconds is the minimum time between patches to the same parameter
	PatchCooldownSeconds int

	// AllowedSeverityLevels is the list of allowed severity levels (nil means all)
	AllowedSeverityLevels []string

	// AllowedSources is the list of allowed patch sources (nil means all)
	AllowedSources []string
}

// DefaultOptions returns default validation options
func DefaultOptions() *Options {
	return &Options{
		MaxPatchesPerMinute:  5,
		PatchCooldownSeconds: 10,
	}
}

// StandardValidator implements standard patch validation logic
type StandardValidator struct {
	options        *Options
	patchHistory   []interfaces.ConfigPatch
	paramLastPatch map[string]time.Time  // Last patch time per parameter
	patchCount     int                   // Number of patches in the current minute
	lastMinute     time.Time             // The start of the current minute window
}

// NewStandardValidator creates a new standard validator with the given options
func NewStandardValidator(options *Options) *StandardValidator {
	if options == nil {
		options = DefaultOptions()
	}

	return &StandardValidator{
		options:        options,
		patchHistory:   make([]interfaces.ConfigPatch, 0, 100),
		paramLastPatch: make(map[string]time.Time),
		lastMinute:     time.Now(),
	}
}

// Validate checks if a patch is valid based on the configured options
func (v *StandardValidator) Validate(patch interfaces.ConfigPatch) error {
	// Check if patch has all required fields
	if patch.PatchID == "" {
		return &ValidationError{
			Message: "patch ID is required",
			PatchID: "unknown",
		}
	}

	if patch.TargetProcessorName.String() == "" {
		return &ValidationError{
			Message:  "target processor name is required",
			PatchID:  patch.PatchID,
			Severity: patch.Severity,
		}
	}

	if patch.ParameterPath == "" {
		return &ValidationError{
			Message:  "parameter path is required",
			PatchID:  patch.PatchID,
			Severity: patch.Severity,
		}
	}

	// Check for TTL expiration
	if patch.Timestamp > 0 && patch.TTLSeconds > 0 {
		expirationTime := time.Unix(patch.Timestamp, 0).Add(time.Duration(patch.TTLSeconds) * time.Second)
		if time.Now().After(expirationTime) {
			return &ValidationError{
				Message:  "patch has expired",
				PatchID:  patch.PatchID,
				Severity: patch.Severity,
			}
		}
	}

	// Check if source is allowed
	if len(v.options.AllowedSources) > 0 {
		sourceAllowed := false
		for _, source := range v.options.AllowedSources {
			if source == patch.Source {
				sourceAllowed = true
				break
			}
		}

		if !sourceAllowed {
			return &ValidationError{
				Message:  fmt.Sprintf("source '%s' is not allowed", patch.Source),
				PatchID:  patch.PatchID,
				Severity: patch.Severity,
			}
		}
	}

	// Check if severity is allowed
	if len(v.options.AllowedSeverityLevels) > 0 {
		severityAllowed := false
		for _, severity := range v.options.AllowedSeverityLevels {
			if severity == patch.Severity {
				severityAllowed = true
				break
			}
		}

		if !severityAllowed {
			return &ValidationError{
				Message:  fmt.Sprintf("severity '%s' is not allowed", patch.Severity),
				PatchID:  patch.PatchID,
				Severity: patch.Severity,
			}
		}
	}

	// Rate limiting
	now := time.Now()

	// Reset counter if minute boundary passed
	if now.Sub(v.lastMinute) > time.Minute {
		v.patchCount = 0
		v.lastMinute = now
	}

	// Check overall rate limit
	if v.patchCount >= v.options.MaxPatchesPerMinute {
		return &ValidationError{
			Message:  "rate limit exceeded: too many patches per minute",
			PatchID:  patch.PatchID,
			Severity: patch.Severity,
		}
	}

	// Check parameter cooldown
	paramKey := fmt.Sprintf("%s.%s", patch.TargetProcessorName, patch.ParameterPath)
	lastPatchTime, exists := v.paramLastPatch[paramKey]
	if exists {
		cooldownTime := time.Duration(v.options.PatchCooldownSeconds) * time.Second
		if now.Sub(lastPatchTime) < cooldownTime {
			return &ValidationError{
				Message:  fmt.Sprintf("parameter on cooldown for %d more seconds", int(cooldownTime.Seconds()-now.Sub(lastPatchTime).Seconds())),
				PatchID:  patch.PatchID,
				Severity: patch.Severity,
			}
		}
	}

	// If we got here, all validation passed
	// Update state for future validations
	v.patchCount++
	v.paramLastPatch[paramKey] = now

	// Add to history (with a cap)
	v.patchHistory = append(v.patchHistory, patch)
	if len(v.patchHistory) > 100 {
		// Keep the most recent 100 patches
		v.patchHistory = v.patchHistory[len(v.patchHistory)-100:]
	}

	return nil
}

// GetHistory returns the patch history (newest first)
func (v *StandardValidator) GetHistory() []interfaces.ConfigPatch {
	history := make([]interfaces.ConfigPatch, len(v.patchHistory))
	
	// Copy in reverse order (newest first)
	for i, j := 0, len(v.patchHistory)-1; j >= 0; i, j = i+1, j-1 {
		history[i] = v.patchHistory[j]
	}
	
	return history
}

// Reset clears the validator's internal state
func (v *StandardValidator) Reset() {
	v.patchHistory = make([]interfaces.ConfigPatch, 0, 100)
	v.paramLastPatch = make(map[string]time.Time)
	v.patchCount = 0
	v.lastMinute = time.Now()
}