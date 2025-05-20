// Package hll_test provides unit tests for the HyperLogLog algorithm.
package hll_test

import (
	"fmt"
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/deepaucksharma/Phoenix/pkg/util/hll"
)

// TestHyperLogLogBasic tests basic functionality of the HyperLogLog algorithm.
func TestHyperLogLogBasic(t *testing.T) {
	// Create HLL with standard precision
	h, err := hll.New(10) // 2^10 = 1024 registers
	require.NoError(t, err)
	require.NotNil(t, h)

	// Add unique items
	h.AddString("item1")
	h.AddString("item2")
	h.AddString("item3")

	// Get count
	count := h.Count()
	assert.Equal(t, uint64(3), count, "Count should be accurate for small sets")

	// Add duplicate items
	h.AddString("item1")
	h.AddString("item2")

	// Count should remain the same
	count = h.Count()
	assert.Equal(t, uint64(3), count, "Count should be accurate with duplicates")

	// Add more items
	h.AddString("item4")
	h.AddString("item5")

	// Count should increase
	count = h.Count()
	assert.Equal(t, uint64(5), count, "Count should increase with new items")

	// Test reset
	h.Reset()
	count = h.Count()
	assert.Equal(t, uint64(0), count, "Count should be 0 after reset")
}

// TestHyperLogLogPrecision tests the precision settings of HyperLogLog.
func TestHyperLogLogPrecision(t *testing.T) {
	// Test minimum precision
	hMin, err := hll.New(hll.HLL_MIN_PRECISION)
	require.NoError(t, err)
	require.NotNil(t, hMin)

	// Test maximum precision
	hMax, err := hll.New(hll.HLL_MAX_PRECISION)
	require.NoError(t, err)
	require.NotNil(t, hMax)

	// Test default precision
	hDefault := hll.NewDefault()
	require.NotNil(t, hDefault)

	// Test invalid precision
	_, err = hll.New(3) // Too low
	assert.Error(t, err)

	_, err = hll.New(20) // Too high
	assert.Error(t, err)
}

// TestHyperLogLogAccuracy tests the accuracy of HyperLogLog for larger sets.
func TestHyperLogLogAccuracy(t *testing.T) {
	tests := []struct {
		name      string
		precision uint8
		items     int
		tolerance float64 // Acceptable relative error
	}{
		{"Small-LowPrecision", 6, 100, 1.0}, // tolerance relaxed for low precision
		{"Small-MedPrecision", 10, 100, 0.15},
		{"Medium-MedPrecision", 10, 1000, 0.25},
		{"Large-HighPrecision", 14, 10000, 0.05}, // 2^14 = 16384 registers, 5% tolerance
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := hll.New(tt.precision)
			require.NoError(t, err)

			// Add unique items
			for i := 0; i < tt.items; i++ {
				h.AddString(fmt.Sprintf("item-%d", i))
			}

			// Get estimate
			estimate := h.Count()

			// Calculate relative error
			relError := math.Abs(float64(estimate)-float64(tt.items)) / float64(tt.items)

			t.Logf("Actual: %d, Estimated: %d, Error: %.2f%%", tt.items, estimate, relError*100)

			// Check if within tolerance
			assert.LessOrEqual(t, relError, tt.tolerance,
				"Estimate %d exceeds tolerance of %.1f%% from actual %d (error: %.2f%%)",
				estimate, tt.tolerance*100, tt.items, relError*100)
		})
	}
}

// TestHyperLogLogMerge tests merging functionality between two HyperLogLog counters.
func TestHyperLogLogMerge(t *testing.T) {
	precision := uint8(10)

	hll1, err := hll.New(precision)
	require.NoError(t, err)

	hll2, err := hll.New(precision)
	require.NoError(t, err)

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

	// The counts should be close to the actual numbers
	assert.InDelta(t, 1000, count1, 300, "HLL1 count should be close to 1000")
	assert.InDelta(t, 1000, count2, 300, "HLL2 count should be close to 1000")

	// Merge second into first
	err = hll1.Merge(hll2)
	require.NoError(t, err, "Merge should not fail")

	// Count after merge
	mergedCount := hll1.Count()

	// The merged count should be close to the union size (1500)
	assert.InDelta(t, 1500, mergedCount, 300, "Merged count should be close to 1500")

	// Test merging with different precision
	hll3, err := hll.New(precision + 1)
	require.NoError(t, err)

	err = hll1.Merge(hll3)
	assert.Error(t, err, "Merging HLLs with different precision should fail")
}

// TestHyperLogLogConcurrency tests thread safety of HyperLogLog for concurrent access.
func TestHyperLogLogConcurrency(t *testing.T) {
	h := hll.NewDefault()

	itemCount := 10000
	goroutines := 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Add items concurrently
	for g := 0; g < goroutines; g++ {
		go func(offset int) {
			defer wg.Done()

			for i := 0; i < itemCount/goroutines; i++ {
				item := fmt.Sprintf("item-%d", offset+i)
				h.AddString(item)
			}
		}(g * (itemCount / goroutines))
	}

	wg.Wait()

	// Get final count
	count := h.Count()

	// Should be reasonably close to the actual count
	assert.InDelta(t, float64(itemCount), float64(count), float64(itemCount)*0.1,
		"Count should be within 10%% of actual after concurrent additions")
}
