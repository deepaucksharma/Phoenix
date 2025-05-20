// Package safety provides mechanisms for monitoring system resources and
// activating safety measures when thresholds are exceeded.
package safety

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Monitor monitors system resources and activates/deactivates safe mode
type Monitor struct {
	config      *Config
	telemetry   component.TelemetrySettings
	inSafeMode  bool
	lock        sync.RWMutex
	stopCh      chan struct{}
	metricsProvider MetricsProvider

	// Threshold values
	cpuThreshold int
	memThreshold int
	
	// Threshold override
	overrideActive bool
	overrideExpiry time.Time
	
	// Safe mode cooldown
	safeModeEnd time.Time
}

// NewMonitor creates a new safety monitor
func NewMonitor(config *Config, telemetry component.TelemetrySettings) *Monitor {
	monitor := &Monitor{
		config:      config,
		telemetry:   telemetry,
		inSafeMode:  false,
		stopCh:      make(chan struct{}),
		metricsProvider: &DefaultMetricsProvider{},
		cpuThreshold: config.CPUUsageThresholdMCores,
		memThreshold: config.MemoryThresholdMiB,
	}
	
	return monitor
}

// Start starts the safety monitor
func (m *Monitor) Start(ctx context.Context, host component.Host) error {
	// Start a goroutine to check metrics periodically
	go m.checkMetrics(ctx)
	return nil
}

// Shutdown stops the safety monitor
func (m *Monitor) Shutdown(ctx context.Context) error {
	close(m.stopCh)
	return nil
}

// SetMetricsProvider sets the metrics provider for testing
func (m *Monitor) SetMetricsProvider(provider MetricsProvider) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.metricsProvider = provider
}

// IsInSafeMode returns whether the system is in safe mode
func (m *Monitor) IsInSafeMode() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.inSafeMode
}

// TemporarilyOverrideThresholds temporarily increases safety thresholds
func (m *Monitor) TemporarilyOverrideThresholds(seconds int) {
	m.lock.Lock()
	defer m.lock.Unlock()
	
	// Increase thresholds by the configured multiplier
	m.cpuThreshold = int(float64(m.config.CPUUsageThresholdMCores) * m.config.OverrideMultiplier)
	m.memThreshold = int(float64(m.config.MemoryThresholdMiB) * m.config.OverrideMultiplier)
	
	// Set expiry time
	expirySeconds := seconds
	if expirySeconds <= 0 {
		expirySeconds = m.config.OverrideExpirySeconds
	}
	m.overrideExpiry = time.Now().Add(time.Duration(expirySeconds) * time.Second)
	m.overrideActive = true
}

// GetCurrentCPUThreshold returns the current CPU threshold
func (m *Monitor) GetCurrentCPUThreshold() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.cpuThreshold
}

// GetCurrentMemThreshold returns the current memory threshold
func (m *Monitor) GetCurrentMemThreshold() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.memThreshold
}

// checkMetrics periodically checks system metrics
func (m *Monitor) checkMetrics(ctx context.Context) {
	interval := time.Duration(m.config.MetricsCheckIntervalMs) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.checkAndUpdateState()
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		}
	}
}

// checkAndUpdateState checks metrics and updates system state
func (m *Monitor) checkAndUpdateState() {
	m.lock.Lock()
	defer m.lock.Unlock()
	
	// Check if threshold override has expired
	if m.overrideActive && time.Now().After(m.overrideExpiry) {
		// Reset to normal thresholds
		m.cpuThreshold = m.config.CPUUsageThresholdMCores
		m.memThreshold = m.config.MemoryThresholdMiB
		m.overrideActive = false
	}
	
	// Check if safe mode cooldown has ended
	if m.inSafeMode && !m.safeModeEnd.IsZero() && time.Now().After(m.safeModeEnd) {
		m.inSafeMode = false
		m.safeModeEnd = time.Time{}
	}
	
	// Get current metrics
	cpuUsage := m.metricsProvider.GetCPUUsage()
	memUsage := m.metricsProvider.GetMemoryUsage()
	
	// Check if thresholds are exceeded
	if cpuUsage > m.cpuThreshold || memUsage > m.memThreshold {
		// Enter safe mode
		m.inSafeMode = true
		m.safeModeEnd = time.Time{} // Clear end time while thresholds are still exceeded
	} else if m.inSafeMode && m.safeModeEnd.IsZero() {
		// Start cooldown period
		m.safeModeEnd = time.Now().Add(time.Duration(m.config.SafeModeCooldownSeconds) * time.Second)
	}
}