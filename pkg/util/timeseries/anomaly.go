package timeseries

import "math"

// DetectZScore returns the indices of values whose absolute z-score exceeds the
// given threshold.
func DetectZScore(data []float64, threshold float64) []int {
	n := len(data)
	if n == 0 {
		return nil
	}
	if threshold <= 0 {
		threshold = 3
	}
	var sum float64
	for _, v := range data {
		sum += v
	}
	mean := sum / float64(n)
	var varSum float64
	for _, v := range data {
		diff := v - mean
		varSum += diff * diff
	}
	std := math.Sqrt(varSum / float64(n))
	if std == 0 {
		return nil
	}
	anomalies := []int{}
	for i, v := range data {
		z := math.Abs((v - mean) / std)
		if z > threshold {
			anomalies = append(anomalies, i)
		}
	}
	return anomalies
}
