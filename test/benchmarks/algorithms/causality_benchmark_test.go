package algorithms

import (
	"testing"

	"github.com/deepaucksharma/Phoenix/pkg/util/causality"
)

func BenchmarkGranger(b *testing.B) {
	x := make([]float64, 200)
	y := make([]float64, 200)
	for i := 0; i < 200; i++ {
		x[i] = float64(i)
		y[i] = float64(i) + 1
	}
	for i := 0; i < b.N; i++ {
		_, _ = causality.GrangerCausality(x, y, 2)
	}
}
