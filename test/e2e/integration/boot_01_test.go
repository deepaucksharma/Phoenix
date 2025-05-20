// Scenario: BOOT-01
package e2e

import (
	"os/exec"
	"testing"

	"github.com/deepaucksharma/Phoenix/test/testutils"
)

func TestBoot01(t *testing.T) {
	t.Skip("Scenario temporarily disabled")

	_ = testutils.GenerateMetrics()

	cmd := exec.Command("go", "run", "test/generator/workload.go", "--duration", "1s")
	_ = cmd
}
