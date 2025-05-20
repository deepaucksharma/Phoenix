// Package metric_pipeline implements a combined processor that handles resource filtering
// and metric transformation in a single processing step for efficiency.
package metric_pipeline

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/base"
	"github.com/deepaucksharma/Phoenix/internal/processor/config"
	"github.com/deepaucksharma/Phoenix/internal/processor/resource_filter"
	"github.com/deepaucksharma/Phoenix/pkg/util/topk"
)

// processorImpl is the implementation of the metric_pipeline processor
type processorImpl struct {
	*base.UpdateableProcessor
	config           *Config
	priorityRules    []*regexp.Regexp
	topkAlgo         *topk.SpaceSaving
	topkSet          map[string]struct{}
	totalItems       int
	totalIncluded    int
	configManager    *config.Manager
	histogramBuckets map[string][]float64

	// Self-metrics
	metricsCollector *metrics.UnifiedMetricsCollector
	priorityCounts   map[string]int
	rollupResources  int
	histogramCount   int
}

// Ensure the processor implements required interfaces
var _ processor.Metrics = (*processorImpl)(nil)
var _ interfaces.UpdateableProcessor = (*processorImpl)(nil)

// newProcessor creates a new metric_pipeline processor
func newProcessor(cfg *Config, settings processor.Settings, nextConsumer consumer.Metrics) (*processorImpl, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Create the updateable processor
	up := base.NewUpdateableProcessor(
		settings.TelemetrySettings.Logger,
		nextConsumer,
		Type,
		settings.ID,
		cfg,
	)

	// Create histogram buckets map
	histogramBuckets := make(map[string][]float64)
	for metricName, histogramCfg := range cfg.Transformation.Histograms.Metrics {
		histogramBuckets[metricName] = histogramCfg.Boundaries
	}

	// Create the processor
	p := &processorImpl{
		UpdateableProcessor: up,
		config:              cfg,
		priorityRules:       make([]*regexp.Regexp, len(cfg.ResourceFilter.PriorityRules)),
		topkSet:             make(map[string]struct{}),
		histogramBuckets:    histogramBuckets,

		// Initialize self-metrics related fields
		metricsCollector: metrics.NewUnifiedMetricsCollector(settings.TelemetrySettings.Logger),
		priorityCounts:   make(map[string]int),
		rollupResources:  0,
		histogramCount:   0,
	}

	// Create configuration manager
	p.configManager = config.NewManager(settings.TelemetrySettings.Logger, p, cfg)

	// Set default attributes for metrics
	p.metricsCollector.SetDefaultAttributes(map[string]string{
		"processor":       Type,
		"processor_id":    settings.ID.String(),
		"filter_strategy": string(cfg.ResourceFilter.FilterStrategy),
	})

	// Initialize priority rules
	for i, rule := range cfg.ResourceFilter.PriorityRules {
		re, err := regexp.Compile(rule.Match)
		if err != nil {
			return nil, err
		}
		p.priorityRules[i] = re
	}

	// Initialize topk algorithm if needed
	if cfg.ResourceFilter.FilterStrategy == resource_filter.StrategyTopK ||
		cfg.ResourceFilter.FilterStrategy == resource_filter.StrategyHybrid {
		p.topkAlgo = topk.NewSpaceSaving(cfg.ResourceFilter.TopK.KValue)
	}

	return p, nil
}

// ConsumeMetrics implements the consumer.Metrics interface
func (p *processorImpl) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.Lock()
	defer p.Unlock()

	// Reset per-batch metric counters
	p.priorityCounts = make(map[string]int)
	p.rollupResources = 0
	p.histogramCount = 0

	// Track processing time
	startTime := time.Now()

	// Process in stages, but within a single processor call for efficiency

	// Stage 1: Apply resource filtering
	filteredMetrics, err := p.applyResourceFiltering(md)
	if err != nil {
		p.GetLogger().Error("Error applying resource filtering", zap.Error(err))
		filteredMetrics = md // Continue with original metrics on error
	}

	// Stage 2: Apply metric transformations
	transformedMetrics, err := p.applyMetricTransformations(filteredMetrics)
	if err != nil {
		p.GetLogger().Error("Error applying metric transformations", zap.Error(err))
		transformedMetrics = filteredMetrics // Continue with filtered metrics on error
	}

	// Calculate and record processing duration
	duration := time.Since(startTime).Milliseconds()
	p.metricsCollector.AddGauge("phoenix.processing.duration_ms", "", "").
		WithValue(float64(duration))

	// Emit metrics collected during processing
	p.emitMetrics(ctx)

	// Pass the processed metrics to the next consumer
	return p.GetNext().ConsumeMetrics(ctx, transformedMetrics)
}

// applyResourceFiltering applies resource filtering to metrics
func (p *processorImpl) applyResourceFiltering(md pmetric.Metrics) (pmetric.Metrics, error) {
	if !p.config.ResourceFilter.Enabled {
		return md, nil // Skip filtering if disabled
	}

	// Reset counters
	p.totalItems = 0
	p.totalIncluded = 0

	// Step 1: Apply priority tagging if using priority or hybrid strategy
	if p.config.ResourceFilter.FilterStrategy == resource_filter.StrategyPriority ||
		p.config.ResourceFilter.FilterStrategy == resource_filter.StrategyHybrid {
		p.applyPriorityTagging(md)
	}

	// Step 2: Collect information for top-k algorithm if using topk or hybrid strategy
	if p.config.ResourceFilter.FilterStrategy == resource_filter.StrategyTopK ||
		p.config.ResourceFilter.FilterStrategy == resource_filter.StrategyHybrid {
		p.collectTopKInfo(md)
		p.updateTopKSet()
	}

	// Step 3: Filter metrics based on the configured strategy
	filteredMetrics, err := p.filterMetrics(md)
	if err != nil {
		return md, err
	}

	// Step 4: Apply rollup aggregation if enabled
	if p.config.ResourceFilter.Rollup.Enabled {
		return p.applyRollup(filteredMetrics), nil
	}

	return filteredMetrics, nil
}

// applyPriorityTagging tags resources with priority levels based on configured rules
func (p *processorImpl) applyPriorityTagging(md pmetric.Metrics) {
	// Reset priority counts for this batch
	p.priorityCounts = make(map[string]int)

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		resource := rm.Resource()

		// Find resource identifier (typically process.name)
		var resourceID string
		for j := 0; j < resource.Attributes().Len(); j++ {
			k, v := resource.Attributes().At(j)
			if k == "process.executable.name" || k == "process.name" {
				resourceID = v.Str()
				break
			}
		}

		if resourceID == "" {
			continue // Skip resources without an identifier
		}

		// Apply matching rules
		for j, re := range p.priorityRules {
			if re != nil && re.MatchString(resourceID) {
				// Get priority level as string
				priority := string(p.config.ResourceFilter.PriorityRules[j].Priority)

				// Add priority attribute
				resource.Attributes().PutStr(
					p.config.ResourceFilter.PriorityAttribute,
					priority,
				)

				// Count resources by priority for metrics
				p.priorityCounts[priority]++

				break // Stop at first match
			}
		}
	}
}

// collectTopKInfo collects information for the top-k algorithm
func (p *processorImpl) collectTopKInfo(md pmetric.Metrics) {
	// Reset the total items counter for this batch
	p.totalItems = 0

	// Iterate through all resource metrics
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)

		// Get resource identifier
		var resourceID string
		if val, ok := rm.Resource().Attributes().Get(p.config.ResourceFilter.TopK.ResourceField); ok {
			resourceID = val.AsString()
		} else {
			// Skip resources without the specified field
			continue
		}

		// Find the counter field value across all metrics
		var counterValue float64
		metricsFound := false

		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)

			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)

				if metric.Name() == p.config.ResourceFilter.TopK.CounterField {
					// Extract value based on metric type
					switch metric.Type() {
					case pmetric.MetricTypeGauge:
						if metric.Gauge().DataPoints().Len() > 0 {
							dp := metric.Gauge().DataPoints().At(0)
							counterValue = dp.DoubleValue()
							metricsFound = true
						}
					case pmetric.MetricTypeSum:
						if metric.Sum().DataPoints().Len() > 0 {
							dp := metric.Sum().DataPoints().At(0)
							counterValue = dp.DoubleValue()
							metricsFound = true
						}
					}

					// Found the counter, no need to continue looking
					if metricsFound {
						break
					}
				}
			}

			if metricsFound {
				break
			}
		}

		if metricsFound {
			// Add to Space-Saving algorithm
			p.topkAlgo.Add(resourceID, counterValue)
			p.totalItems++
		}
	}
}

// updateTopKSet updates the set of top-k items
func (p *processorImpl) updateTopKSet() {
	// Clear existing set
	p.topkSet = make(map[string]struct{})

	// Get top-k items from algorithm
	topkItems := p.topkAlgo.GetTopK()

	// Add to set
	for _, item := range topkItems {
		p.topkSet[item.ID] = struct{}{}
	}
}

// filterMetrics filters metrics based on the configured strategy
func (p *processorImpl) filterMetrics(md pmetric.Metrics) (pmetric.Metrics, error) {
	p.totalIncluded = 0

	// Create a new metrics collection for filtered results
	filtered := pmetric.NewMetrics()

	// Iterate through all resource metrics
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)

		// Determine if this resource should be included based on the strategy
		includeResource := false

		switch p.config.ResourceFilter.FilterStrategy {
		case resource_filter.StrategyPriority:
			// Include based on priority
			if val, ok := rm.Resource().Attributes().Get(p.config.ResourceFilter.PriorityAttribute); ok {
				priority := resource_filter.PriorityLevel(val.Str())
				includeResource = p.isPriorityIncluded(priority)
			}

		case resource_filter.StrategyTopK:
			// Include based on top-k
			if val, ok := rm.Resource().Attributes().Get(p.config.ResourceFilter.TopK.ResourceField); ok {
				resourceID := val.Str()
				_, includeResource = p.topkSet[resourceID]
			}

		case resource_filter.StrategyHybrid:
			// Include if either priority is high enough or in top-k
			if val, ok := rm.Resource().Attributes().Get(p.config.ResourceFilter.PriorityAttribute); ok {
				priority := resource_filter.PriorityLevel(val.Str())
				if p.isPriorityIncluded(priority) {
					includeResource = true
				}
			}

			if !includeResource {
				// Check top-k as a fallback
				if val, ok := rm.Resource().Attributes().Get(p.config.ResourceFilter.TopK.ResourceField); ok {
					resourceID := val.Str()
					_, includeResource = p.topkSet[resourceID]
				}
			}
		}

		if includeResource {
			// Include this resource in the output
			newRM := filtered.ResourceMetrics().AppendEmpty()
			rm.CopyTo(newRM)

			// Add filter tag to the resource
			newRM.Resource().Attributes().PutStr("aemf.filter.included", "true")

			p.totalIncluded++
		}
	}

	return filtered, nil
}

// isPriorityIncluded determines if a priority level should be included in the output
func (p *processorImpl) isPriorityIncluded(priority resource_filter.PriorityLevel) bool {
	switch p.config.ResourceFilter.Rollup.PriorityThreshold {
	case resource_filter.PriorityLow:
		return priority == resource_filter.PriorityMedium ||
			priority == resource_filter.PriorityHigh ||
			priority == resource_filter.PriorityCritical
	case resource_filter.PriorityMedium:
		return priority == resource_filter.PriorityHigh ||
			priority == resource_filter.PriorityCritical
	case resource_filter.PriorityHigh:
		return priority == resource_filter.PriorityCritical
	default:
		return false
	}
}

// applyRollup applies rollup aggregation to filtered metrics
func (p *processorImpl) applyRollup(md pmetric.Metrics) pmetric.Metrics {
	// Create output metrics
	out := pmetric.NewMetrics()
	out.ResourceMetrics().EnsureCapacity(md.ResourceMetrics().Len())

	// Aggregation map
	type agg struct {
		sum   float64
		count int
		typ   pmetric.MetricType
	}
	metricsAgg := map[string]*agg{}

	// Reset rollup resources counter for this batch
	p.rollupResources = 0

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)

		// Check if resource should be rolled up
		shouldRollup := false
		if val, ok := rm.Resource().Attributes().Get(p.config.ResourceFilter.PriorityAttribute); ok {
			priority := resource_filter.PriorityLevel(val.Str())
			shouldRollup = priority == p.config.ResourceFilter.Rollup.PriorityThreshold ||
				p.isPriorityLowerThan(priority, p.config.ResourceFilter.Rollup.PriorityThreshold)
		}

		if shouldRollup {
			// Aggregate metrics
			p.rollupResources++
			for j := 0; j < rm.ScopeMetrics().Len(); j++ {
				sm := rm.ScopeMetrics().At(j)
				for k := 0; k < sm.Metrics().Len(); k++ {
					m := sm.Metrics().At(k)
					var value float64

					switch m.Type() {
					case pmetric.MetricTypeGauge:
						if m.Gauge().DataPoints().Len() == 0 {
							continue
						}
						value = m.Gauge().DataPoints().At(0).DoubleValue()
					case pmetric.MetricTypeSum:
						if m.Sum().DataPoints().Len() == 0 {
							continue
						}
						value = m.Sum().DataPoints().At(0).DoubleValue()
					default:
						continue
					}

					a := metricsAgg[m.Name()]
					if a == nil {
						a = &agg{typ: m.Type()}
						metricsAgg[m.Name()] = a
					}
					a.sum += value
					a.count++
				}
			}
		} else {
			// Keep as is - not rolled up
			newRM := out.ResourceMetrics().AppendEmpty()
			rm.CopyTo(newRM)
		}
	}

	// Build aggregated resource if any metrics aggregated
	if len(metricsAgg) > 0 {
		aggRM := out.ResourceMetrics().AppendEmpty()
		res := aggRM.Resource()
		res.Attributes().PutStr("process.executable.name", p.config.ResourceFilter.Rollup.NamePrefix)
		res.Attributes().PutStr(p.config.ResourceFilter.PriorityAttribute,
			string(p.config.ResourceFilter.Rollup.PriorityThreshold))
		res.Attributes().PutStr("aemf.rollup.count", fmt.Sprintf("%d", p.rollupResources))

		sm := aggRM.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName(Type)

		now := pcommon.NewTimestampFromTime(time.Now())
		for name, a := range metricsAgg {
			m := sm.Metrics().AppendEmpty()
			m.SetName(fmt.Sprintf("%s.%s", p.config.ResourceFilter.Rollup.NamePrefix, name))

			switch a.typ {
			case pmetric.MetricTypeSum:
				sum := m.SetEmptySum()
				sum.SetIsMonotonic(true)
				sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
				dp := sum.DataPoints().AppendEmpty()
				val := a.sum
				if p.config.ResourceFilter.Rollup.Strategy == resource_filter.AggregationAvg && a.count > 0 {
					val = a.sum / float64(a.count)
				}
				dp.SetDoubleValue(val)
				dp.SetTimestamp(now)
			default:
				gauge := m.SetEmptyGauge()
				dp := gauge.DataPoints().AppendEmpty()
				val := a.sum
				if p.config.ResourceFilter.Rollup.Strategy == resource_filter.AggregationAvg && a.count > 0 {
					val = a.sum / float64(a.count)
				}
				dp.SetDoubleValue(val)
				dp.SetTimestamp(now)
			}
		}
	}

	return out
}

// isPriorityLowerThan checks if priority1 is lower than priority2
func (p *processorImpl) isPriorityLowerThan(
	priority1,
	priority2 resource_filter.PriorityLevel,
) bool {
	priorityValues := map[resource_filter.PriorityLevel]int{
		resource_filter.PriorityCritical: 3,
		resource_filter.PriorityHigh:     2,
		resource_filter.PriorityMedium:   1,
		resource_filter.PriorityLow:      0,
	}

	return priorityValues[priority1] < priorityValues[priority2]
}

// applyMetricTransformations applies various transformations to metrics
func (p *processorImpl) applyMetricTransformations(md pmetric.Metrics) (pmetric.Metrics, error) {
	// Apply histograms if enabled
	if p.config.Transformation.Histograms.Enabled {
		md = p.applyHistograms(md)
	}

	// Apply attribute actions
	if len(p.config.Transformation.Attributes.Actions) > 0 {
		md = p.applyAttributeActions(md)
	}

	return md, nil
}

// applyHistograms converts individual metrics to histograms
func (p *processorImpl) applyHistograms(md pmetric.Metrics) pmetric.Metrics {
	// If no histogram buckets are configured, return unchanged
	if len(p.histogramBuckets) == 0 {
		return md
	}

	// Reset histogram count
	p.histogramCount = 0

	// Create a new metrics collection
	result := pmetric.NewMetrics()

	// Iterate through all resource metrics
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		newRM := result.ResourceMetrics().AppendEmpty()
		rm.Resource().CopyTo(newRM.Resource())

		// Process each scope metrics
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			newSM := newRM.ScopeMetrics().AppendEmpty()
			sm.Scope().CopyTo(newSM.Scope())

			// Track metrics that will be converted to histograms
			histogramMetrics := make(map[string]bool)

			// First, identify metrics that need histogram conversion
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)

				// Check if this metric has histogram buckets configured
				if boundaries, ok := p.histogramBuckets[metric.Name()]; ok && len(boundaries) > 0 {
					histogramMetrics[metric.Name()] = true
				}
			}

			// Second, process all metrics
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)

				// Check if this metric should be converted to histogram
				if _, needsHistogram := histogramMetrics[metric.Name()]; needsHistogram {
					// Count histogram conversions for metrics
					p.histogramCount++

					// Record in counter metric
					p.metricsCollector.AddCounter("phoenix.histogram.conversions", "", "").
						WithValue(1.0).
						WithAttributes(map[string]string{
							"metric_name": metric.Name(),
						})

					// Create histogram metric
					newMetric := newSM.Metrics().AppendEmpty()
					newMetric.SetName(metric.Name() + "_histogram")
					newMetric.SetDescription(metric.Description())
					newMetric.SetUnit(metric.Unit())

					histogram := newMetric.SetEmptyHistogram()
					histogram.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

					// Extract the source value
					var sourceValue float64
					switch metric.Type() {
					case pmetric.MetricTypeGauge:
						if metric.Gauge().DataPoints().Len() > 0 {
							dp := metric.Gauge().DataPoints().At(0)
							sourceValue = dp.DoubleValue()

							// Create histogram data point
							histDP := histogram.DataPoints().AppendEmpty()
							dp.Timestamp().CopyTo(histDP.Timestamp())
							dp.StartTimestamp().CopyTo(histDP.StartTimestamp())
							dp.Attributes().CopyTo(histDP.Attributes())

							// Set boundaries from configuration
							boundaries := p.histogramBuckets[metric.Name()]
							histDP.ExplicitBounds().FromRaw(boundaries)

							// Calculate bucket counts
							counts := make([]uint64, len(boundaries)+1)
							for i, boundary := range boundaries {
								if sourceValue <= boundary {
									counts[i] = 1
									break
								}
							}

							if sourceValue > boundaries[len(boundaries)-1] {
								counts[len(boundaries)] = 1
							}

							histDP.BucketCounts().FromRaw(counts)
							histDP.SetCount(1)
							histDP.SetSum(sourceValue)
						}
					case pmetric.MetricTypeSum:
						if metric.Sum().DataPoints().Len() > 0 {
							dp := metric.Sum().DataPoints().At(0)
							sourceValue = dp.DoubleValue()

							// Create histogram data point
							histDP := histogram.DataPoints().AppendEmpty()
							dp.Timestamp().CopyTo(histDP.Timestamp())
							dp.StartTimestamp().CopyTo(histDP.StartTimestamp())
							dp.Attributes().CopyTo(histDP.Attributes())

							// Set boundaries from configuration
							boundaries := p.histogramBuckets[metric.Name()]
							histDP.ExplicitBounds().FromRaw(boundaries)

							// Calculate bucket counts
							counts := make([]uint64, len(boundaries)+1)
							for i, boundary := range boundaries {
								if sourceValue <= boundary {
									counts[i] = 1
									break
								}
							}

							if sourceValue > boundaries[len(boundaries)-1] {
								counts[len(boundaries)] = 1
							}

							histDP.BucketCounts().FromRaw(counts)
							histDP.SetCount(1)
							histDP.SetSum(sourceValue)
						}
					}

					// Also keep the original metric
					newOriginalMetric := newSM.Metrics().AppendEmpty()
					metric.CopyTo(newOriginalMetric)
				} else {
					// Copy metric as is
					newMetric := newSM.Metrics().AppendEmpty()
					metric.CopyTo(newMetric)
				}
			}
		}
	}

	return result
}

// applyAttributeActions applies attribute actions to all resources
func (p *processorImpl) applyAttributeActions(md pmetric.Metrics) pmetric.Metrics {
	// Process each resource
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		resource := rm.Resource()

		// Apply actions to resource attributes
		for _, action := range p.config.Transformation.Attributes.Actions {
			switch action.Action {
			case "insert":
				if _, ok := resource.Attributes().Get(action.Key); !ok {
					setAttributeValue(resource.Attributes(), action.Key, action.Value)
				}
			case "update":
				if _, ok := resource.Attributes().Get(action.Key); ok {
					setAttributeValue(resource.Attributes(), action.Key, action.Value)
				}
			case "upsert":
				setAttributeValue(resource.Attributes(), action.Key, action.Value)
			case "delete":
				resource.Attributes().Remove(action.Key)
			}
		}
	}

	return md
}

// setAttributeValue sets an attribute value with the appropriate type
func setAttributeValue(attrs pcommon.Map, key string, value interface{}) {
	switch v := value.(type) {
	case string:
		attrs.PutStr(key, v)
	case int:
		attrs.PutInt(key, int64(v))
	case int64:
		attrs.PutInt(key, v)
	case float64:
		attrs.PutDouble(key, v)
	case bool:
		attrs.PutBool(key, v)
	}
}

// Start initializes the processor and sets up metrics collection
func (p *processorImpl) Start(ctx context.Context, host component.Host) error {
	// Call parent Start method
	if err := p.UpdateableProcessor.Start(ctx, host); err != nil {
		return err
	}

	// Initialize and register all metrics
	p.initializeMetrics()

	return nil
}

// Shutdown cleans up resources
func (p *processorImpl) Shutdown(ctx context.Context) error {
	// Emit final metrics before shutdown
	if err := p.metricsCollector.Emit(ctx); err != nil {
		p.GetLogger().Warn("Failed to emit final metrics during shutdown", zap.Error(err))
	}

	// Call parent Shutdown method
	return p.UpdateableProcessor.Shutdown(ctx)
}

// initializeMetrics initializes all metrics that will be collected
func (p *processorImpl) initializeMetrics() {
	// Register resource filtering metrics
	p.metricsCollector.AddGauge(
		"phoenix.filter.resources.total",
		"Total number of resources processed by the filter",
		"count",
	)

	p.metricsCollector.AddGauge(
		"phoenix.filter.resources.included",
		"Number of resources included after filtering",
		"count",
	)

	p.metricsCollector.AddGauge(
		"phoenix.filter.coverage_ratio",
		"Ratio of included resources to total resources",
		"ratio",
	)

	// Register priority tagging metrics
	p.metricsCollector.AddGauge(
		"phoenix.priority_tagged.resources",
		"Number of resources tagged with each priority level",
		"count",
	)

	// Register TopK metrics
	p.metricsCollector.AddGauge(
		"phoenix.topk.k_value",
		"Current value of K in the topK algorithm",
		"count",
	)

	p.metricsCollector.AddGauge(
		"phoenix.topk.included_resources",
		"Number of resources included in the top K set",
		"count",
	)

	// Register rollup metrics
	p.metricsCollector.AddGauge(
		"phoenix.rollup.aggregated_resources",
		"Number of resources aggregated in the rollup",
		"count",
	)

	// Register histogram transformation metrics
	p.metricsCollector.AddCounter(
		"phoenix.histogram.conversions",
		"Number of metrics converted to histograms",
		"count",
	)

	// Register performance metrics
	p.metricsCollector.AddGauge(
		"phoenix.processing.duration_ms",
		"Time taken to process metrics batch in milliseconds",
		"ms",
	)
}

// emitMetrics emits all collected metrics
func (p *processorImpl) emitMetrics(ctx context.Context) {
	// Update calculated metrics

	// Filter coverage ratio
	if p.totalItems > 0 {
		coverageRatio := float64(p.totalIncluded) / float64(p.totalItems)
		p.metricsCollector.AddGauge("phoenix.filter.coverage_ratio", "", "").
			WithValue(coverageRatio)
	}

	// Total resources
	p.metricsCollector.AddGauge("phoenix.filter.resources.total", "", "").
		WithValue(float64(p.totalItems))

	// Included resources
	p.metricsCollector.AddGauge("phoenix.filter.resources.included", "", "").
		WithValue(float64(p.totalIncluded))

	// TopK value
	if p.topkAlgo != nil {
		p.metricsCollector.AddGauge("phoenix.topk.k_value", "", "").
			WithValue(float64(p.topkAlgo.K()))

		p.metricsCollector.AddGauge("phoenix.topk.included_resources", "", "").
			WithValue(float64(len(p.topkSet)))
	}

	// Priority tagged resources
	for priority, count := range p.priorityCounts {
		p.metricsCollector.AddGauge("phoenix.priority_tagged.resources", "", "").
			WithValue(float64(count)).
			WithAttributes(map[string]string{
				"priority": priority,
			})
	}

	// Rollup resources
	p.metricsCollector.AddGauge("phoenix.rollup.aggregated_resources", "", "").
		WithValue(float64(p.rollupResources))

	// Emit all metrics
	if err := p.metricsCollector.Emit(ctx); err != nil {
		p.GetLogger().Warn("Failed to emit metrics", zap.Error(err))
	}

	// Reset histogram count and rollup resources for next batch
	p.histogramCount = 0
	p.rollupResources = 0
}

// OnConfigPatch implements the UpdateableProcessor interface
func (p *processorImpl) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	// Add a metric for the config patch
	p.metricsCollector.AddCounter("phoenix.config.patches", "Configuration patches applied", "count").
		WithValue(1.0).
		WithAttributes(map[string]string{
			"parameter": patch.Parameter,
		})

	return p.configManager.HandleConfigPatch(ctx, patch)
}

// GetConfigStatus implements the UpdateableProcessor interface
func (p *processorImpl) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	return p.configManager.GetConfigStatus(ctx)
}
