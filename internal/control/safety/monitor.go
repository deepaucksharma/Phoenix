// Package safety provides mechanisms for monitoring system resources and
// activating safety measures when thresholds are exceeded.
package safety

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"go.uber.org/zap"
)

// SafetyLevel defines different levels of safety mode
type SafetyLevel int

const (
	// SafetyLevelNormal indicates normal operation
	SafetyLevelNormal SafetyLevel = iota

	// SafetyLevelWarning indicates approaching resource limits
	SafetyLevelWarning

	// SafetyLevelCritical indicates critical resource constraints
	SafetyLevelCritical

	// SafetyLevelEmergency indicates severe resource exhaustion requiring immediate action
	SafetyLevelEmergency
)

// MonitorConfig contains configuration for the safety monitor
type MonitorConfig struct {
	// CPU thresholds in millicores (1000 = 1 core)
	CPUWarningThresholdMCores   int `mapstructure:"cpu_warning_threshold_mcores"`
	CPUCriticalThresholdMCores  int `mapstructure:"cpu_critical_threshold_mcores"`
	CPUEmergencyThresholdMCores int `mapstructure:"cpu_emergency_threshold_mcores"`

	// Memory thresholds in MiB
	MemoryWarningThresholdMiB   int `mapstructure:"memory_warning_threshold_mib"`
	MemoryCriticalThresholdMiB  int `mapstructure:"memory_critical_threshold_mib"`
	MemoryEmergencyThresholdMiB int `mapstructure:"memory_emergency_threshold_mib"`

	// Monitoring interval
	MonitoringIntervalSeconds int `mapstructure:"monitoring_interval_seconds"`

	// Recovery configuration
	RecoveryThresholdMultiplier float64 `mapstructure:"recovery_threshold_multiplier"`
	RecoveryTimeSeconds         int     `mapstructure:"recovery_time_seconds"`
}

// GetDefaultConfig returns the default configuration for the safety monitor
func GetDefaultConfig() *MonitorConfig {
	return &MonitorConfig{
		CPUWarningThresholdMCores:   800, // 80% of 1 core
		CPUCriticalThresholdMCores:  950, // 95% of 1 core
		CPUEmergencyThresholdMCores: 980, // 98% of 1 core

		MemoryWarningThresholdMiB:   256, // 256 MiB
		MemoryCriticalThresholdMiB:  384, // 384 MiB
		MemoryEmergencyThresholdMiB: 448, // 448 MiB

		MonitoringIntervalSeconds: 5, // Check every 5 seconds

		RecoveryThresholdMultiplier: 0.8, // Recover when below 80% of threshold
		RecoveryTimeSeconds:         30,  // Must be below threshold for 30 seconds
	}
}

// SafetyMonitor tracks system resources and provides safety level information
type SafetyMonitor struct {
	config         *MonitorConfig
	logger         *zap.Logger
	proc           *process.Process
	currentLevel   SafetyLevel
	lock           sync.RWMutex
	stopCh         chan struct{}
	levelChangedCh chan SafetyLevel

	// Recovery tracking
	recoveryStartTime time.Time

	// Current usage metrics
	currentCPUMCores int
	currentMemoryMiB int
}

// NewSafetyMonitor creates a new safety monitor with the given configuration
func NewSafetyMonitor(config *MonitorConfig, logger *zap.Logger) *SafetyMonitor {
	if config == nil {
		config = GetDefaultConfig()
	}

	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		if logger != nil {
			logger.Warn("Failed to initialize process handle", zap.Error(err))
		}
	}

	return &SafetyMonitor{
		config:         config,
		logger:         logger,
		proc:           proc,
		currentLevel:   SafetyLevelNormal,
		stopCh:         make(chan struct{}),
		levelChangedCh: make(chan SafetyLevel, 10),
	}
}

// Start begins monitoring system resources
func (sm *SafetyMonitor) Start(ctx context.Context) {
	interval := time.Duration(sm.config.MonitoringIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sm.logger.Info("Safety monitor started",
		zap.Int("interval_seconds", sm.config.MonitoringIntervalSeconds))

	for {
		select {
		case <-ticker.C:
			sm.checkResources()
		case <-ctx.Done():
			sm.logger.Info("Safety monitor stopping due to context cancellation")
			close(sm.stopCh)
			return
		case <-sm.stopCh:
			sm.logger.Info("Safety monitor stopped")
			return
		}
	}
}

// Stop stops the safety monitor
func (sm *SafetyMonitor) Stop() {
	close(sm.stopCh)
}

// GetCurrentLevel returns the current safety level
func (sm *SafetyMonitor) GetCurrentLevel() SafetyLevel {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	return sm.currentLevel
}

// GetCurrentUsage returns the current CPU and memory usage
func (sm *SafetyMonitor) GetCurrentUsage() (cpuMCores int, memoryMiB int) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	return sm.currentCPUMCores, sm.currentMemoryMiB
}

// Subscribe returns a channel that receives notifications when the safety level changes
func (sm *SafetyMonitor) Subscribe() <-chan SafetyLevel {
	return sm.levelChangedCh
}

// checkResources checks current resource usage and updates safety level if needed
func (sm *SafetyMonitor) checkResources() {
	// Get current memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Convert to MiB
	currentMemoryMiB := int(memStats.Alloc / (1024 * 1024))

	// Get CPU usage for the current process
	currentCPUMCores := 0
	if sm.proc != nil {
		cpuPercent, err := sm.proc.Percent(0)
		if err != nil {
			sm.logger.Warn("failed to read CPU usage", zap.Error(err))
		} else {
			// Convert percentage to millicores (100% = 1000 mcores)
			currentCPUMCores = int(cpuPercent * 10)
		}
	}

	sm.lock.Lock()

	// Update current metrics
	sm.currentCPUMCores = currentCPUMCores
	sm.currentMemoryMiB = currentMemoryMiB

	// Determine safety level based on CPU and memory
	var newLevel SafetyLevel

	// Check CPU
	if currentCPUMCores >= sm.config.CPUEmergencyThresholdMCores {
		newLevel = SafetyLevelEmergency
	} else if currentCPUMCores >= sm.config.CPUCriticalThresholdMCores {
		newLevel = SafetyLevelCritical
	} else if currentCPUMCores >= sm.config.CPUWarningThresholdMCores {
		newLevel = SafetyLevelWarning
	} else {
		newLevel = SafetyLevelNormal
	}

	// Check memory (take the higher safety level)
	if currentMemoryMiB >= sm.config.MemoryEmergencyThresholdMiB {
		if newLevel < SafetyLevelEmergency {
			newLevel = SafetyLevelEmergency
		}
	} else if currentMemoryMiB >= sm.config.MemoryCriticalThresholdMiB {
		if newLevel < SafetyLevelCritical {
			newLevel = SafetyLevelCritical
		}
	} else if currentMemoryMiB >= sm.config.MemoryWarningThresholdMiB {
		if newLevel < SafetyLevelWarning {
			newLevel = SafetyLevelWarning
		}
	}

	// Check for recovery conditions
	if sm.currentLevel > SafetyLevelNormal && newLevel < sm.currentLevel {
		recoveryThresholdCPU := int(float64(sm.config.CPUWarningThresholdMCores) *
			sm.config.RecoveryThresholdMultiplier)
		recoveryThresholdMem := int(float64(sm.config.MemoryWarningThresholdMiB) *
			sm.config.RecoveryThresholdMultiplier)

		// Check if we're below recovery thresholds
		if currentCPUMCores < recoveryThresholdCPU &&
			currentMemoryMiB < recoveryThresholdMem {

			// If we just started recovery, record the time
			if sm.recoveryStartTime.IsZero() {
				sm.recoveryStartTime = time.Now()
			} else if time.Since(sm.recoveryStartTime) >=
				time.Duration(sm.config.RecoveryTimeSeconds)*time.Second {
				// We've been below threshold for the required recovery time
				sm.logger.Info("Resource usage recovered, returning to normal mode",
					zap.Int("previous_level", int(sm.currentLevel)),
					zap.Int("new_level", int(newLevel)),
					zap.Int("cpu_mcores", currentCPUMCores),
					zap.Int("memory_mib", currentMemoryMiB))

				sm.currentLevel = newLevel

				// Reset recovery start time
				sm.recoveryStartTime = time.Time{}

				// Notify subscribers
				select {
				case sm.levelChangedCh <- newLevel:
					// Notification sent
				default:
					// Channel buffer is full, log and continue
					sm.logger.Warn("Couldn't notify subscribers of safety level change, channel full")
				}
			}
		} else {
			// We went back above threshold during recovery, reset timer
			sm.recoveryStartTime = time.Time{}
		}
	} else if newLevel > sm.currentLevel {
		// Immediate escalation
		sm.logger.Warn("Resource usage exceeded threshold, increasing safety level",
			zap.Int("previous_level", int(sm.currentLevel)),
			zap.Int("new_level", int(newLevel)),
			zap.Int("cpu_mcores", currentCPUMCores),
			zap.Int("memory_mib", currentMemoryMiB))

		sm.currentLevel = newLevel

		// Reset recovery start time on escalation
		sm.recoveryStartTime = time.Time{}

		// Notify subscribers
		select {
		case sm.levelChangedCh <- newLevel:
			// Notification sent
		default:
			// Channel buffer is full, log and continue
			sm.logger.Warn("Couldn't notify subscribers of safety level change, channel full")
		}
	}

	sm.lock.Unlock()
}

// IsInSafeMode returns true if the monitor is in any non-normal safety level
func (sm *SafetyMonitor) IsInSafeMode() bool {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	return sm.currentLevel > SafetyLevelNormal
}

// ForceLevel forces the safety monitor to a specific level (for testing or manual intervention)
func (sm *SafetyMonitor) ForceLevel(level SafetyLevel) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if level != sm.currentLevel {
		sm.logger.Info("Forcing safety level",
			zap.Int("previous_level", int(sm.currentLevel)),
			zap.Int("new_level", int(level)))

		sm.currentLevel = level

		// Reset recovery start time
		sm.recoveryStartTime = time.Time{}

		// Notify subscribers
		select {
		case sm.levelChangedCh <- level:
			// Notification sent
		default:
			// Channel buffer is full, log and continue
			sm.logger.Warn("Couldn't notify subscribers of safety level change, channel full")
		}
	}
}
