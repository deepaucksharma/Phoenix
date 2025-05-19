package bayesian

import (
	"math"
	"testing"
)

func TestGaussianProcessPredict(t *testing.T) {
	gp := NewGaussianProcess(1.0, 1e-6)
	gp.AddSample([]float64{0}, 0)
	gp.AddSample([]float64{math.Pi / 2}, 1)
	gp.AddSample([]float64{math.Pi}, 0)

	mean, _ := gp.Predict([]float64{math.Pi / 4})
	if math.Abs(mean-math.Sqrt2/2) > 0.2 {
		t.Errorf("expected mean close to %v got %v", math.Sqrt2/2, mean)
	}
}
