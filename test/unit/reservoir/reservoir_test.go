package reservoir_test

import (
	"fmt"
	"math"
	"testing"
	
	"github.com/yourorg/sa-omf/pkg/util/reservoir"
)

func TestReservoirSampler(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		items    int
	}{
		{"Small", 10, 100},
		{"Medium", 50, 1000},
		{"Large", 100, 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := reservoir.NewReservoirSampler(tt.capacity)

			// Add items
			for i := 0; i < tt.items; i++ {
				rs.Add(i)
			}

			// Check count
			if rs.Count() != int64(tt.items) {
				t.Errorf("Count = %d, want %d", rs.Count(), tt.items)
			}

			// Check reservoir size
			samples := rs.GetSamples()
			expectedSize := tt.capacity
			if tt.items < tt.capacity {
				expectedSize = tt.items
			}
			
			if len(samples) != expectedSize {
				t.Errorf("Reservoir size = %d, want %d", len(samples), expectedSize)
			}

			// Check uniqueness
			uniqueItems := make(map[interface{}]struct{})
			for _, item := range samples {
				uniqueItems[item] = struct{}{}
			}
			
			if len(uniqueItems) != len(samples) {
				t.Errorf("Found %d unique items, but reservoir size is %d", len(uniqueItems), len(samples))
			}
		})
	}
}

func TestReservoirSamplerDistribution(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping distribution test in short mode")
	}
	
	capacity := 1000
	itemCount := 100000
	
	rs := reservoir.NewReservoirSampler(capacity)

	// Add items
	for i := 0; i < itemCount; i++ {
		rs.Add(i)
	}

	// Get samples
	samples := rs.GetSamples()
	
	// Check distribution
	buckets := 10
	histogram := make([]int, buckets)
	
	for _, item := range samples {
		val := item.(int)
		bucket := (val * buckets) / itemCount
		if bucket >= buckets {
			bucket = buckets - 1
		}
		histogram[bucket]++
	}
	
	// Expected number per bucket is capacity / buckets
	expected := float64(capacity) / float64(buckets)
	for i, count := range histogram {
		// Allow 20% error (chi-squared would be better but this is simple)
		error := math.Abs(float64(count)-expected) / expected
		t.Logf("Bucket %d: %d items (%.2f%% error)", i, count, error*100)
		
		if error > 0.2 {
			t.Errorf("Bucket %d has %d items, expected around %.1f (error: %.2f%%)", 
				i, count, expected, error*100)
		}
	}
}

func TestStratifiedReservoirSampler(t *testing.T) {
	srs := reservoir.NewStratifiedReservoirSampler()
	
	// Add items to different strata
	strata := []string{"low", "medium", "high"}
	capacities := map[string]int{
		"low":    10,
		"medium": 20,
		"high":   30,
	}
	
	itemsPerStratum := 100
	
	for _, stratum := range strata {
		for i := 0; i < itemsPerStratum; i++ {
			srs.Add(stratum, fmt.Sprintf("%s-item-%d", stratum, i), capacities[stratum])
		}
	}
	
	// Check total count
	expectedCount := int64(itemsPerStratum * len(strata))
	if srs.Count() != expectedCount {
		t.Errorf("Total count = %d, want %d", srs.Count(), expectedCount)
	}
	
	// Check individual strata
	for _, stratum := range strata {
		capacity := capacities[stratum]
		samples := srs.GetStratumSamples(stratum)
		
		if len(samples) != capacity {
			t.Errorf("Stratum %s: reservoir size = %d, want %d", stratum, len(samples), capacity)
		}
		
		// Check all items belong to this stratum
		for _, item := range samples {
			s := item.(string)
			if s[:len(stratum)] != stratum {
				t.Errorf("Item %s found in wrong stratum %s", s, stratum)
			}
		}
	}
	
	// Test capacity change
	newCapacity := 5
	srs.SetCapacity("low", newCapacity)
	samples := srs.GetStratumSamples("low")
	if len(samples) != newCapacity {
		t.Errorf("After capacity change, stratum 'low': reservoir size = %d, want %d", 
			len(samples), newCapacity)
	}
	
	// Test reset
	srs.Reset()
	for _, stratum := range strata {
		samples := srs.GetStratumSamples(stratum)
		if len(samples) != 0 {
			t.Errorf("After reset, stratum %s has %d samples, expected 0", stratum, len(samples))
		}
	}
}