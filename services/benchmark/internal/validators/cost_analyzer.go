package validators

import (
	"fmt"
	"math"
)

// CostAnalyzer analyzes and validates cost efficiency metrics
type CostAnalyzer struct {
	costPerMillionDatapoints float64
	maxAcceptableCost        float64
}

type CostMetrics struct {
	DatapointsProcessed int64
	ComputeCostUSD      float64
	StorageCostUSD      float64
	NetworkCostUSD      float64
}

type CostAnalysis struct {
	TotalCostUSD             float64
	CostPerMillionDatapoints float64
	CostEfficiencyScore      float64
	PipelineCostBreakdown    map[string]float64
}

func NewCostAnalyzer() *CostAnalyzer {
	return &CostAnalyzer{
		costPerMillionDatapoints: 0.50, // $0.50 per million datapoints baseline
		maxAcceptableCost:        1.00, // $1.00 per million datapoints max
	}
}

func (ca *CostAnalyzer) Analyze(metrics CostMetrics) *CostAnalysis {
	totalCost := metrics.ComputeCostUSD + metrics.StorageCostUSD + metrics.NetworkCostUSD
	
	costPerMillion := 0.0
	if metrics.DatapointsProcessed > 0 {
		costPerMillion = (totalCost / float64(metrics.DatapointsProcessed)) * 1000000
	}
	
	// Calculate efficiency score (1.0 = baseline, >1.0 = better than baseline)
	efficiencyScore := 1.0
	if costPerMillion > 0 {
		efficiencyScore = ca.costPerMillionDatapoints / costPerMillion
	}
	
	return &CostAnalysis{
		TotalCostUSD:             totalCost,
		CostPerMillionDatapoints: costPerMillion,
		CostEfficiencyScore:      efficiencyScore,
		PipelineCostBreakdown: map[string]float64{
			"compute": metrics.ComputeCostUSD,
			"storage": metrics.StorageCostUSD,
			"network": metrics.NetworkCostUSD,
		},
	}
}

func (ca *CostAnalyzer) Validate(analysis *CostAnalysis) (bool, []string) {
	var passed = true
	var failures []string

	if analysis.CostPerMillionDatapoints > ca.maxAcceptableCost {
		passed = false
		failures = append(failures, fmt.Sprintf("Cost per million datapoints $%.3f exceeds maximum $%.3f", 
			analysis.CostPerMillionDatapoints, ca.maxAcceptableCost))
	}

	if analysis.CostEfficiencyScore < 0.8 {
		passed = false
		failures = append(failures, fmt.Sprintf("Cost efficiency score %.2f is below acceptable threshold 0.80", 
			analysis.CostEfficiencyScore))
	}

	// Check for cost anomalies
	if analysis.PipelineCostBreakdown["compute"] > analysis.TotalCostUSD*0.7 {
		failures = append(failures, "Warning: Compute costs exceed 70% of total costs")
	}

	return passed, failures
}

func (ca *CostAnalyzer) EstimateMonthlyCost(hourlyMetrics CostMetrics) float64 {
	analysis := ca.Analyze(hourlyMetrics)
	return analysis.TotalCostUSD * 24 * 30 // Hours * Days
}

func (ca *CostAnalyzer) CalculateROI(costSavings, implementationCost float64) float64 {
	if implementationCost == 0 {
		return math.Inf(1)
	}
	return (costSavings - implementationCost) / implementationCost * 100
}

func (ca *CostAnalyzer) Name() string {
	return "cost_analyzer"
}