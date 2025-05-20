package pid

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// OscillationDetector monitors a controller for oscillation patterns
// and can trip a circuit breaker to prevent unstable behavior
type OscillationDetector struct {
	// Configuration
	sampleWindow       int           // Number of samples to track for oscillation detection
	oscillationThresholdPercent float64 // Percentage of zero crossings required to detect oscillation
	minSignalMagnitude float64      // Minimum magnitude for a signal to be considered significant
	minDuration       time.Duration // Minimum duration of oscillation before tripping
	resetDuration     time.Duration // Time after which to auto-reset the circuit breaker
	
	// State
	signalHistory      []float64     // History of signal values for oscillation detection
	valueHistory       []float64     // History of measured values
	signalTimeHistory  []time.Time   // Timestamps of each signal sample
	isTripped          bool          // Whether the circuit breaker is currently tripped
	tripTime           time.Time     // When the circuit breaker was tripped
	overrideUntil      time.Time     // Manual override expiration
	
	lock               sync.RWMutex  // For thread safety
}

// NewOscillationDetector creates a new oscillation detector with default parameters
func NewOscillationDetector() *OscillationDetector {
	return &OscillationDetector{
		sampleWindow:       20,                 // Track 20 samples
		oscillationThresholdPercent: 60.0,      // 60% of samples must show oscillation
		minSignalMagnitude: 0.05,               // Minimum magnitude to be significant
		minDuration:        time.Second * 30,   // 30 seconds of oscillation before tripping
		resetDuration:      time.Minute * 5,    // Auto-reset after 5 minutes
		signalHistory:      make([]float64, 0, 20),
		valueHistory:       make([]float64, 0, 20),
		signalTimeHistory:  make([]time.Time, 0, 20),
		isTripped:          false,
	}
}

// Configure sets custom configuration parameters
func (od *OscillationDetector) Configure(sampleWindow int, thresholdPercent, minMagnitude float64, 
                                        minDuration, resetDuration time.Duration) {
	od.lock.Lock()
	defer od.lock.Unlock()
	
	if sampleWindow > 0 {
		od.sampleWindow = sampleWindow
	}
	
	if thresholdPercent > 0 && thresholdPercent <= 100 {
		od.oscillationThresholdPercent = thresholdPercent
	}
	
	if minMagnitude > 0 {
		od.minSignalMagnitude = minMagnitude
	}
	
	if minDuration > 0 {
		od.minDuration = minDuration
	}
	
	if resetDuration > 0 {
		od.resetDuration = resetDuration
	}
}

// AddSample adds a new signal sample and checks for oscillation
// Returns true if oscillation is detected
func (od *OscillationDetector) AddSample(controlSignal, measuredValue float64) bool {
	od.lock.Lock()
	defer od.lock.Unlock()
	
	// If override is active and expired, clear it
	if !od.overrideUntil.IsZero() && time.Now().After(od.overrideUntil) {
		od.overrideUntil = time.Time{}
	}
	
	// Check for auto-reset
	if od.isTripped && time.Since(od.tripTime) > od.resetDuration {
		od.isTripped = false
	}
	
	// If already tripped and not in override, just return
	if od.isTripped && od.overrideUntil.IsZero() {
		return true
	}
	
	// Add sample to history
	now := time.Now()
	od.signalHistory = append(od.signalHistory, controlSignal)
	od.valueHistory = append(od.valueHistory, measuredValue)
	od.signalTimeHistory = append(od.signalTimeHistory, now)
	
	// Trim history to window size
	if len(od.signalHistory) > od.sampleWindow {
		od.signalHistory = od.signalHistory[1:]
		od.valueHistory = od.valueHistory[1:]
		od.signalTimeHistory = od.signalTimeHistory[1:]
	}
	
	// Need at least a few samples to detect oscillation
	if len(od.signalHistory) < 4 {
		return false
	}
	
	// Check for oscillation
	oscillating := od.detectOscillation()
	
	// Check minimum duration
	if oscillating {
		windowDuration := now.Sub(od.signalTimeHistory[0])
		if windowDuration >= od.minDuration {
			od.isTripped = true
			od.tripTime = time.Now()
			return true
		}
	}
	
	return false
}

// detectOscillation checks if the signal is oscillating
// by counting zero crossings and analyzing the pattern
func (od *OscillationDetector) detectOscillation() bool {
	if len(od.signalHistory) < 4 {
		return false
	}
	
	// Count zero crossings (sign changes)
	zeroCrossings := 0
	significantSignals := 0
	
	for i := 1; i < len(od.signalHistory); i++ {
		// Check if signal has significant magnitude
		if math.Abs(od.signalHistory[i]) > od.minSignalMagnitude {
			significantSignals++
			
			// Count sign changes
			if (od.signalHistory[i-1] < 0 && od.signalHistory[i] > 0) || 
			   (od.signalHistory[i-1] > 0 && od.signalHistory[i] < 0) {
				zeroCrossings++
			}
		}
	}
	
	// If no significant signals, no oscillation
	if significantSignals < 3 {
		return false
	}
	
	// Calculate percentage of samples that show oscillation
	crossingPercentage := float64(zeroCrossings) / float64(len(od.signalHistory)-1) * 100
	
	// Check if the percentage exceeds the threshold
	return crossingPercentage >= od.oscillationThresholdPercent
}

// IsTripped returns true if the circuit breaker is tripped
func (od *OscillationDetector) IsTripped() bool {
	od.lock.RLock()
	defer od.lock.RUnlock()
	
	// Check for auto-reset
	if od.isTripped && time.Since(od.tripTime) > od.resetDuration {
		// Use a write lock to modify state
		od.lock.RUnlock()
		od.lock.Lock()
		od.isTripped = false
		result := od.isTripped
		od.lock.Unlock()
		return result
	}
	
	// Check for manual override
	if !od.overrideUntil.IsZero() && time.Now().Before(od.overrideUntil) {
		return false
	}
	
	return od.isTripped
}

// Reset manually resets the circuit breaker
func (od *OscillationDetector) Reset() {
	od.lock.Lock()
	defer od.lock.Unlock()
	
	od.isTripped = false
	od.tripTime = time.Time{}
	od.signalHistory = make([]float64, 0, od.sampleWindow)
	od.valueHistory = make([]float64, 0, od.sampleWindow)
	od.signalTimeHistory = make([]time.Time, 0, od.sampleWindow)
}

// TemporaryOverride allows the controller to operate despite the circuit breaker
// for a specified duration. This is useful for manual interventions.
func (od *OscillationDetector) TemporaryOverride(duration time.Duration) {
	od.lock.Lock()
	defer od.lock.Unlock()
	
	od.overrideUntil = time.Now().Add(duration)
}

// GetStatus returns the current status of the oscillation detector
func (od *OscillationDetector) GetStatus() map[string]interface{} {
	od.lock.RLock()
	defer od.lock.RUnlock()
	
	var oscillationPercent float64
	var recentSignals, recentValues []float64
	
	if len(od.signalHistory) > 1 {
		// Count zero crossings
		zeroCrossings := 0
		for i := 1; i < len(od.signalHistory); i++ {
			if (od.signalHistory[i-1] < 0 && od.signalHistory[i] > 0) || 
			   (od.signalHistory[i-1] > 0 && od.signalHistory[i] < 0) {
				zeroCrossings++
			}
		}
		
		oscillationPercent = float64(zeroCrossings) / float64(len(od.signalHistory)-1) * 100
		
		// Include last few samples
		numSamples := 5
		if len(od.signalHistory) < numSamples {
			numSamples = len(od.signalHistory)
		}
		recentSignals = od.signalHistory[len(od.signalHistory)-numSamples:]
		recentValues = od.valueHistory[len(od.valueHistory)-numSamples:]
	}
	
	timeSinceTrip := ""
	if !od.tripTime.IsZero() {
		timeSinceTrip = fmt.Sprintf("%.1fs", time.Since(od.tripTime).Seconds())
	}
	
	return map[string]interface{}{
		"tripped":             od.isTripped,
		"oscillation_percent": oscillationPercent,
		"threshold_percent":   od.oscillationThresholdPercent,
		"sample_count":        len(od.signalHistory),
		"window_size":         od.sampleWindow,
		"recent_signals":      recentSignals,
		"recent_values":       recentValues,
		"time_since_trip":     timeSinceTrip,
		"override_active":     !od.overrideUntil.IsZero() && time.Now().Before(od.overrideUntil),
	}
}