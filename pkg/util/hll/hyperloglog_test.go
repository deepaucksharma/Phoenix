package hll

import (
	"fmt"
	"testing"
)

func TestHyperLogLog(t *testing.T) {
	tests := []struct {
		name      string
		precision uint8
		items     int
		tolerance float64 // Acceptable relative error
	}{
		{"Small", 10, 100, 0.1},    // Small set with 10% tolerance
		{"Medium", 12, 10000, 0.05}, // Medium set with 5% tolerance
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hll, err := New(tt.precision)
			if err != nil {
				t.Fatalf("Failed to create HyperLogLog: %v", err)
			}

			// Add unique items
			for i := 0; i < tt.items; i++ {
				hll.AddString(fmt.Sprintf("item-%d", i))
			}

			// Get estimate
			estimate := hll.Count()

			// Calculate relative error
			relError := float64(estimate-uint64(tt.items)) / float64(tt.items)
			if relError < 0 {
				relError = -relError // Absolute value
			}

			t.Logf("Actual: %d, Estimated: %d, Error: %.2f%%", tt.items, estimate, relError*100)

			// Check if within tolerance
			if relError > tt.tolerance {
				t.Errorf("Estimate %d exceeds tolerance of %.1f%% from actual %d (error: %.2f%%)",
					estimate, tt.tolerance*100, tt.items, relError*100)
			}
		})
	}
}

func TestHyperLogLogMerge(t *testing.T) {
	precision := uint8(10)
	
	hll1, _ := New(precision)
	hll2, _ := New(precision)
	
	// Add items to first HLL (1-1000)
	for i := 1; i <= 1000; i++ {
		hll1.AddString(fmt.Sprintf("item-%d", i))
	}
	
	// Add items to second HLL (501-1500)
	for i := 501; i <= 1500; i++ {
		hll2.AddString(fmt.Sprintf("item-%d", i))
	}
	
	// Counts before merge
	count1 := hll1.Count()
	count2 := hll2.Count()
	
	// Merge second into first
	err := hll1.Merge(hll2)
	if err != nil {
		t.Fatalf("Failed to merge HyperLogLogs: %v", err)
	}
	
	// Count after merge
	mergedCount := hll1.Count()
	
	t.Logf("HLL1 count: %d, HLL2 count: %d, Merged count: %d", count1, count2, mergedCount)
	
	// Expected union cardinality is 1500
	expected := uint64(1500)
	tolerance := 0.1 // 10% tolerance
	
	relError := float64(mergedCount-expected) / float64(expected)
	if relError < 0 {
		relError = -relError // Absolute value
	}
	
	if relError > tolerance {
		t.Errorf("Merged estimate %d exceeds tolerance of %.1f%% from expected %d (error: %.2f%%)",
			mergedCount, tolerance*100, expected, relError*100)
	}
}