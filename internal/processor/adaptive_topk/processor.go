// Package adaptive_topk implements a processor that dynamically selects top-k resources 
// based on self-tuning mechanisms.
package adaptive_topk

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/yourorg/sa-omf/internal/interfaces"
	"github.com/yourorg/sa-omf/pkg/metrics"
	"github.com/yourorg/sa-omf/pkg/util/topk"
)

// processorImpl is the implementation of the adaptive_topk processor.
type processorImpl struct {
	config       *Config
	logger       *zap.Logger
	next         consumer.Metrics
	topkAlgo     *topk.SpaceSaving
	lock         sync.RWMutex
	metricsEmitter *metrics.MetricsEmitter
	topkSet      map[string]struct{} // Set of current top-k items
	totalItems   int                 // Total number of items seen
	totalIncluded int                // Number of items included (top-k)
}

// Ensure processorImpl implements the required interfaces.
var _ processor.Metrics = (*processorImpl)(nil)
var _ interfaces.UpdateableProcessor = (*processorImpl)(nil)

// newProcessor creates a new adaptive_topk processor.
func newProcessor(cfg *Config, settings processor.CreateSettings, nextConsumer consumer.Metrics) (*processorImpl, error) {
	p := &processorImpl{
		config:     cfg,
		logger:     settings.Logger,
		next:       nextConsumer,
		topkAlgo:   topk.NewSpaceSaving(cfg.KValue),
		topkSet:    make(map[string]struct{}),
	}
	
	return p, nil
}

// Start implements the Component interface.
func (p *processorImpl) Start(ctx context.Context, host component.Host) error {
	// Set up metrics if provider available
	return nil
}

// Shutdown implements the Component interface.
func (p *processorImpl) Shutdown(ctx context.Context) error {
	return nil
}

// Capabilities implements the processor.Metrics interface.
func (p *processorImpl) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// ConsumeMetrics implements the consumer.Metrics interface.
func (p *processorImpl) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	
	if !p.config.Enabled {
		// Pass the data unmodified if processor is disabled
		return p.next.ConsumeMetrics(ctx, md)
	}
	
	// First pass: collect information for topk algorithm
	p.collectTopKInfo(md)
	
	// Update the set of top-k items
	p.updateTopKSet()
	
	// Second pass: filter metrics based on topk set
	if err := p.filterMetrics(md); err != nil {
		p.logger.Error("Error filtering metrics", zap.Error(err))
		// Continue with unfiltered metrics on error
	}
	
	// Update coverage metrics
	coverage := p.calculateCoverage()
	if p.metricsEmitter != nil {
		// Would record coverage metric here
	}
	
	p.logger.Debug("Adaptive topk processor metrics",
		zap.Int("total_items", p.totalItems),
		zap.Int("included_items", p.totalIncluded),
		zap.Float64("coverage", coverage),
		zap.Int("k_value", p.config.KValue),
	)
	
	// Pass the filtered metrics to the next consumer
	return p.next.ConsumeMetrics(ctx, md)
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
	
	// Replace the original metrics with the filtered ones
	filtered.CopyTo(md)
	
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
	p.lock.Lock()
	defer p.lock.Unlock()
	
	switch patch.ParameterPath {
	case "k_value":
		// Type assertion
		newK, ok := patch.NewValue.(int)
		if !ok {
			return fmt.Errorf("invalid value type for k_value: %T", patch.NewValue)
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
		
		p.logger.Info("Updated k_value", zap.Int("new_k", newK))
		return nil
		
	case "enabled":
		// Type assertion
		enabled, ok := patch.NewValue.(bool)
		if !ok {
			return fmt.Errorf("invalid value type for enabled: %T", patch.NewValue)
		}
		
		// Apply the change
		p.config.Enabled = enabled
		
		p.logger.Info("Updated enabled state", zap.Bool("enabled", enabled))
		return nil
		
	default:
		return fmt.Errorf("unknown parameter: %s", patch.ParameterPath)
	}
}

// GetConfigStatus implements UpdateableProcessor interface.
func (p *processorImpl) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	
	return interfaces.ConfigStatus{
		Parameters: map[string]interface{}{
			"k_value": p.config.KValue,
			"k_min":   p.config.KMin,
			"k_max":   p.config.KMax,
		},
		Enabled: p.config.Enabled,
	}, nil
}