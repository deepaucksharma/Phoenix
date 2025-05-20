package timeseries_estimator

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/base"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
	"github.com/deepaucksharma/Phoenix/pkg/util/hll"
)

// timeseriesEstimatorProcessor estimates the number of unique time series in metrics data.
type timeseriesEstimatorProcessor struct {
	*base.UpdateableProcessor
	config             *Config
	logger             *zap.Logger
	nextConsumer       consumer.Metrics
	uniqueTimeSeries   map[string]struct{}
	hllEstimator       *hll.HyperLogLog
	lock               sync.RWMutex
	lastCleanupTime    time.Time
	lastEstimateValue  int64
	lastEstimateTime   time.Time
	isMemoryConstrained bool
	metricsCollector   *metrics.UnifiedMetricsCollector
}

// Ensure the processor implements required interfaces
var _ processor.Metrics = (*timeseriesEstimatorProcessor)(nil)
var _ interfaces.UpdateableProcessor = (*timeseriesEstimatorProcessor)(nil)

// newProcessor creates a new timeseries_estimator processor.
func newProcessor(config *Config, settings processor.Settings, nextConsumer consumer.Metrics) (*timeseriesEstimatorProcessor, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration for timeseries_estimator processor: %v", err)
	}

	// Create base processor
	baseProcessor := base.NewUpdateableProcessor(
		settings.TelemetrySettings.Logger,
		nextConsumer,
		typeStr,
		settings.ID,
		config,
	)

	// Create HLL estimator if using that algorithm
	var hllEstimator *hll.HyperLogLog
	if config.EstimatorType == "hll" {
		var err error
		hllEstimator, err = hll.New(uint8(config.HLLPrecision))
		if err != nil {
			return nil, fmt.Errorf("failed to create HyperLogLog estimator: %v", err)
		}
	}

	return &timeseriesEstimatorProcessor{
		UpdateableProcessor: baseProcessor,
		config:              config,
		logger:              settings.TelemetrySettings.Logger,
		nextConsumer:        nextConsumer,
		uniqueTimeSeries:    make(map[string]struct{}),
		hllEstimator:        hllEstimator,
		lastCleanupTime:     time.Now(),
		lastEstimateValue:   0,
		lastEstimateTime:    time.Now(),
		isMemoryConstrained: false,
		metricsCollector:    metrics.NewUnifiedMetricsCollector(settings.TelemetrySettings.Logger),
	}, nil
}

// Start implements the component.Component interface.
func (p *timeseriesEstimatorProcessor) Start(ctx context.Context, host component.Host) error {
	p.initMetrics()
	return p.UpdateableProcessor.Start(ctx, host)
}

// Shutdown implements the component.Component interface.
func (p *timeseriesEstimatorProcessor) Shutdown(ctx context.Context) error {
	// Emit final metrics before shutdown
	if err := p.metricsCollector.Emit(ctx); err != nil {
		p.logger.Warn("Failed to emit final metrics during shutdown", zap.Error(err))
	}
	
	return p.UpdateableProcessor.Shutdown(ctx)
}

// initMetrics initializes the metrics that this processor will emit
func (p *timeseriesEstimatorProcessor) initMetrics() {
	p.metricsCollector.SetDefaultAttributes(map[string]string{
		"processor": typeStr,
		"processor_id": p.ComponentID().String(),
	})

	// Register metrics
	p.metricsCollector.AddGauge(
		"phoenix.timeseries.estimate",
		"Estimated number of unique time series",
		"count",
	)
	
	p.metricsCollector.AddGauge(
		"phoenix.timeseries.memory_usage_mb",
		"Memory usage of the time series tracker in MB",
		"MB",
	)
	
	p.metricsCollector.AddGauge(
		"phoenix.timeseries.mode",
		"Current operating mode (0=exact, 1=hll)",
		"",
	).WithValue(func() float64 {
		if p.config.EstimatorType == "exact" {
			return 0
		}
		return 1
	}())
	
	p.metricsCollector.AddGauge(
		"phoenix.timeseries.memory_constrained",
		"Indicates if memory usage is constrained (0=no, 1=yes)",
		"",
	)
}

// emitMetrics emits processor metrics
func (p *timeseriesEstimatorProcessor) emitMetrics(ctx context.Context) {
	// Update memory usage metric
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// Convert to MB
	memUsageMB := float64(memStats.Alloc) / 1024.0 / 1024.0
	
	// Update memory constrained status
	p.isMemoryConstrained = memUsageMB > float64(p.config.MemoryLimitMB)
	
	// Emit metrics
	p.metricsCollector.AddGauge("phoenix.timeseries.estimate", "", "").
		WithValue(float64(p.lastEstimateValue))
	
	p.metricsCollector.AddGauge("phoenix.timeseries.memory_usage_mb", "", "").
		WithValue(memUsageMB)
	
	p.metricsCollector.AddGauge("phoenix.timeseries.memory_constrained", "", "").
		WithValue(func() float64 {
			if p.isMemoryConstrained {
				return 1
			}
			return 0
		}())
	
	// Send metrics
	if err := p.metricsCollector.Emit(ctx); err != nil {
		p.logger.Warn("Failed to emit metrics", zap.Error(err))
	}
}

// countTimeSeries adds a time series to the uniqueness tracker
func (p *timeseriesEstimatorProcessor) countTimeSeries(metricName string, resource pcommon.Resource, attributes pcommon.Map) {
	// Create a unique identifier for this time series
	// Format: metricName + resource attributes hash + datapoint attributes hash
	var tsID string
	
	// If memory is constrained and we're in exact mode, use HLL instead to save memory
	if p.isMemoryConstrained && p.config.EstimatorType == "exact" && p.hllEstimator != nil {
		// Just calculate the ID and add it to HLL
		resourceStr := hashResource(resource)
		attrStr := hashAttributes(attributes)
		tsID = fmt.Sprintf("%s|%s|%s", metricName, resourceStr, attrStr)
		
		// Add to HLL instead of map to save memory
		p.hllEstimator.AddString(tsID)
		return
	}
	
	// Normal processing based on configured estimator type
	if p.config.EstimatorType == "exact" {
		resourceStr := hashResource(resource)
		attrStr := hashAttributes(attributes)
		tsID = fmt.Sprintf("%s|%s|%s", metricName, resourceStr, attrStr)
		
		// Add to exact counter
		p.uniqueTimeSeries[tsID] = struct{}{}
	} else if p.config.EstimatorType == "hll" {
		// For HLL, we still need to generate the ID string
		resourceStr := hashResource(resource)
		attrStr := hashAttributes(attributes)
		tsID = fmt.Sprintf("%s|%s|%s", metricName, resourceStr, attrStr)
		
		// Add to HLL
		p.hllEstimator.AddString(tsID)
	}
}

// processDataPoints processes numeric data points for Gauge and Sum metrics
func (p *timeseriesEstimatorProcessor) processDataPoints(metricName string, resource pcommon.Resource, dataPoints pmetric.NumberDataPointSlice) {
	for i := 0; i < dataPoints.Len(); i++ {
		dataPoint := dataPoints.At(i)
		p.countTimeSeries(metricName, resource, dataPoint.Attributes())
	}
}

// processHistogramDataPoints processes histogram data points
func (p *timeseriesEstimatorProcessor) processHistogramDataPoints(metricName string, resource pcommon.Resource, dataPoints pmetric.HistogramDataPointSlice) {
	for i := 0; i < dataPoints.Len(); i++ {
		dataPoint := dataPoints.At(i)
		p.countTimeSeries(metricName, resource, dataPoint.Attributes())
	}
}

// processSummaryDataPoints processes summary data points
func (p *timeseriesEstimatorProcessor) processSummaryDataPoints(metricName string, resource pcommon.Resource, dataPoints pmetric.SummaryDataPointSlice) {
	for i := 0; i < dataPoints.Len(); i++ {
		dataPoint := dataPoints.At(i)
		p.countTimeSeries(metricName, resource, dataPoint.Attributes())
	}
}

// processExponentialHistogramDataPoints processes exponential histogram data points
func (p *timeseriesEstimatorProcessor) processExponentialHistogramDataPoints(metricName string, resource pcommon.Resource, dataPoints pmetric.ExponentialHistogramDataPointSlice) {
	for i := 0; i < dataPoints.Len(); i++ {
		dataPoint := dataPoints.At(i)
		p.countTimeSeries(metricName, resource, dataPoint.Attributes())
	}
}

// processMetrics processes metrics data to count unique time series
func (p *timeseriesEstimatorProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// If not enabled, just pass through
	if !p.config.IsEnabled() {
		return md, nil
	}

	// Check if we should do periodic cleanup/reset
	now := time.Now()
	if now.Sub(p.lastCleanupTime) > p.config.RefreshInterval {
		p.logger.Debug("Performing periodic cleanup of time series estimator",
			zap.String("processor", "timeseries_estimator"),
			zap.Time("last_cleanup", p.lastCleanupTime),
			zap.Duration("interval", p.config.RefreshInterval))
		
		// Reset counters
		if p.config.EstimatorType == "exact" {
			p.uniqueTimeSeries = make(map[string]struct{})
		} else if p.config.EstimatorType == "hll" {
			p.hllEstimator.Reset()
		}
		
		p.lastCleanupTime = now
	}

	// Check memory usage before processing
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memUsageMB := float64(memStats.Alloc) / 1024.0 / 1024.0
	p.isMemoryConstrained = memUsageMB > float64(p.config.MemoryLimitMB)
	
	if p.isMemoryConstrained {
		p.logger.Warn("Memory usage exceeds configured limit, using HyperLogLog as fallback",
			zap.Float64("memory_usage_mb", memUsageMB),
			zap.Int("memory_limit_mb", p.config.MemoryLimitMB))
		
		// Initialize HLL if needed as fallback
		if p.config.EstimatorType == "exact" && p.hllEstimator == nil {
			var err error
			p.hllEstimator, err = hll.New(uint8(p.config.HLLPrecision))
			if err != nil {
				p.logger.Error("Failed to create HyperLogLog fallback, memory usage may be high",
					zap.Error(err))
			}
		}
	}

	// Process each metric to identify unique time series
	resourceMetrics := md.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		resourceMetric := resourceMetrics.At(i)
		resource := resourceMetric.Resource()
		
		scopeMetrics := resourceMetric.ScopeMetrics()
		for j := 0; j < scopeMetrics.Len(); j++ {
			scopeMetric := scopeMetrics.At(j)
			
			metricSlice := scopeMetric.Metrics()
			for k := 0; k < metricSlice.Len(); k++ {
				metric := metricSlice.At(k)
				metricName := metric.Name()
				
				// Skip processing our own output metric to avoid recursion
				if metricName == p.config.OutputMetricName {
					continue
				}
				
				// Process based on metric type
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					p.processDataPoints(metricName, resource, metric.Gauge().DataPoints())
				case pmetric.MetricTypeSum:
					p.processDataPoints(metricName, resource, metric.Sum().DataPoints())
				case pmetric.MetricTypeHistogram:
					p.processHistogramDataPoints(metricName, resource, metric.Histogram().DataPoints())
				case pmetric.MetricTypeSummary:
					p.processSummaryDataPoints(metricName, resource, metric.Summary().DataPoints())
				case pmetric.MetricTypeExponentialHistogram:
					p.processExponentialHistogramDataPoints(metricName, resource, metric.ExponentialHistogram().DataPoints())
				}
			}
		}
	}
	
	// Add our estimate metric
	var estimateValue int64
	if p.config.EstimatorType == "exact" && !p.isMemoryConstrained {
		estimateValue = int64(len(p.uniqueTimeSeries))
	} else if p.hllEstimator != nil {
		estimateValue = int64(p.hllEstimator.Count())
	}
	
	p.lastEstimateValue = estimateValue
	p.lastEstimateTime = time.Now()
	
	p.addEstimateMetric(md, estimateValue)
	
	// Emit self-monitoring metrics
	p.emitMetrics(ctx)
	
	p.logger.Debug("Processed metric batch",
		zap.String("processor", "timeseries_estimator"),
		zap.Int64("estimate", estimateValue),
		zap.Bool("memory_constrained", p.isMemoryConstrained))

	return md, nil
}

// addEstimateMetric adds a metric with the current time series estimate
func (p *timeseriesEstimatorProcessor) addEstimateMetric(md pmetric.Metrics, estimate int64) {
	// Create a new resource metric or use existing one
	var rm pmetric.ResourceMetrics
	if md.ResourceMetrics().Len() == 0 {
		rm = md.ResourceMetrics().AppendEmpty()
		// Add some attributes to identify this as a Phoenix internal metric
		rm.Resource().Attributes().PutStr("service.name", "phoenix")
		rm.Resource().Attributes().PutStr("phoenix.processor", typeStr)
	} else {
		rm = md.ResourceMetrics().At(0)
	}
	
	// Find or create a scope metric for the estimate
	var scopeMetric pmetric.ScopeMetrics
	if rm.ScopeMetrics().Len() == 0 {
		scopeMetric = rm.ScopeMetrics().AppendEmpty()
		scopeMetric.Scope().SetName("phoenix")
	} else {
		scopeMetric = rm.ScopeMetrics().At(0)
	}
	
	// Create our gauge metric
	metricSlice := scopeMetric.Metrics()
	metric := metricSlice.AppendEmpty()
	metric.SetName(p.config.OutputMetricName)
	metric.SetDescription("Estimated number of active time series being produced by Phoenix")
	metric.SetUnit("1") // Count/dimensionless
	
	gauge := metric.SetEmptyGauge()
	dataPoint := gauge.DataPoints().AppendEmpty()
	dataPoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dataPoint.SetIntValue(estimate)
}

// ConsumeMetrics implements the consumer.Metrics interface
func (p *timeseriesEstimatorProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	processedMetrics, err := p.processMetrics(ctx, md)
	if err != nil {
		return err
	}
	
	return p.nextConsumer.ConsumeMetrics(ctx, processedMetrics)
}

// Capabilities implements the processor.Metrics interface
func (p *timeseriesEstimatorProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// OnConfigPatch implements the interfaces.UpdateableProcessor interface
func (p *timeseriesEstimatorProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	
	switch patch.ParameterPath {
	case "enabled":
		if v, ok := patch.NewValue.(bool); ok {
			p.config.Enabled = v
			p.logger.Info("Updated enabled setting",
				zap.Bool("enabled", v))
			return nil
		}
		return fmt.Errorf("invalid type for enabled")
		
	case "estimator_type":
		if v, ok := patch.NewValue.(string); ok {
			if v != "exact" && v != "hll" {
				return fmt.Errorf("estimator_type must be either 'exact' or 'hll'")
			}
			
			// If switching from exact to HLL, make sure we have an HLL estimator
			if v == "hll" && p.config.EstimatorType == "exact" {
				var err error
				p.hllEstimator, err = hll.New(uint8(p.config.HLLPrecision))
				if err != nil {
					return fmt.Errorf("failed to create HyperLogLog estimator: %v", err)
				}
				
				// Reset map to free memory
				p.uniqueTimeSeries = make(map[string]struct{})
			}
			
			// If switching from HLL to exact, initialize map
			if v == "exact" && p.config.EstimatorType == "hll" {
				p.uniqueTimeSeries = make(map[string]struct{})
			}
			
			p.config.EstimatorType = v
			p.logger.Info("Updated estimator type",
				zap.String("estimator_type", v))
			return nil
		}
		return fmt.Errorf("invalid type for estimator_type")
		
	case "hll_precision":
		if v, ok := patch.NewValue.(int); ok {
			if v < 4 || v > 16 {
				return fmt.Errorf("hll_precision must be between 4 and 16")
			}
			
			// Only update if precision is different
			if v != p.config.HLLPrecision {
				p.config.HLLPrecision = v
				
				// If using HLL, recreate with new precision
				if p.config.EstimatorType == "hll" || p.isMemoryConstrained {
					var err error
					p.hllEstimator, err = hll.New(uint8(v))
					if err != nil {
						return fmt.Errorf("failed to create HyperLogLog estimator: %v", err)
					}
				}
				
				p.logger.Info("Updated HLL precision",
					zap.Int("hll_precision", v))
			}
			return nil
		}
		return fmt.Errorf("invalid type for hll_precision")
		
	case "memory_limit_mb":
		if v, ok := patch.NewValue.(int); ok {
			if v <= 0 {
				return fmt.Errorf("memory_limit_mb must be greater than 0")
			}
			
			p.config.MemoryLimitMB = v
			p.logger.Info("Updated memory limit",
				zap.Int("memory_limit_mb", v))
			
			// Check if we're now constrained
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			memUsageMB := float64(memStats.Alloc) / 1024.0 / 1024.0
			p.isMemoryConstrained = memUsageMB > float64(v)
			
			return nil
		}
		return fmt.Errorf("invalid type for memory_limit_mb")
		
	case "refresh_interval":
		if v, ok := patch.NewValue.(int); ok {
			if v <= 0 {
				return fmt.Errorf("refresh_interval must be greater than 0")
			}
			
			p.config.RefreshInterval = time.Duration(v) * time.Second
			p.logger.Info("Updated refresh interval",
				zap.Duration("refresh_interval", p.config.RefreshInterval))
			return nil
		}
		return fmt.Errorf("invalid type for refresh_interval")
		
	default:
		return fmt.Errorf("unknown parameter %s", patch.ParameterPath)
	}
}

// GetConfigStatus implements the interfaces.UpdateableProcessor interface
func (p *timeseriesEstimatorProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	
	return interfaces.ConfigStatus{
		Parameters: map[string]any{
			"estimator_type":    p.config.EstimatorType,
			"hll_precision":     p.config.HLLPrecision,
			"memory_limit_mb":   p.config.MemoryLimitMB,
			"refresh_interval":  p.config.RefreshInterval.Seconds(),
			"last_estimate":     p.lastEstimateValue,
			"memory_constrained": p.isMemoryConstrained,
		},
		Enabled: p.config.Enabled,
	}, nil
}

// Helper functions for hashing resources and attributes

// hashResource creates a string hash representation of a resource
func hashResource(resource pcommon.Resource) string {
	if resource.Attributes().Len() == 0 {
		return ""
	}
	
	// Sort keys for consistent hash
	keys := make([]string, 0, resource.Attributes().Len())
	resource.Attributes().Range(func(k string, v pcommon.Value) bool {
		keys = append(keys, k)
		return true
	})
	
	// Build hash string
	var result string
	for _, k := range keys {
		v, _ := resource.Attributes().Get(k)
		result += fmt.Sprintf("%s=%s;", k, v.AsString())
	}
	
	return result
}

// hashAttributes creates a string hash representation of attributes
func hashAttributes(attrs pcommon.Map) string {
	if attrs.Len() == 0 {
		return ""
	}
	
	// Sort keys for consistent hash
	keys := make([]string, 0, attrs.Len())
	attrs.Range(func(k string, v pcommon.Value) bool {
		keys = append(keys, k)
		return true
	})
	
	// Build hash string
	var result string
	for _, k := range keys {
		v, _ := attrs.Get(k)
		result += fmt.Sprintf("%s=%s;", k, v.AsString())
	}
	
	return result
}