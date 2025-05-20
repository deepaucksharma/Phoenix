package pic_control_ext

import (
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"go.opentelemetry.io/collector/component"

	"github.com/deepaucksharma/Phoenix/pkg/policy"
)

// Expose internal methods for testing.

func (e *Extension) TestEnterSafeMode() error              { return e.enterSafeMode() }
func (e *Extension) TestExitSafeMode() error               { return e.exitSafeMode() }
func (e *Extension) TestLoadPolicyBytes(data []byte) error { return e.loadPolicyBytes(data) }

func (e *Extension) TestProcessors() map[component.ID]interfaces.UpdateableProcessor {
	return e.processors
}
func (e *Extension) TestPolicy() *policy.Policy                 { return e.policy }
func (e *Extension) TestPatchHistory() []interfaces.ConfigPatch { return e.patchHistory }
func (e *Extension) TestConfig() *Config                        { return e.config }
func (e *Extension) TestSafeMode() bool                         { return e.safeMode }
