// Scenario: PIPE-03
package e2e

import (
	"testing"

	"github.com/deepaucksharma/Phoenix/test/testutils"
)

func TestPipe03(t *testing.T) {
	t.Skip("Scenario temporarily disabled")

	_ = testutils.GenerateHighCardinalityMetrics(5)
}
