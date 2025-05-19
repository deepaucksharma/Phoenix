package causality

import (
	"errors"
	"math"
)

// GrangerCausality computes a simple Granger causality F statistic between two
// time series x and y with the given lag. It returns the F statistic which can
// be compared against an F distribution to determine significance.
func GrangerCausality(x, y []float64, lag int) (float64, error) {
	n := len(x)
	if len(y) != n {
		return 0, errors.New("series length mismatch")
	}
	if lag <= 0 || n <= lag+1 {
		return 0, errors.New("insufficient data")
	}

	obs := n - lag
	Xr := make([][]float64, obs)
	Xu := make([][]float64, obs)
	target := make([]float64, obs)
	for t := lag; t < n; t++ {
		row := t - lag
		Xr[row] = make([]float64, lag)
		Xu[row] = make([]float64, 2*lag)
		target[row] = x[t]
		for i := 0; i < lag; i++ {
			Xr[row][i] = x[t-i-1]
			Xu[row][i] = x[t-i-1]
			Xu[row][i+lag] = y[t-i-1]
		}
	}

	br, err := leastSquares(Xr, target)
	if err != nil {
		return 0, err
	}
	bu, err := leastSquares(Xu, target)
	if err != nil {
		return 0, err
	}

	rssR := residualSumSquares(Xr, target, br)
	rssU := residualSumSquares(Xu, target, bu)

	df1 := float64(lag)
	df2 := float64(obs - 2*lag)
	if df2 <= 0 {
		return 0, errors.New("invalid degrees of freedom")
	}
	f := ((rssR - rssU) / df1) / (rssU / df2)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, errors.New("invalid statistic")
	}
	return f, nil
}

// residualSumSquares returns the residual sum of squares for X * beta ~ y.
func residualSumSquares(X [][]float64, y, beta []float64) float64 {
	var rss float64
	for i := range X {
		pred := 0.0
		for j, v := range X[i] {
			pred += v * beta[j]
		}
		d := y[i] - pred
		rss += d * d
	}
	return rss
}

// leastSquares solves the normal equations using Gaussian elimination.
func leastSquares(X [][]float64, y []float64) ([]float64, error) {
	m := len(X)
	if m == 0 {
		return nil, errors.New("empty matrix")
	}
	n := len(X[0])
	A := make([][]float64, n)
	for i := range A {
		A[i] = make([]float64, n+1)
	}
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			for k := 0; k < n; k++ {
				A[j][k] += X[i][j] * X[i][k]
			}
			A[j][n] += X[i][j] * y[i]
		}
	}

	// Gauss-Jordan elimination
	for i := 0; i < n; i++ {
		maxRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(A[k][i]) > math.Abs(A[maxRow][i]) {
				maxRow = k
			}
		}
		A[i], A[maxRow] = A[maxRow], A[i]
		pivot := A[i][i]
		if math.Abs(pivot) < 1e-12 {
			return nil, errors.New("singular matrix")
		}
		for j := i; j < n+1; j++ {
			A[i][j] /= pivot
		}
		for k := 0; k < n; k++ {
			if k == i {
				continue
			}
			factor := A[k][i]
			for j := i; j < n+1; j++ {
				A[k][j] -= factor * A[i][j]
			}
		}
	}

	beta := make([]float64, n)
	for i := 0; i < n; i++ {
		beta[i] = A[i][n]
	}
	return beta, nil
}
