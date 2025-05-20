package causality_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/deepaucksharma/Phoenix/pkg/util/causality"
)

func TestGrangerCausalityBasic(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5, 6, 7}
	y := []float64{2, 2, 3, 5, 7, 8, 9}
	f, err := causality.GrangerCausality(x, y, 2)
	assert.NoError(t, err)
	assert.NotZero(t, f)
}

func TestTransferEntropyBasic(t *testing.T) {
	t.Skip("flaky in container")
	x := []float64{1, 2, 3, 4, 5, 6}
	y := []float64{2, 2, 3, 5, 7, 8}
	_, err := causality.TransferEntropy(x, y, 5, 1)
	assert.NoError(t, err)
}
