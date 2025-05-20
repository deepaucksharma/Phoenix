package bayesian

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
)

// Optimizer performs basic Bayesian optimization using a Gaussian process
// and the Expected Improvement acquisition function.
type Optimizer struct {
	gp         *GaussianProcess
	bounds     [][2]float64
	candidates int
	rng        *rand.Rand
	bestY      float64
	bestX      []float64
	samples    int         // Number of samples collected
	
	// Hyperparameters
	explorationWeight float64   // Weight for exploration vs. exploitation (xi in EI)
	lenScales        []float64  // Length scales for each dimension
	noiseLevel       float64    // Observation noise level
	lock             sync.Mutex // For thread safety
}

// NewOptimizer creates a new optimizer for the given bounds.
func NewOptimizer(bounds [][2]float64) *Optimizer {
	dim := len(bounds)
	
	// Create dimension-specific length scales
	lenScales := make([]float64, dim)
	for i, bound := range bounds {
		// Default length scale is 10% of the parameter range
		lenScales[i] = (bound[1] - bound[0]) * 0.1
	}
	
	return &Optimizer{
		gp:                NewGaussianProcess(1.0, 1e-6),
		bounds:           bounds,
		candidates:       100,         // More candidates for better exploration
		rng:              rand.New(rand.NewSource(time.Now().UnixNano())),
		bestY:            math.Inf(-1),
		bestX:            make([]float64, dim),
		samples:          0,
		explorationWeight: 0.01,       // Small value for more exploitation focus
		lenScales:         lenScales,   // Dimension-specific length scales
		noiseLevel:        1e-5,        // Small noise level for numerical stability
	}
}

// ConfigureOptimizer sets optimizer hyperparameters
func (o *Optimizer) ConfigureOptimizer(candidates int, explorationWeight float64, noiseLevel float64) {
	o.lock.Lock()
	defer o.lock.Unlock()
	
	if candidates > 0 {
		o.candidates = candidates
	}
	
	if explorationWeight > 0 {
		o.explorationWeight = explorationWeight
	}
	
	if noiseLevel > 0 {
		o.noiseLevel = noiseLevel
		// Update GP noise level
		o.gp.SetNoise(noiseLevel)
	}
}

// SetLengthScales sets custom length scales for each dimension
func (o *Optimizer) SetLengthScales(lenScales []float64) error {
	o.lock.Lock()
	defer o.lock.Unlock()
	
	if len(lenScales) != len(o.bounds) {
		return fmt.Errorf("length scales dimension mismatch: got %d, expected %d", 
		                 len(lenScales), len(o.bounds))
	}
	
	// Copy the length scales
	for i, scale := range lenScales {
		if scale <= 0 {
			return fmt.Errorf("length scale must be positive: dimension %d has value %f", i, scale)
		}
		o.lenScales[i] = scale
	}
	
	// Update GP with new kernel parameters
	o.gp.SetLengthScales(o.lenScales)
	
	return nil
}

// AddSample records the observation of value y at position x.
func (o *Optimizer) AddSample(x []float64, y float64) {
	o.lock.Lock()
	defer o.lock.Unlock()
	
	o.gp.AddSample(x, y)
	o.samples++
	
	if y > o.bestY {
		o.bestY = y
		o.bestX = append([]float64{}, x...)
	}
	
	// Adapt exploration weight as more samples are collected
	// Start with more exploration, then focus more on exploitation
	if o.samples > 10 {
		o.explorationWeight = math.Max(0.005, o.explorationWeight * 0.95)
	}
}

// Suggest returns the next point to evaluate based on expected improvement.
func (o *Optimizer) Suggest() []float64 {
	o.lock.Lock()
	defer o.lock.Unlock()
	
	dim := len(o.bounds)
	
	// If no samples yet, start with latin hypercube sampling for initial points
	if len(o.gp.x) == 0 {
		// Return midpoint of bounds for first sample
		mid := make([]float64, dim)
		for i, b := range o.bounds {
			mid[i] = (b[0] + b[1]) / 2
		}
		return mid
	} else if len(o.gp.x) < dim + 1 {
		// For first dim+1 samples, do quasi-random sampling with Sobol sequence
		// For simplicity, we'll use a basic corner-sampling approach
		point := make([]float64, dim)
		for j, b := range o.bounds {
			// Use binary encoding of number of samples to select corners
			if (len(o.gp.x) & (1 << j)) != 0 {
				point[j] = b[1] // Upper bound
			} else {
				point[j] = b[0] // Lower bound
			}
		}
		return point
	}

	// Use Latin Hypercube Sampling for candidates to ensure good coverage
	candidates := generateLatinHypercubeSamples(o.candidates, o.bounds, o.rng)
	
	bestEI := -math.MaxFloat64
	bestPoint := make([]float64, dim)
	
	// Find point with best expected improvement
	for _, p := range candidates {
		mean, variance := o.gp.Predict(p)
		ei := expectedImprovementWithExploration(mean, math.Sqrt(variance), o.bestY, o.explorationWeight)
		if ei > bestEI {
			bestEI = ei
			copy(bestPoint, p)
		}
	}
	
	return bestPoint
}

// expectedImprovementWithExploration calculates the expected improvement with exploration parameter
func expectedImprovementWithExploration(mean, std, best, xi float64) float64 {
	if std <= 0 {
		return 0
	}
	
	// Improvement term with exploration factor xi
	improvement := mean - best - xi
	
	z := improvement / std
	normal := distuv.UnitNormal
	
	// Expected improvement formula: E[max(0, I)]
	return improvement*normal.CDF(z) + std*normal.Prob(z)
}

// expectedImprovement is the classic expected improvement without exploration param
func expectedImprovement(mean, std, best float64) float64 {
	return expectedImprovementWithExploration(mean, std, best, 0.0)
}

// generateLatinHypercubeSamples creates a Latin Hypercube sample in the parameter space
// This ensures better coverage of the space than pure random sampling
func generateLatinHypercubeSamples(n int, bounds [][2]float64, rng *rand.Rand) [][]float64 {
	dim := len(bounds)
	result := make([][]float64, n)
	
	// Initialize each point
	for i := 0; i < n; i++ {
		result[i] = make([]float64, dim)
	}
	
	// For each dimension, create a permutation of intervals
	for j := 0; j < dim; j++ {
		// Create even spacing for this dimension
		spacing := make([]float64, n)
		for i := 0; i < n; i++ {
			spacing[i] = float64(i) / float64(n)
		}
		
		// Shuffle the spacing
		for i := n - 1; i > 0; i-- {
			k := rng.Intn(i + 1)
			spacing[i], spacing[k] = spacing[k], spacing[i]
		}
		
		// Assign to points and scale to bounds
		min, max := bounds[j][0], bounds[j][1]
		for i := 0; i < n; i++ {
			// Add random jitter within each interval
			jitter := rng.Float64() / float64(n)
			result[i][j] = min + (spacing[i] + jitter) * (max - min)
		}
	}
	
	return result
}

// GetBestSolution returns the best solution found so far
func (o *Optimizer) GetBestSolution() ([]float64, float64) {
	o.lock.Lock()
	defer o.lock.Unlock()
	
	bestX := make([]float64, len(o.bestX))
	copy(bestX, o.bestX)
	
	return bestX, o.bestY
}

// GetNumSamples returns the number of samples collected so far
func (o *Optimizer) GetNumSamples() int {
	o.lock.Lock()
	defer o.lock.Unlock()
	
	return o.samples
}
