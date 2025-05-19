// Package hll implements the HyperLogLog algorithm for cardinality estimation.
// HyperLogLog is a probabilistic data structure used to estimate the number of
// unique elements in a multiset with minimal memory usage.
package hll

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"sync"
)

const (
	// HLL_MIN_PRECISION is the minimum precision (4 bits to create 16 registers)
	HLL_MIN_PRECISION = 4
	
	// HLL_MAX_PRECISION is the maximum precision (16 bits to create 65536 registers)
	HLL_MAX_PRECISION = 16
	
	// HLL_DEFAULT_PRECISION is the default precision (10 bits to create 1024 registers)
	HLL_DEFAULT_PRECISION = 10
)

// HyperLogLog implements the HyperLogLog algorithm for cardinality estimation.
type HyperLogLog struct {
	registers []uint8 // Array of registers
	m         uint32  // Number of registers (2^precision)
	precision uint8   // Number of bits for register addressing
	alpha     float64 // Bias correction factor
	lock      sync.RWMutex
}

// New creates a new HyperLogLog with the specified precision.
func New(precision uint8) (*HyperLogLog, error) {
	if precision < HLL_MIN_PRECISION || precision > HLL_MAX_PRECISION {
		return nil, fmt.Errorf("precision must be between %d and %d", HLL_MIN_PRECISION, HLL_MAX_PRECISION)
	}
	
	m := uint32(1) << precision
	
	// Compute alpha based on precision
	var alpha float64
	switch m {
	case 16:
		alpha = 0.673
	case 32:
		alpha = 0.697
	case 64:
		alpha = 0.709
	default:
		alpha = 0.7213 / (1.0 + 1.079/float64(m))
	}
	
	return &HyperLogLog{
		registers: make([]uint8, m),
		m:         m,
		precision: precision,
		alpha:     alpha,
	}, nil
}

// NewDefault creates a new HyperLogLog with the default precision.
func NewDefault() *HyperLogLog {
	hll, _ := New(HLL_DEFAULT_PRECISION)
	return hll
}

// Add adds an element to the HyperLogLog.
func (h *HyperLogLog) Add(data []byte) {
	h.lock.Lock()
	defer h.lock.Unlock()
	
	// Compute 64-bit hash
	hash := computeHash(data)
	
	// Determine register index using first 'precision' bits
	idx := hash & (h.m - 1)
	
	// Count leading zeros in the remaining bits (shifted by precision)
	zeros := countLeadingZeros(hash >> h.precision)
	
	// Update register if the new value is larger
	if zeros > h.registers[idx] {
		h.registers[idx] = zeros
	}
}

// AddString adds a string element to the HyperLogLog.
func (h *HyperLogLog) AddString(s string) {
	h.Add([]byte(s))
}

// Count returns the estimated cardinality.
func (h *HyperLogLog) Count() uint64 {
	h.lock.RLock()
	defer h.lock.RUnlock()
	
	// Compute estimate using harmonic mean
	sum := 0.0
	for _, val := range h.registers {
		sum += math.Pow(2.0, -float64(val))
	}
	
	estimate := h.alpha * float64(h.m*h.m) / sum
	
	// Apply correction for small and large ranges
	if estimate <= 2.5*float64(h.m) {
		// Small range correction
		zeros := 0
		for _, val := range h.registers {
			if val == 0 {
				zeros++
			}
		}
		
		if zeros > 0 {
			// Linear counting for small ranges
			return uint64(float64(h.m) * math.Log(float64(h.m)/float64(zeros)))
		}
	} else if estimate > float64(uint32(1)<<32)/30.0 {
		// Large range correction
		return uint64(-math.Pow(2.0, 32) * math.Log(1.0-estimate/math.Pow(2.0, 32)))
	}
	
	return uint64(estimate)
}

// Merge combines this HyperLogLog with another HyperLogLog.
func (h *HyperLogLog) Merge(other *HyperLogLog) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	
	if h.precision != other.precision {
		return fmt.Errorf("cannot merge HyperLogLogs with different precision (%d vs %d)", 
			h.precision, other.precision)
	}
	
	for i := uint32(0); i < h.m; i++ {
		if other.registers[i] > h.registers[i] {
			h.registers[i] = other.registers[i]
		}
	}
	
	return nil
}

// Reset clears all registers.
func (h *HyperLogLog) Reset() {
	h.lock.Lock()
	defer h.lock.Unlock()
	
	for i := range h.registers {
		h.registers[i] = 0
	}
}

// computeHash generates a 64-bit hash for the input data.
func computeHash(data []byte) uint32 {
	h := fnv.New32a()
	h.Write(data)
	return h.Sum32()
}

// countLeadingZeros counts the number of leading zeros in a 32-bit value.
func countLeadingZeros(x uint32) uint8 {
	if x == 0 {
		return 32
	}
	
	n := uint8(0)
	if x&0xFFFF0000 == 0 {
		n += 16
		x <<= 16
	}
	if x&0xFF000000 == 0 {
		n += 8
		x <<= 8
	}
	if x&0xF0000000 == 0 {
		n += 4
		x <<= 4
	}
	if x&0xC0000000 == 0 {
		n += 2
		x <<= 2
	}
	if x&0x80000000 == 0 {
		n += 1
	}
	
	return n + 1 // Add 1 for the 1-indexed position
}