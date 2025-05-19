package bayesian

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// GaussianProcess implements a simple Gaussian Process regression model with
// an RBF kernel. It is intentionally lightweight for optimization routines.
type GaussianProcess struct {
	lengthScale float64
	noise       float64

	x [][]float64
	y []float64
}

// NewGaussianProcess creates a new GaussianProcess with the given kernel
// length scale and observation noise.
func NewGaussianProcess(lengthScale, noise float64) *GaussianProcess {
	if lengthScale <= 0 {
		lengthScale = 1
	}
	if noise <= 0 {
		noise = 1e-6
	}
	return &GaussianProcess{
		lengthScale: lengthScale,
		noise:       noise,
		x:           make([][]float64, 0),
		y:           make([]float64, 0),
	}
}

// AddSample adds a new observation to the process.
func (gp *GaussianProcess) AddSample(x []float64, value float64) {
	xv := make([]float64, len(x))
	copy(xv, x)
	gp.x = append(gp.x, xv)
	gp.y = append(gp.y, value)
}

// Predict returns the mean and variance for a given point.
func (gp *GaussianProcess) Predict(x []float64) (float64, float64) {
	n := len(gp.x)
	if n == 0 {
		return 0, 1 // prior
	}

	K := mat.NewSymDense(n, nil)
	for i := 0; i < n; i++ {
		for j := i; j < n; j++ {
			val := rbf(gp.x[i], gp.x[j], gp.lengthScale)
			if i == j {
				val += gp.noise
			}
			K.SetSym(i, j, val)
		}
	}

	// Compute k(x, X)
	kVec := make([]float64, n)
	for i := 0; i < n; i++ {
		kVec[i] = rbf(x, gp.x[i], gp.lengthScale)
	}

	var chol mat.Cholesky
	if ok := chol.Factorize(K); !ok {
		return 0, 1
	}

	// Solve for alpha = K^{-1} y
	yVec := mat.NewVecDense(n, gp.y)
	alpha := mat.NewVecDense(n, nil)
	if err := chol.SolveVec(alpha, yVec); err != nil {
		return 0, 1
	}

	// Compute mean = k^T * alpha
	mean := mat.Dot(mat.NewVecDense(n, kVec), alpha)

	// Solve for v = K^{-1} k
	v := mat.NewVecDense(n, nil)
	if err := chol.SolveVec(v, mat.NewVecDense(n, kVec)); err != nil {
		return mean, 1
	}
	kxx := rbf(x, x, gp.lengthScale) + gp.noise
	varience := kxx - mat.Dot(mat.NewVecDense(n, kVec), v)
	if varience < 1e-8 {
		varience = 1e-8
	}
	return mean, varience
}

func rbf(a, b []float64, l float64) float64 {
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Exp(-0.5 * sum / (l * l))
}
