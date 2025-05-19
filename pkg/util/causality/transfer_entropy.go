package causality

import (
	"errors"
	"math"
)

// TransferEntropy computes a simple discrete transfer entropy from series y to x
// using the specified number of bins and lag. It returns the estimated transfer
// entropy in bits.
func TransferEntropy(x, y []float64, bins, lag int) (float64, error) {
	n := len(x)
	if len(y) != n {
		return 0, errors.New("series length mismatch")
	}
	if bins <= 1 || lag <= 0 || n <= lag+1 {
		return 0, errors.New("invalid parameters")
	}

	// Determine ranges for binning
	minX, maxX := minMax(x)
	minY, maxY := minMax(y)

	countsXYZ := make(map[[3]int]int)
	countsXZ := make(map[[2]int]int)
	countsYZ := make(map[[2]int]int)
	countsZ := make(map[int]int)
	samples := 0

	for t := lag; t < n-1; t++ {
		xt := bin(x[t], minX, maxX, bins)
		xNext := bin(x[t+1], minX, maxX, bins)
		yt := bin(y[t], minY, maxY, bins)
		keyXYZ := [3]int{xNext, xt, yt}
		keyXZ := [2]int{xNext, xt}
		keyYZ := [2]int{yt, xt}
		countsXYZ[keyXYZ]++
		countsXZ[keyXZ]++
		countsYZ[keyYZ]++
		countsZ[xt]++
		samples++
	}

	var te float64
	for key, cXYZ := range countsXYZ {
		cXZ := countsXZ[[2]int{key[0], key[1]}]
		cYZ := countsYZ[[2]int{key[2], key[1]}]
		cZ := countsZ[key[1]]
		pXYZ := float64(cXYZ) / float64(samples)
		pXZ := float64(cXZ) / float64(samples)
		pYZ := float64(cYZ) / float64(samples)
		pZ := float64(cZ) / float64(samples)
		if pXYZ == 0 || pXZ == 0 || pYZ == 0 || pZ == 0 {
			continue
		}
		te += pXYZ * math.Log2((pZ*pXYZ)/(pXZ*pYZ))
	}
	return te, nil
}

func minMax(v []float64) (float64, float64) {
	min := v[0]
	max := v[0]
	for _, val := range v {
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}
	return min, max
}

func bin(val, min, max float64, bins int) int {
	if max == min {
		return 0
	}
	pos := (val - min) / (max - min)
	if pos < 0 {
		pos = 0
	}
	if pos > 1 {
		pos = 1
	}
	idx := int(math.Floor(pos * float64(bins)))
	if idx >= bins {
		idx = bins - 1
	}
	return idx
}
