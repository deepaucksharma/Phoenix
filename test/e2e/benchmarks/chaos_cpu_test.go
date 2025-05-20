// Scenario: CHAOS-CPU
package benchmark

import (
	"os/exec"
	"testing"

	"github.com/deepaucksharma/Phoenix/test/testutils"
)

func TestChaosCPU(t *testing.T) {
	t.Skip("Scenario temporarily disabled")

	_ = testutils.GenerateTestMetrics(1)

	cmd := exec.Command("go", "run", "test/generator/workload.go", "--processes", "100", "--spike-freq", "1.0", "--duration", "1s")
	_ = cmd
}
