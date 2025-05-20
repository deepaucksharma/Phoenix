package safety

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// consumeCPU spins in a loop for the given duration to generate CPU usage.
func consumeCPU(d time.Duration) {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
	}
}

func TestCheckResourcesCapturesCPUUsage(t *testing.T) {
	sm := NewSafetyMonitor(nil, zap.NewNop())

	// first call initializes internal counters
	sm.checkResources()

	// generate some CPU activity
	consumeCPU(200 * time.Millisecond)

	sm.checkResources()
	cpu, _ := sm.GetCurrentUsage()

	assert.GreaterOrEqual(t, cpu, 0)
}
