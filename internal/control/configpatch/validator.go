package configpatch

import (
	"sync"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

type Options struct {
	AllowedSeverityLevels []string
	AllowedSources        []string
	MaxPatchesPerMinute   int
	PatchCooldownSeconds  int
}

type StandardValidator struct {
	mu      sync.Mutex
	history []interfaces.ConfigPatch
	opts    *Options
}

func NewStandardValidator(opts *Options) *StandardValidator {
	return &StandardValidator{opts: opts}
}

func (v *StandardValidator) Validate(patch interfaces.ConfigPatch) error { return nil }
func (v *StandardValidator) GetHistory() []interfaces.ConfigPatch        { return v.history }
func (v *StandardValidator) Reset()                                      { v.history = nil }
