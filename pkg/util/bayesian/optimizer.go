package bayesian

import (
	"math"
	"math/rand"
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
}

// NewOptimizer creates a new optimizer for the given bounds.
func NewOptimizer(bounds [][2]float64) *Optimizer {
	dim := len(bounds)
	return &Optimizer{
		gp:         NewGaussianProcess(1.0, 1e-6),
		bounds:     bounds,
		candidates: 50,
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
		bestY:      math.Inf(-1),
		bestX:      make([]float64, dim),
	}
}

// AddSample records the observation of value y at position x.
func (o *Optimizer) AddSample(x []float64, y float64) {
	o.gp.AddSample(x, y)
	if y > o.bestY {
		o.bestY = y
		o.bestX = append([]float64{}, x...)
	}
}

// Suggest returns the next point to evaluate based on expected improvement.
func (o *Optimizer) Suggest() []float64 {
	dim := len(o.bounds)
	if len(o.gp.x) == 0 {
		mid := make([]float64, dim)
		for i, b := range o.bounds {
			mid[i] = (b[0] + b[1]) / 2
		}
		return mid
	}

	bestEI := -math.MaxFloat64
	bestPoint := make([]float64, dim)
	for i := 0; i < o.candidates; i++ {
		p := make([]float64, dim)
		for j, b := range o.bounds {
			p[j] = b[0] + o.rng.Float64()*(b[1]-b[0])
		}
		mean, variance := o.gp.Predict(p)
		ei := expectedImprovement(mean, math.Sqrt(variance), o.bestY)
		if ei > bestEI {
			bestEI = ei
			copy(bestPoint, p)
		}
	}
	return bestPoint
}

func expectedImprovement(mean, std, best float64) float64 {
	if std <= 0 {
		return 0
	}
	z := (mean - best) / std
	normal := distuv.UnitNormal
	return (mean-best)*normal.CDF(z) + std*normal.Prob(z)
}
