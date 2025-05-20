package bayesian

import (
	"math"
	"sync"

	"gonum.org/v1/gonum/mat"
)

// GaussianProcess implements a simple Gaussian Process regression model with
// an RBF kernel. It is intentionally lightweight for optimization routines.
type GaussianProcess struct {
	lengthScales []float64  // Length scales for each dimension (anisotropic RBF kernel)
	noise        float64    // Observation noise level
	variance     float64    // Signal variance (output scale)

	x [][]float64  // Input points
	y []float64    // Observed values
	
	lock sync.RWMutex // For thread safety
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
		lengthScales: []float64{lengthScale}, // Start with isotropic kernel
		noise:       noise,
		variance:    1.0,                    // Default signal variance
		x:           make([][]float64, 0),
		y:           make([]float64, 0),
	}
}

// SetLengthScales sets dimension-specific length scales for anisotropic kernel
func (gp *GaussianProcess) SetLengthScales(lengthScales []float64) {
	gp.lock.Lock()
	defer gp.lock.Unlock()
	
	// Make a copy of the length scales
	gp.lengthScales = make([]float64, len(lengthScales))
	copy(gp.lengthScales, lengthScales)
}

// SetNoise sets the observation noise level
func (gp *GaussianProcess) SetNoise(noise float64) {
	gp.lock.Lock()
	defer gp.lock.Unlock()
	
	if noise > 0 {
		gp.noise = noise
	}
}

// SetVariance sets the signal variance (output scale)
func (gp *GaussianProcess) SetVariance(variance float64) {
	gp.lock.Lock()
	defer gp.lock.Unlock()
	
	if variance > 0 {
		gp.variance = variance
	}
}

// AddSample adds a new observation to the process.
func (gp *GaussianProcess) AddSample(x []float64, value float64) {
	gp.lock.Lock()
	defer gp.lock.Unlock()
	
	// Make a deep copy of the input point
	xv := make([]float64, len(x))
	copy(xv, x)
	
	// Auto-expand lengthScales if needed
	if len(gp.lengthScales) < len(x) {
		newScales := make([]float64, len(x))
		// Copy existing scales
		for i := 0; i < len(gp.lengthScales); i++ {
			newScales[i] = gp.lengthScales[i]
		}
		// Fill in remaining dimensions with default value
		for i := len(gp.lengthScales); i < len(x); i++ {
			newScales[i] = 1.0
		}
		gp.lengthScales = newScales
	}
	
	gp.x = append(gp.x, xv)
	gp.y = append(gp.y, value)
}

// Predict returns the mean and variance for a given point.
func (gp *GaussianProcess) Predict(x []float64) (float64, float64) {
	gp.lock.RLock()
	defer gp.lock.RUnlock()
	
	n := len(gp.x)
	if n == 0 {
		return 0, gp.variance // return prior
	}
	
	// Auto-expand lengthScales if needed
	lengthScales := gp.lengthScales
	if len(lengthScales) < len(x) {
		tempScales := make([]float64, len(x))
		// Copy existing scales
		for i := 0; i < len(gp.lengthScales); i++ {
			tempScales[i] = gp.lengthScales[i]
		}
		// Fill in remaining dimensions with default value
		for i := len(gp.lengthScales); i < len(x); i++ {
			tempScales[i] = 1.0
		}
		lengthScales = tempScales
	}

	// Build covariance matrix K
	K := mat.NewSymDense(n, nil)
	for i := 0; i < n; i++ {
		for j := i; j < n; j++ {
			val := rbfAnisotropic(gp.x[i], gp.x[j], lengthScales) * gp.variance
			if i == j {
				val += gp.noise
			}
			K.SetSym(i, j, val)
		}
	}

	// Compute k(x, X) - covariance between test point and training points
	kVec := make([]float64, n)
	for i := 0; i < n; i++ {
		kVec[i] = rbfAnisotropic(x, gp.x[i], lengthScales) * gp.variance
	}

	// Cholesky decomposition of K
	var chol mat.Cholesky
	if ok := chol.Factorize(K); !ok {
		// If factorization fails, add more noise to the diagonal
		for i := 0; i < n; i++ {
			K.SetSym(i, i, K.At(i, i) + 1e-6)
		}
		if ok := chol.Factorize(K); !ok {
			// If still fails, return prior
			return 0, gp.variance
		}
	}

	// Solve for alpha = K^{-1} y
	yVec := mat.NewVecDense(n, gp.y)
	alpha := mat.NewVecDense(n, nil)
	// Use alternative solution since SolveVec method is not available
	cholMat := &mat.Dense{}
	chol.SolveTo(cholMat, mat.NewDense(n, 1, yVec.RawVector().Data))
	alpha.CopyVec(mat.NewVecDense(n, cholMat.RawMatrix().Data))

	// Compute mean = k^T * alpha
	mean := mat.Dot(mat.NewVecDense(n, kVec), alpha)

	// Solve for v = K^{-1} k
	v := mat.NewVecDense(n, nil)
	// Use alternative solution since SolveVec method is not available
	kVecDense := mat.NewDense(n, 1, kVec)
	vDense := &mat.Dense{}
	chol.SolveTo(vDense, kVecDense)
	v.CopyVec(mat.NewVecDense(n, vDense.RawMatrix().Data))
	
	// Compute variance: k(x,x) - k^T K^-1 k
	kxx := rbfAnisotropic(x, x, lengthScales) * gp.variance + gp.noise
	variance := kxx - mat.Dot(mat.NewVecDense(n, kVec), v)
	
	// Ensure positive variance with numerical stability safeguard
	if variance < 1e-8 {
		variance = 1e-8
	}
	
	return mean, variance
}

// rbfAnisotropic implements an RBF kernel with dimension-specific length scales
func rbfAnisotropic(a, b []float64, lengthScales []float64) float64 {
	sum := 0.0
	// Use the minimum of the dimensions in both arrays
	dim := len(a)
	if len(b) < dim {
		dim = len(b)
	}
	if len(lengthScales) < dim {
		dim = len(lengthScales)
	}
	
	// Compute weighted distance
	for i := 0; i < dim; i++ {
		d := a[i] - b[i]
		// Use dimension-specific length scale
		ls := lengthScales[i]
		if ls <= 0 {
			ls = 1.0 // Fallback for invalid length scale
		}
		sum += (d * d) / (ls * ls)
	}
	
	return math.Exp(-0.5 * sum)
}

// rbf implements a standard isotropic RBF kernel with single length scale
func rbf(a, b []float64, l float64) float64 {
	sum := 0.0
	for i := range a {
		if i >= len(b) {
			break
		}
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Exp(-0.5 * sum / (l * l))
}
