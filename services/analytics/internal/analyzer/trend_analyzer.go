package analyzer

import (
	"math"
	"sort"
	"time"

	"gonum.org/v1/gonum/stat"
)

type TrendAnalyzer struct {
	windowSize int
}

type TrendResult struct {
	Slope       float64
	Intercept   float64
	R2          float64
	Trend       string // "increasing", "decreasing", "stable"
	Confidence  float64
	Prediction  float64
	ChangeRate  float64
}

type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

func NewTrendAnalyzer(windowSize int) *TrendAnalyzer {
	return &TrendAnalyzer{
		windowSize: windowSize,
	}
}

func (ta *TrendAnalyzer) AnalyzeTrend(data []DataPoint) *TrendResult {
	if len(data) < 2 {
		return &TrendResult{Trend: "insufficient_data"}
	}

	// Sort by timestamp
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	// Prepare data for regression
	n := len(data)
	if n > ta.windowSize {
		data = data[n-ta.windowSize:]
		n = ta.windowSize
	}

	x := make([]float64, n)
	y := make([]float64, n)
	baseTime := data[0].Timestamp.Unix()

	for i, dp := range data {
		x[i] = float64(dp.Timestamp.Unix() - baseTime)
		y[i] = dp.Value
	}

	// Linear regression
	alpha, beta := stat.LinearRegression(x, y, nil, false)

	// Calculate R-squared
	meanY := stat.Mean(y, nil)
	var ssTotal, ssRes float64
	for i := range y {
		yPred := alpha + beta*x[i]
		ssTotal += math.Pow(y[i]-meanY, 2)
		ssRes += math.Pow(y[i]-yPred, 2)
	}
	r2 := 1 - (ssRes / ssTotal)

	// Determine trend
	trend := "stable"
	changeRate := (beta / meanY) * 100 // Percentage change per second
	
	if math.Abs(changeRate) > 0.1 {
		if beta > 0 {
			trend = "increasing"
		} else {
			trend = "decreasing"
		}
	}

	// Predict next value (1 hour ahead)
	nextX := x[n-1] + 3600
	prediction := alpha + beta*nextX

	// Calculate confidence based on R-squared and data points
	confidence := r2 * (float64(n) / float64(ta.windowSize))

	return &TrendResult{
		Slope:      beta,
		Intercept:  alpha,
		R2:         r2,
		Trend:      trend,
		Confidence: confidence,
		Prediction: prediction,
		ChangeRate: changeRate,
	}
}

func (ta *TrendAnalyzer) DetectAnomaly(data []DataPoint, threshold float64) []int {
	if len(data) < ta.windowSize {
		return nil
	}

	anomalies := []int{}
	
	// Calculate moving average and standard deviation
	for i := ta.windowSize; i < len(data); i++ {
		window := data[i-ta.windowSize : i]
		values := make([]float64, ta.windowSize)
		for j, dp := range window {
			values[j] = dp.Value
		}
		
		mean := stat.Mean(values, nil)
		stdDev := stat.StdDev(values, nil)
		
		// Z-score calculation
		zScore := math.Abs((data[i].Value - mean) / stdDev)
		if zScore > threshold {
			anomalies = append(anomalies, i)
		}
	}
	
	return anomalies
}

func (ta *TrendAnalyzer) CalculateSeasonality(data []DataPoint, period int) map[int]float64 {
	if len(data) < period*2 {
		return nil
	}

	seasonalComponents := make(map[int]float64)
	counts := make(map[int]int)

	for i, dp := range data {
		bucket := i % period
		seasonalComponents[bucket] += dp.Value
		counts[bucket]++
	}

	// Calculate average for each seasonal component
	for bucket := range seasonalComponents {
		if counts[bucket] > 0 {
			seasonalComponents[bucket] /= float64(counts[bucket])
		}
	}

	return seasonalComponents
}