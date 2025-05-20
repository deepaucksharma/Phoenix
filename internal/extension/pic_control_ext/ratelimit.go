package pic_control_ext

import (
	"time"
)

// checkRateLimit checks if the current patch rate is within limits
func (e *Extension) checkRateLimit() bool {
	// Reset counter if minute boundary passed
	now := time.Now()
	if now.Sub(e.patchCountReset) >= time.Minute {
		e.patchCountReset = now
		e.patchCount = 0
	}

	// Check if rate limit reached
	return e.patchCount < e.config.MaxPatchesPerMinute
}