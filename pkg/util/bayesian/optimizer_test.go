package bayesian

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
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

// TestLatinHypercubeSampling tests Latin Hypercube Sampling
func TestLatinHypercubeSampling(t *testing.T) {
	bounds := [][2]float64{{0, 1}, {-1, 1}}
	n := 20
	rng := NewOptimizer(bounds).rng

	samples := generateLatinHypercubeSamples(n, bounds, rng)

	// Check number of samples
	assert.Equal(t, n, len(samples), "Should generate n samples")

	// Check that each sample has the right dimensionality
	for _, sample := range samples {
		assert.Equal(t, len(bounds), len(sample), "Each sample should have correct dimensions")
	}

	// Check that samples are within bounds
	for _, sample := range samples {
		for j, bound := range bounds {
			assert.GreaterOrEqual(t, sample[j], bound[0], "Sample should be within lower bound")
			assert.LessOrEqual(t, sample[j], bound[1], "Sample should be within upper bound")
		}
	}
}

// TestOptimizerInitialSampling tests the initial sampling phase
func TestOptimizerInitialSampling(t *testing.T) {
	bounds := [][2]float64{{0, 10}, {-5, 5}}
	optimizer := NewOptimizer(bounds)

	// First sample should be at the midpoint of bounds
	firstPoint := optimizer.Suggest()
	assert.InDelta(t, 5.0, firstPoint[0], 0.001, "First x coordinate should be midpoint of bounds")
	assert.InDelta(t, 0.0, firstPoint[1], 0.001, "First y coordinate should be midpoint of bounds")

	// Add the first sample
	optimizer.AddSample(firstPoint, 0.5)

	// Next dim samples should be near corners
	for i := 0; i < len(bounds); i++ {
		point := optimizer.Suggest()
		optimizer.AddSample(point, float64(i))
	}

	// After initial sampling, should switch to actual optimization
	optimizer.AddSample([]float64{3, 1}, 2.0)
	optimizer.AddSample([]float64{7, -2}, 3.0)
	optimizer.AddSample([]float64{2, 2}, 4.0)

	// Now should be using EI
	nextPoint := optimizer.Suggest()
	assert.NotNil(t, nextPoint, "Should suggest a valid point")
}

// TestExpectedImprovementWithExploration tests the EI acquisition function
func TestExpectedImprovementWithExploration(t *testing.T) {
	// Test basic EI
	ei0 := expectedImprovement(5.0, 1.0, 3.0)
	assert.Greater(t, ei0, 0.0, "EI should be positive when mean > best")

	// Test with exploration parameter
	ei1 := expectedImprovementWithExploration(5.0, 1.0, 3.0, 0.1)
	assert.Greater(t, ei0, ei1, "EI with exploration should be lower due to penalty")

	// Test EI with uncertain regions
	eiUncertain := expectedImprovementWithExploration(3.0, 3.0, 3.0, 0.1)
	eiCertain := expectedImprovementWithExploration(3.0, 0.1, 3.0, 0.1)
	assert.Greater(t, eiUncertain, eiCertain, "Higher variance should increase EI")
}

// TestOptimizerConfigure tests the configuration options
func TestOptimizerConfigure(t *testing.T) {
	bounds := [][2]float64{{0, 1}, {0, 1}}
	optimizer := NewOptimizer(bounds)

	// Test changing hyperparameters
	optimizer.ConfigureOptimizer(200, 0.05, 1e-3)

	// Test length scale configuration
	err := optimizer.SetLengthScales([]float64{0.2, 0.3})
	assert.NoError(t, err, "Setting valid length scales should not error")

	// Test dimension mismatch
	err = optimizer.SetLengthScales([]float64{0.2, 0.3, 0.4})
	assert.Error(t, err, "Setting wrong dimension length scales should error")
}

// TestOptimizerGetBestSolution tests retrieving the best solution
func TestOptimizerGetBestSolution(t *testing.T) {
	bounds := [][2]float64{{0, 1}, {0, 1}}
	optimizer := NewOptimizer(bounds)

	// Add some samples
	optimizer.AddSample([]float64{0.2, 0.3}, 1.0)
	optimizer.AddSample([]float64{0.5, 0.5}, 2.0)
	optimizer.AddSample([]float64{0.8, 0.7}, 1.5)

	// Get best solution
	bestX, bestY := optimizer.GetBestSolution()

	// Should be the second sample
	assert.Equal(t, 2.0, bestY, "Best Y should be the highest observed value")
	assert.InDeltaSlice(t, []float64{0.5, 0.5}, bestX, 0.0001, "Best X should match the point with highest value")
}

// TestOptimizerExploration tests the exploration-exploitation tradeoff
func TestOptimizerExploration(t *testing.T) {
	// Simple 1D optimization
	bounds := [][2]float64{{0, 10}}

	// Create two optimizers - one with high exploration, one with low
	exploringOptimizer := NewOptimizer(bounds)
	exploringOptimizer.ConfigureOptimizer(100, 0.5, 1e-5) // High exploration

	exploitingOptimizer := NewOptimizer(bounds)
	exploitingOptimizer.ConfigureOptimizer(100, 0.01, 1e-5) // Low exploration

	// Add the same samples to both
	for _, x := range []float64{1.0, 2.0, 3.0} {
		// Simple quadratic function with maximum at x=5
		y := -math.Pow(x-5, 2) + 5

		exploringOptimizer.AddSample([]float64{x}, y)
		exploitingOptimizer.AddSample([]float64{x}, y)
	}

	// Exploiting optimizer should suggest closer to observed maximum (x=3)
	nextExploiting := exploitingOptimizer.Suggest()[0]

	// Exploring optimizer should suggest points farther from observed data
	nextExploring := exploringOptimizer.Suggest()[0]

	// Distance from exploiting point to observed maximum (x=3)
	distExploiting := math.Abs(nextExploiting - 3.0)

	// Distance from exploring point to observed maximum (x=3)
	distExploring := math.Abs(nextExploring - 3.0)

	// Exploring optimizer should suggest points farther from observed maximum
	assert.Greater(t, distExploring, distExploiting,
		"Exploring optimizer should suggest points farther from observed data")
}
