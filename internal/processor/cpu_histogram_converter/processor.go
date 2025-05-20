package cpu_histogram_converter

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

// cpuHistogramProcessor implements a processor that converts CPU time metrics to utilization histograms.
type cpuHistogramProcessor struct {
	logger             *zap.Logger
	config             *Config
	nextConsumer       consumer.Metrics
	metricsCollector   *metrics.UnifiedMetricsCollector
	utilizationValues  []float64
	
	// State for delta calculations
	lastCPUTimeByProcess map[string]processState
	lastStateFlushTime   time.Time
	stateFileLock        sync.Mutex
	
	lock sync.Mutex
}

// processState holds the state for a single process
type processState struct {
	lastCPUTime   float64
	lastTimestamp pcommon.Timestamp
}

// newProcessor creates a new cpu_histogram_converter processor
func newProcessor(config *Config, settings component.TelemetrySettings) (*cpuHistogramProcessor, error) {
	logger := settings.Logger
	
	// Auto-detect CPU count if not configured
	if config.HostCPUCount <= 0 {
		config.HostCPUCount = runtime.NumCPU()
		logger.Info("Auto-detected CPU count", zap.Int("cpu_count", config.HostCPUCount))
	}
	
	p := &cpuHistogramProcessor{
		logger:       logger,
		config:       config,
		utilizationValues: make([]float64, 0, 100), // Pre-allocate space for 100 processes
		lastCPUTimeByProcess: make(map[string]processState),
		lastStateFlushTime: time.Now(),
		metricsCollector: metrics.NewUnifiedMetricsCollector(logger),
	}
	
	// Initialize metrics collector
	p.metricsCollector.SetDefaultAttributes(map[string]string{
		"processor": typeStr,
	})
	
	// Load state from disk if state storage path is set
	if config.StateStoragePath != "" {
		if err := p.loadStateFromDisk(); err != nil {
			logger.Warn("Failed to load state from disk, starting with empty state", zap.Error(err))
		}
	}
	
	return p, nil
}

// processMetrics processes metrics to convert CPU time to CPU utilization histograms
func (p *cpuHistogramProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	
	// Reset utilization values for this batch
	p.utilizationValues = p.utilizationValues[:0]
	
	// Track metrics for this batch
	var processedCount, histogramCount int
	var earliestTimestamp, latestTimestamp pcommon.Timestamp
	startTime := time.Now()
	
	// Process each metric
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		
		// Skip if TopKOnly is enabled and resource is not in top-k
		if p.config.TopKOnly {
			isTopK := false
			if val, ok := rm.Resource().Attributes().Get("aemf.filter.included"); ok {
				isTopK = val.Str() == "true"
			}
			
			if !isTopK {
				continue
			}
		}
		
		// Get process identifier from resource
		processID := p.getProcessIdentifier(rm.Resource().Attributes())
		
		// Process each scope metrics
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			
			// Process each metric
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				
				// Only process CPU time metrics
				if metric.Name() != p.config.InputMetricName {
					continue
				}
				
				// Handle different metric types, but we expect sum
				if metric.Type() == pmetric.MetricTypeSum {
					dataPoints := metric.Sum().DataPoints()
					
					for l := 0; l < dataPoints.Len(); l++ {
						dp := dataPoints.At(l)
						timestamp := dp.Timestamp()
						cpuTime := dp.DoubleValue()
						
						// Update timestamp tracking
						if earliestTimestamp == 0 || timestamp < earliestTimestamp {
							earliestTimestamp = timestamp
						}
						if latestTimestamp == 0 || timestamp > latestTimestamp {
							latestTimestamp = timestamp
						}
						
						// Calculate CPU utilization from CPU time delta
						if state, ok := p.lastCPUTimeByProcess[processID]; ok {
							// Only calculate if we have previous data and timestamps are different
							if state.lastCPUTime > 0 && timestamp > state.lastTimestamp {
								// Calculate time delta in seconds
								timeDeltaNs := timestamp - state.lastTimestamp
								timeDeltaSec := float64(timeDeltaNs) / 1e9
								
								// Calculate CPU time delta
								cpuTimeDelta := cpuTime - state.lastCPUTime
								
								if timeDeltaSec > 0 && cpuTimeDelta >= 0 {
									// Calculate CPU utilization as percentage of one CPU core
									// CPU time is in seconds, so dividing by elapsed time gives
									// fraction of a CPU core, and multiplying by 100 gives percentage
									utilizationPercent := (cpuTimeDelta / timeDeltaSec) * 100.0
									
									// Add to values for histogram generation
									p.utilizationValues = append(p.utilizationValues, utilizationPercent)
									processedCount++
								}
							}
						}
						
						// Update last state for this process
						p.lastCPUTimeByProcess[processID] = processState{
							lastCPUTime:   cpuTime,
							lastTimestamp: timestamp,
						}
					}
				}
			}
		}
	}
	
	// Clean up old processes if we exceed our limit
	if len(p.lastCPUTimeByProcess) > p.config.MaxProcessesInMemory {
		p.evictOldProcesses()
	}
	
	// If we have utilization values, generate histograms
	if len(p.utilizationValues) > 0 {
		p.addCPUUtilizationHistogram(md, p.utilizationValues)
		histogramCount = 1
	}
	
	// Periodically flush state to disk if configured
	if p.config.StateStoragePath != "" {
		now := time.Now()
		if now.Sub(p.lastStateFlushTime) > time.Duration(p.config.StateFlushIntervalSeconds)*time.Second {
			go p.flushStateToDisk() // Run in background to avoid blocking
			p.lastStateFlushTime = now
		}
	}
	
	// Record metrics
	p.metricsCollector.AddGauge("phoenix.cpu_histogram.processes_tracked", "Number of processes being tracked", "count").
		WithValue(float64(len(p.lastCPUTimeByProcess)))
	
	p.metricsCollector.AddGauge("phoenix.cpu_histogram.processes_processed", "Number of processes processed in this batch", "count").
		WithValue(float64(processedCount))
	
	p.metricsCollector.AddGauge("phoenix.cpu_histogram.histograms_generated", "Number of histograms generated", "count").
		WithValue(float64(histogramCount))
	
	p.metricsCollector.AddGauge("phoenix.cpu_histogram.processing_time_ms", "Time taken to process batch", "ms").
		WithValue(float64(time.Since(startTime).Milliseconds()))
	
	// Emit metrics
	if err := p.metricsCollector.Emit(ctx); err != nil {
		p.logger.Warn("Failed to emit metrics", zap.Error(err))
	}
	
	return md, nil
}

// getProcessIdentifier creates a unique identifier for a process based on resource attributes
func (p *cpuHistogramProcessor) getProcessIdentifier(attributes pcommon.Map) string {
	var processName, processID string
	
	// Try to get process name
	attributes.Range(func(k string, v pcommon.Value) bool {
		if k == "process.executable.name" || k == "process.name" {
			if v.Type() == pcommon.ValueTypeStr {
				processName = v.Str()
			}
			return false // Stop iteration
		}
		return true // Continue iteration
	})
	
	// Try to get process ID
	attributes.Range(func(k string, v pcommon.Value) bool {
		if k == "process.pid" {
			if v.Type() == pcommon.ValueTypeInt {
				processID = fmt.Sprintf("%d", v.Int())
			} else if v.Type() == pcommon.ValueTypeStr {
				processID = v.Str()
			}
			return false // Stop iteration
		}
		return true // Continue iteration
	})
	
	// Combine to create a unique ID
	if processName != "" && processID != "" {
		return processName + ":" + processID
	} else if processName != "" {
		return processName
	} else if processID != "" {
		return "pid:" + processID
	}
	
	// Fallback to a hash of all attributes
	return "unknown"
}

// addCPUUtilizationHistogram adds a histogram metric for CPU utilization
func (p *cpuHistogramProcessor) addCPUUtilizationHistogram(md pmetric.Metrics, utilizationValues []float64) {
	// Sort utilization values for easier bucketing
	sort.Float64s(utilizationValues)
	
	// Get bucket boundaries from config
	boundaries := p.config.HistogramBuckets
	
	// Create histogram
	// Find resource metrics to add to
	var rm pmetric.ResourceMetrics
	if md.ResourceMetrics().Len() == 0 {
		rm = md.ResourceMetrics().AppendEmpty()
		// Add some attributes to identify this as a Phoenix internal metric
		rm.Resource().Attributes().PutStr("service.name", "phoenix")
		rm.Resource().Attributes().PutStr("processor", typeStr)
	} else {
		rm = md.ResourceMetrics().At(0)
	}
	
	// Find or create scope metrics for the histogram
	var sm pmetric.ScopeMetrics
	if rm.ScopeMetrics().Len() == 0 {
		sm = rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("phoenix")
	} else {
		sm = rm.ScopeMetrics().At(0)
	}
	
	// Create histogram metric
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(p.config.OutputMetricName)
	metric.SetDescription("Distribution of CPU utilization across processes")
	metric.SetUnit("%") // Percentage
	
	histogram := metric.SetEmptyHistogram()
	histogram.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	
	// Create data point
	dp := histogram.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-time.Duration(p.config.CollectionIntervalSeconds) * time.Second)))
	
	// Set boundaries
	dp.ExplicitBounds().FromRaw(boundaries)
	
	// Calculate counts for each bucket
	counts := make([]uint64, len(boundaries)+1)
	currentBucket := 0
	
	// Count each value in appropriate bucket
	for _, val := range utilizationValues {
		// Find bucket for this value
		for currentBucket < len(boundaries) && val > boundaries[currentBucket] {
			currentBucket++
		}
		counts[currentBucket]++
	}
	
	// Set counts and summary statistics
	dp.BucketCounts().FromRaw(counts)
	dp.SetCount(uint64(len(utilizationValues)))
	
	// Calculate sum (total utilization across all processes)
	var sum float64
	for _, val := range utilizationValues {
		sum += val
	}
	dp.SetSum(sum)
}

// evictOldProcesses removes the oldest processes from tracking when we exceed our memory limits
func (p *cpuHistogramProcessor) evictOldProcesses() {
	// If we have fewer processes than max, nothing to do
	if len(p.lastCPUTimeByProcess) <= p.config.MaxProcessesInMemory {
		return
	}
	
	// Calculate how many to remove
	toRemove := len(p.lastCPUTimeByProcess) - p.config.MaxProcessesInMemory
	
	// Create slice of processes sorted by timestamp
	type processWithTimestamp struct {
		id         string
		timestamp  pcommon.Timestamp
	}
	processes := make([]processWithTimestamp, 0, len(p.lastCPUTimeByProcess))
	
	for id, state := range p.lastCPUTimeByProcess {
		processes = append(processes, processWithTimestamp{
			id:        id,
			timestamp: state.lastTimestamp,
		})
	}
	
	// Sort by timestamp (oldest first)
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].timestamp < processes[j].timestamp
	})
	
	// Remove oldest processes
	for i := 0; i < toRemove && i < len(processes); i++ {
		delete(p.lastCPUTimeByProcess, processes[i].id)
	}
	
	p.logger.Info("Evicted old processes from tracking",
		zap.Int("evicted_count", toRemove),
		zap.Int("remaining_count", len(p.lastCPUTimeByProcess)))
}

// flushStateToDisk saves the current process state to disk
func (p *cpuHistogramProcessor) flushStateToDisk() {
	p.stateFileLock.Lock()
	defer p.stateFileLock.Unlock()
	
	if p.config.StateStoragePath == "" {
		return
	}
	
	// Ensure directory exists
	dir := filepath.Dir(p.config.StateStoragePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		p.logger.Error("Failed to create state directory", zap.Error(err))
		return
	}
	
	// Lock to get consistent snapshot
	p.lock.Lock()
	
	// Prepare serializable state
	type serializedState struct {
		LastCPUTime   float64 `json:"last_cpu_time"`
		LastTimestamp int64   `json:"last_timestamp"`
	}
	
	state := make(map[string]serializedState)
	for id, processState := range p.lastCPUTimeByProcess {
		state[id] = serializedState{
			LastCPUTime:   processState.lastCPUTime,
			LastTimestamp: int64(processState.lastTimestamp),
		}
	}
	p.lock.Unlock()
	
	// Serialize
	data, err := json.Marshal(state)
	if err != nil {
		p.logger.Error("Failed to serialize state", zap.Error(err))
		return
	}
	
	// Write to temp file first, then rename for atomicity
	tempFile := p.config.StateStoragePath + ".tmp"
	if err := ioutil.WriteFile(tempFile, data, 0644); err != nil {
		p.logger.Error("Failed to write state to temp file", zap.Error(err))
		return
	}
	
	// Rename
	if err := os.Rename(tempFile, p.config.StateStoragePath); err != nil {
		p.logger.Error("Failed to rename temp state file", zap.Error(err))
		return
	}
	
	p.logger.Info("Successfully flushed state to disk",
		zap.String("path", p.config.StateStoragePath),
		zap.Int("process_count", len(state)))
}

// loadStateFromDisk loads process state from disk
func (p *cpuHistogramProcessor) loadStateFromDisk() error {
	p.stateFileLock.Lock()
	defer p.stateFileLock.Unlock()
	
	if p.config.StateStoragePath == "" {
		return fmt.Errorf("state storage path not configured")
	}
	
	// Check if file exists
	if _, err := os.Stat(p.config.StateStoragePath); os.IsNotExist(err) {
		return fmt.Errorf("state file does not exist: %s", p.config.StateStoragePath)
	}
	
	// Read file
	data, err := ioutil.ReadFile(p.config.StateStoragePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %v", err)
	}
	
	// Deserialize
	type serializedState struct {
		LastCPUTime   float64 `json:"last_cpu_time"`
		LastTimestamp int64   `json:"last_timestamp"`
	}
	
	state := make(map[string]serializedState)
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to deserialize state: %v", err)
	}
	
	// Load into processor state
	p.lock.Lock()
	defer p.lock.Unlock()
	
	p.lastCPUTimeByProcess = make(map[string]processState, len(state))
	for id, serialized := range state {
		p.lastCPUTimeByProcess[id] = processState{
			lastCPUTime:   serialized.LastCPUTime,
			lastTimestamp: pcommon.Timestamp(serialized.LastTimestamp),
		}
	}
	
	p.logger.Info("Successfully loaded state from disk",
		zap.String("path", p.config.StateStoragePath),
		zap.Int("process_count", len(state)))
	
	return nil
}