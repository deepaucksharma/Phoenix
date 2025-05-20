// Package adaptive_topk implements a processor that dynamically selects top-k resources 
// based on self-tuning mechanisms.
package adaptive_topk

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/base"
	"github.com/deepaucksharma/Phoenix/pkg/util/topk"
	"github.com/deepaucksharma/Phoenix/pkg/util/typeconv"
)

// processorImpl is the implementation of the adaptive_topk processor.
type processorImpl struct {
	*base.BaseProcessor
	config       *Config
	topkAlgo     *topk.SpaceSaving
	topkSet      map[string]struct{} // Set of current top-k items
	totalItems   int                 // Total number of items seen
	totalIncluded int                // Number of items included (top-k)
}

// Ensure processorImpl implements the required interfaces.
var _ processor.Metrics = (*processorImpl)(nil)
var _ interfaces.UpdateableProcessor = (*processorImpl)(nil)

// newProcessor creates a new adaptive_topk processor.
func newProcessor(cfg *Config, settings component.TelemetrySettings, nextConsumer consumer.Metrics, id component.ID) (*processorImpl, error) {
	p := &processorImpl{
		BaseProcessor: base.NewBaseProcessor(settings.Logger, nextConsumer, "adaptive_topk", id),
		config:        cfg,
		topkAlgo:      topk.NewSpaceSaving(cfg.KValue),
		topkSet:       make(map[string]struct{}),
	}
	
	return p, nil
}

// NewProcessor creates a new adaptive_topk processor - exported for testing
func NewProcessor(cfg *Config, settings component.TelemetrySettings, nextConsumer consumer.Metrics, id component.ID) (*processorImpl, error) {
	return newProcessor(cfg, settings, nextConsumer, id)
}

// Start implements the Component interface.
func (p *processorImpl) Start(ctx context.Context, host component.Host) error {
	// Set up metrics if provider available
	return p.BaseProcessor.Start(ctx, host)
}

// Shutdown implements the Component interface.
func (p *processorImpl) Shutdown(ctx context.Context) error {
	return p.BaseProcessor.Shutdown(ctx)
}

// Capabilities implements the processor.Metrics interface.
func (p *processorImpl) Capabilities() consumer.Capabilities {
	return p.BaseProcessor.Capabilities()
}

// ConsumeMetrics implements the consumer.Metrics interface.
func (p *processorImpl) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.Lock()
	defer p.Unlock()
	
	// Check if processor is disabled - if so, pass through metrics and return
	if !p.config.Enabled {
		return p.GetNext().ConsumeMetrics(ctx, md)
	}
	
	// First pass: collect information for topk algorithm
	p.collectTopKInfo(md)
	
	// Update the set of top-k items
	p.updateTopKSet()
	
	// Second pass: filter metrics based on topk set
	if err := p.filterMetrics(md); err != nil {
		p.GetLogger().Error("Error filtering metrics", zap.Error(err))
		// Continue with unfiltered metrics on error
	}
	
	// Update coverage metrics
	coverage := p.calculateCoverage()
	metricsEmitter := p.GetMetricsEmitter()
	if metricsEmitter != nil {
		// Would record coverage metric here
	}
	
	p.GetLogger().Debug("Adaptive topk processor metrics",
		zap.Int("total_items", p.totalItems),
		zap.Int("included_items", p.totalIncluded),
		zap.Float64("coverage", coverage),
		zap.Int("k_value", p.config.KValue),
	)
	
	// Pass the filtered metrics to the next consumer
	return p.GetNext().ConsumeMetrics(ctx, md)
}

// collectTopKInfo iterates through metrics to update the topk algorithm.
func (p *processorImpl) collectTopKInfo(md pmetric.Metrics) {
	// Reset the total items counter for this batch
	p.totalItems = 0
	
	// Iterate through all resource metrics
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		
		// Get resource identifier
		var resourceID string
		if val, ok := rm.Resource().Attributes().Get(p.config.ResourceField); ok {
			resourceID = val.AsString()
		} else {
			// Skip resources without the specified field
			continue
		}
		
		// Find the counter field value across all metrics
		var counterValue float64
		
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				
				if metric.Name() == p.config.CounterField {
					// Extract value based on metric type
					switch metric.Type() {
					case pmetric.MetricTypeGauge:
						if metric.Gauge().DataPoints().Len() > 0 {
							dp := metric.Gauge().DataPoints().At(0)
							counterValue = dp.DoubleValue()
						}
					case pmetric.MetricTypeSum:
						if metric.Sum().DataPoints().Len() > 0 {
							dp := metric.Sum().DataPoints().At(0)
							counterValue = dp.DoubleValue()
						}
					}
					
					// Found the counter, no need to continue looking
					break
				}
			}
		}
		
		// Add to Space-Saving algorithm
		p.topkAlgo.Add(resourceID, counterValue)
		p.totalItems++
	}
}

// updateTopKSet updates the set of top-k items.
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

// filterMetrics filters the metrics based on the top-k set.
func (p *processorImpl) filterMetrics(md pmetric.Metrics) error {
	p.totalIncluded = 0
	
	// Get original metrics count for logging
	originalCount := md.ResourceMetrics().Len()
	
	// Create a new metrics collection for filtered results
	filtered := pmetric.NewMetrics()
	
	// Iterate through all resource metrics
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		
		// Get resource identifier
		var resourceID string
		if val, ok := rm.Resource().Attributes().Get(p.config.ResourceField); ok {
			resourceID = val.AsString()
		} else {
			// Skip resources without the specified field
			p.GetLogger().Debug("Skipping resource without required field", 
				zap.String("required_field", p.config.ResourceField))
			continue
		}
		
		// Check if this resource is in the top-k set
		_, isTopK := p.topkSet[resourceID]
		
		if isTopK {
			// Include this resource in the output
			newRM := filtered.ResourceMetrics().AppendEmpty()
			rm.CopyTo(newRM)
			
			// Add topk tag to the resource
			newRM.Resource().Attributes().PutStr("aemf.topk.included", "true")
			
			p.totalIncluded++
		}
	}
	
	// Log filtering stats
	p.GetLogger().Debug("Filtered resources",
		zap.Int("original_count", originalCount),
		zap.Int("filtered_count", p.totalIncluded),
		zap.Int("k_value", p.config.KValue))
	
	// Clear the original metrics before copying filtered results
	md.ResourceMetrics().RemoveIf(func(_ pmetric.ResourceMetrics) bool { return true })
	
	// Copy filtered metrics to the original
	filtered.ResourceMetrics().CopyTo(md.ResourceMetrics())
	
	return nil
}

// calculateCoverage returns the fraction of total items covered by the top-k set.
func (p *processorImpl) calculateCoverage() float64 {
	if p.totalItems == 0 {
		return 1.0 // By convention, if no items, coverage is 100%
	}
	
	return float64(p.totalIncluded) / float64(p.totalItems)
}

// OnConfigPatch implements UpdateableProcessor interface.
func (p *processorImpl) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.Lock()
	defer p.Unlock()
	
	switch patch.ParameterPath {
	case "k_value":
		// Convert to int using type converter
		newK, err := typeconv.ToInt(patch.NewValue)
		if err != nil {
			return fmt.Errorf("invalid value for k_value: %v", err)
		}
		
		// Validate range
		if newK < p.config.KMin || newK > p.config.KMax {
			return fmt.Errorf("k_value out of allowed range [%d, %d]: %d", 
				p.config.KMin, p.config.KMax, newK)
		}
		
		// Apply the change
		p.config.KValue = newK
		
		// Update the Space-Saving algorithm
		p.topkAlgo.SetK(newK)
		
		// Update the top-k set
		p.updateTopKSet()
		
		p.GetLogger().Info("Updated k_value", zap.Int("new_k", newK))
		return nil
		
	case "enabled":
		// Convert to bool using type converter
		enabled, err := typeconv.ToBool(patch.NewValue)
		if err != nil {
			return fmt.Errorf("invalid value for enabled: %v", err)
		}
		
		// Apply the change
		p.config.Enabled = enabled
		
		p.GetLogger().Info("Updated enabled state", zap.Bool("enabled", enabled))
		return nil
		
	default:
		return fmt.Errorf("unknown parameter: %s", patch.ParameterPath)
	}
}

// GetConfigStatus implements UpdateableProcessor interface.
func (p *processorImpl) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.RLock()
	defer p.RUnlock()
	
	return interfaces.ConfigStatus{
		Parameters: map[string]interface{}{
			"k_value": p.config.KValue,
			"k_min":   p.config.KMin,
			"k_max":   p.config.KMax,
		},
		Enabled: p.config.Enabled,
	}, nil
}