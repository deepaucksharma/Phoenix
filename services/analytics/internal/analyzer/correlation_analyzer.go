package analyzer

import (
	"math"
	"sort"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

type CorrelationAnalyzer struct {
	minSamples int
}

type CorrelationResult struct {
	Metric1     string
	Metric2     string
	Correlation float64
	PValue      float64
	Strength    string // "strong", "moderate", "weak", "none"
	Direction   string // "positive", "negative", "none"
}

type MetricData struct {
	Name   string
	Values []float64
}

func NewCorrelationAnalyzer(minSamples int) *CorrelationAnalyzer {
	return &CorrelationAnalyzer{
		minSamples: minSamples,
	}
}

func (ca *CorrelationAnalyzer) AnalyzeCorrelations(metrics []MetricData) []CorrelationResult {
	results := []CorrelationResult{}

	for i := 0; i < len(metrics); i++ {
		for j := i + 1; j < len(metrics); j++ {
			if len(metrics[i].Values) < ca.minSamples || len(metrics[j].Values) < ca.minSamples {
				continue
			}

			result := ca.calculateCorrelation(metrics[i], metrics[j])
			if result != nil {
				results = append(results, *result)
			}
		}
	}

	// Sort by absolute correlation value
	sort.Slice(results, func(i, j int) bool {
		return math.Abs(results[i].Correlation) > math.Abs(results[j].Correlation)
	})

	return results
}

func (ca *CorrelationAnalyzer) calculateCorrelation(m1, m2 MetricData) *CorrelationResult {
	// Ensure equal length
	minLen := len(m1.Values)
	if len(m2.Values) < minLen {
		minLen = len(m2.Values)
	}

	v1 := m1.Values[:minLen]
	v2 := m2.Values[:minLen]

	// Calculate Pearson correlation
	correlation := stat.Correlation(v1, v2, nil)

	// Determine strength and direction
	absCorr := math.Abs(correlation)
	strength := "none"
	if absCorr >= 0.7 {
		strength = "strong"
	} else if absCorr >= 0.5 {
		strength = "moderate"
	} else if absCorr >= 0.3 {
		strength = "weak"
	}

	direction := "none"
	if correlation > 0.1 {
		direction = "positive"
	} else if correlation < -0.1 {
		direction = "negative"
	}

	// Calculate p-value (simplified t-test approach)
	n := float64(minLen)
	t := correlation * math.Sqrt((n-2)/(1-correlation*correlation))
	pValue := ca.approximatePValue(t, n-2)

	return &CorrelationResult{
		Metric1:     m1.Name,
		Metric2:     m2.Name,
		Correlation: correlation,
		PValue:      pValue,
		Strength:    strength,
		Direction:   direction,
	}
}

func (ca *CorrelationAnalyzer) approximatePValue(t, df float64) float64 {
	// Simplified p-value approximation using normal distribution
	// For more accurate results, use a proper t-distribution implementation
	z := t / math.Sqrt(df/2)
	pValue := 2 * (1 - ca.normalCDF(math.Abs(z)))
	return pValue
}

func (ca *CorrelationAnalyzer) normalCDF(x float64) float64 {
	// Approximation of the cumulative distribution function for standard normal
	return 0.5 * (1 + math.Erf(x/math.Sqrt(2)))
}

func (ca *CorrelationAnalyzer) CalculatePartialCorrelations(metrics []MetricData, controlIndex int) []CorrelationResult {
	if len(metrics) < 3 || controlIndex >= len(metrics) {
		return nil
	}

	results := []CorrelationResult{}
	control := metrics[controlIndex]

	for i := 0; i < len(metrics); i++ {
		if i == controlIndex {
			continue
		}
		for j := i + 1; j < len(metrics); j++ {
			if j == controlIndex {
				continue
			}

			partialCorr := ca.calculatePartialCorrelation(
				metrics[i], metrics[j], control,
			)
			if partialCorr != nil {
				results = append(results, *partialCorr)
			}
		}
	}

	return results
}

func (ca *CorrelationAnalyzer) calculatePartialCorrelation(m1, m2, control MetricData) *CorrelationResult {
	// Calculate correlations
	r12 := stat.Correlation(m1.Values, m2.Values, nil)
	r13 := stat.Correlation(m1.Values, control.Values, nil)
	r23 := stat.Correlation(m2.Values, control.Values, nil)

	// Partial correlation formula
	numerator := r12 - (r13 * r23)
	denominator := math.Sqrt((1 - r13*r13) * (1 - r23*r23))
	
	if denominator == 0 {
		return nil
	}

	partialCorr := numerator / denominator

	return &CorrelationResult{
		Metric1:     m1.Name + " (controlling for " + control.Name + ")",
		Metric2:     m2.Name,
		Correlation: partialCorr,
		Strength:    ca.getStrength(partialCorr),
		Direction:   ca.getDirection(partialCorr),
	}
}

func (ca *CorrelationAnalyzer) getStrength(correlation float64) string {
	absCorr := math.Abs(correlation)
	if absCorr >= 0.7 {
		return "strong"
	} else if absCorr >= 0.5 {
		return "moderate"
	} else if absCorr >= 0.3 {
		return "weak"
	}
	return "none"
}

func (ca *CorrelationAnalyzer) getDirection(correlation float64) string {
	if correlation > 0.1 {
		return "positive"
	} else if correlation < -0.1 {
		return "negative"
	}
	return "none"
}

func (ca *CorrelationAnalyzer) CalculateCorrelationMatrix(metrics []MetricData) *mat.Dense {
	n := len(metrics)
	corrMatrix := mat.NewDense(n, n, nil)

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				corrMatrix.Set(i, j, 1.0)
			} else {
				corr := stat.Correlation(metrics[i].Values, metrics[j].Values, nil)
				corrMatrix.Set(i, j, corr)
			}
		}
	}

	return corrMatrix
}