// Package reservoir implements reservoir sampling algorithms for maintaining a 
// representative sample of a stream of data with minimal memory usage.
package reservoir

import (
	"math/rand"
	"sync"
	"time"
)

// ReservoirSampler implements the classic reservoir sampling algorithm.
// It maintains a fixed-size random sample from a stream of items.
type ReservoirSampler struct {
	capacity int           // Maximum number of items in the reservoir
	reservoir []interface{} // The sampled items
	count     int64         // Number of items seen so far
	lock      sync.RWMutex
	rng       *rand.Rand    // Random number generator
}

// NewReservoirSampler creates a new reservoir sampler with the specified capacity.
func NewReservoirSampler(capacity int) *ReservoirSampler {
	return &ReservoirSampler{
		capacity:  capacity,
		reservoir: make([]interface{}, 0, capacity),
		count:     0,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Add adds an item to the reservoir using the classic algorithm.
// If the reservoir is not full, the item is added directly.
// Otherwise, the item replaces an existing item with probability (capacity/count).
func (rs *ReservoirSampler) Add(item interface{}) {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	
	rs.count++
	
	// If the reservoir is not full, add the item directly
	if len(rs.reservoir) < rs.capacity {
		rs.reservoir = append(rs.reservoir, item)
		return
	}
	
	// Randomly decide whether to include this item
	j := rs.rng.Int63n(rs.count)
	if j < int64(rs.capacity) {
		// Replace an item in the reservoir
		rs.reservoir[int(j)] = item
	}
}

// GetSamples returns a copy of the current samples in the reservoir.
func (rs *ReservoirSampler) GetSamples() []interface{} {
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	
	result := make([]interface{}, len(rs.reservoir))
	copy(result, rs.reservoir)
	return result
}

// Count returns the total number of items seen so far.
func (rs *ReservoirSampler) Count() int64 {
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	return rs.count
}

// Reset clears the reservoir and resets the count.
func (rs *ReservoirSampler) Reset() {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	
	rs.reservoir = make([]interface{}, 0, rs.capacity)
	rs.count = 0
}

// SetCapacity updates the capacity of the reservoir.
// If new capacity is smaller, the reservoir is randomly downsampled.
// If new capacity is larger, the reservoir is kept as is but with room to grow.
func (rs *ReservoirSampler) SetCapacity(newCapacity int) {
	if newCapacity <= 0 {
		return // Invalid capacity
	}
	
	rs.lock.Lock()
	defer rs.lock.Unlock()
	
	// If reducing capacity, randomly downsample
	if newCapacity < len(rs.reservoir) {
		// Create a new reservoir
		newReservoir := make([]interface{}, newCapacity)
		
		// Randomly select items from the current reservoir
		indices := make(map[int]struct{})
		for len(indices) < newCapacity {
			idx := rs.rng.Intn(len(rs.reservoir))
			if _, exists := indices[idx]; !exists {
				indices[idx] = struct{}{}
			}
		}
		
		// Copy selected items to the new reservoir
		i := 0
		for idx := range indices {
			newReservoir[i] = rs.reservoir[idx]
			i++
		}
		
		rs.reservoir = newReservoir
	} else if newCapacity > rs.capacity {
		// If increasing capacity, resize the underlying slice
		newReservoir := make([]interface{}, len(rs.reservoir), newCapacity)
		copy(newReservoir, rs.reservoir)
		rs.reservoir = newReservoir
	}
	
	rs.capacity = newCapacity
}

// StratifiedReservoirSampler implements a stratified reservoir sampling algorithm.
// It maintains separate reservoirs for different strata (categories) of items.
type StratifiedReservoirSampler struct {
	reservoirs map[string]*ReservoirSampler // Map of stratum name to reservoir
	lock       sync.RWMutex
}

// NewStratifiedReservoirSampler creates a new stratified reservoir sampler.
func NewStratifiedReservoirSampler() *StratifiedReservoirSampler {
	return &StratifiedReservoirSampler{
		reservoirs: make(map[string]*ReservoirSampler),
	}
}

// Add adds an item to the appropriate stratum's reservoir.
func (srs *StratifiedReservoirSampler) Add(stratum string, item interface{}, capacity int) {
	srs.lock.Lock()
	
	// Create reservoir for this stratum if it doesn't exist
	if _, exists := srs.reservoirs[stratum]; !exists {
		srs.reservoirs[stratum] = NewReservoirSampler(capacity)
	}
	
	// Get the reservoir for this stratum
	reservoir := srs.reservoirs[stratum]
	
	// Unlock before adding to allow concurrent adds to different strata
	srs.lock.Unlock()
	
	// Add the item to the reservoir
	reservoir.Add(item)
}

// GetSamples returns all samples from all strata.
func (srs *StratifiedReservoirSampler) GetSamples() map[string][]interface{} {
	srs.lock.RLock()
	defer srs.lock.RUnlock()
	
	result := make(map[string][]interface{})
	
	for stratum, reservoir := range srs.reservoirs {
		result[stratum] = reservoir.GetSamples()
	}
	
	return result
}

// GetStratumSamples returns samples from a specific stratum.
func (srs *StratifiedReservoirSampler) GetStratumSamples(stratum string) []interface{} {
	srs.lock.RLock()
	defer srs.lock.RUnlock()
	
	if reservoir, exists := srs.reservoirs[stratum]; exists {
		return reservoir.GetSamples()
	}
	
	return nil
}

// Count returns the total number of items seen across all strata.
func (srs *StratifiedReservoirSampler) Count() int64 {
	srs.lock.RLock()
	defer srs.lock.RUnlock()
	
	var total int64 = 0
	for _, reservoir := range srs.reservoirs {
		total += reservoir.Count()
	}
	
	return total
}

// Strata returns a list of all strata names.
func (srs *StratifiedReservoirSampler) Strata() []string {
	srs.lock.RLock()
	defer srs.lock.RUnlock()
	
	strata := make([]string, 0, len(srs.reservoirs))
	for stratum := range srs.reservoirs {
		strata = append(strata, stratum)
	}
	
	return strata
}

// SetCapacity updates the capacity for a specific stratum.
func (srs *StratifiedReservoirSampler) SetCapacity(stratum string, capacity int) {
	srs.lock.Lock()
	defer srs.lock.Unlock()
	
	if reservoir, exists := srs.reservoirs[stratum]; exists {
		reservoir.SetCapacity(capacity)
	}
}

// Reset clears all reservoirs.
func (srs *StratifiedReservoirSampler) Reset() {
	srs.lock.Lock()
	defer srs.lock.Unlock()
	
	for _, reservoir := range srs.reservoirs {
		reservoir.Reset()
	}
}