package bayesian

import (
	"math"
	"testing"
)

func TestOptimizerSuggest(t *testing.T) {
	opt := NewOptimizer([][2]float64{{0, 1}})
	opt.AddSample([]float64{0}, 0)
	opt.AddSample([]float64{1}, 0)
	opt.AddSample([]float64{0.5}, 1)

	p := opt.Suggest()
	if len(p) != 1 {
		t.Fatalf("unexpected dimension %d", len(p))
	}
	if diff := math.Abs(p[0] - 0.5); diff > 0.3 {
		t.Errorf("expected suggestion near 0.5 got %v", p[0])
	}
}
