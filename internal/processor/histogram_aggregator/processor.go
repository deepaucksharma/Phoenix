package histogram_aggregator

import (
	"context"
	"fmt"
	"sort"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

// processorImpl implements a histogram aggregation processor to optimize histograms
// for OTLP export.
type processorImpl struct {
	*base.BaseProcessor
	config *Config
}

var _ processor.Metrics = (*processorImpl)(nil)
var _ interfaces.UpdateableProcessor = (*processorImpl)(nil)

// newProcessor creates a new histogram aggregation processor.
func newProcessor(cfg *Config, settings processor.Settings, next consumer.Metrics) (*processorImpl, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &processorImpl{
		BaseProcessor: base.NewBaseProcessor(settings.TelemetrySettings.Logger, next, typeStr, settings.ID),
		config:        cfg,
	}, nil
}

// Start implements the component.Component interface.
func (p *processorImpl) Start(ctx context.Context, host component.Host) error {
	return p.BaseProcessor.Start(ctx, host)
}

// Shutdown implements the component.Component interface.
func (p *processorImpl) Shutdown(ctx context.Context) error {
	return p.BaseProcessor.Shutdown(ctx)
}

// Capabilities implements the processor.Metrics interface.
func (p *processorImpl) Capabilities() consumer.Capabilities {
	return p.BaseProcessor.Capabilities()
}

// ConsumeMetrics optimizes histograms for OTLP export.
func (p *processorImpl) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.Lock()
	defer p.Unlock()

	if !p.config.Enabled {
		return p.GetNext().ConsumeMetrics(ctx, md)
	}

	// Process metrics
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		
		// Check if this is a target process
		processName := rm.Resource().Attributes().AsRaw()["process.executable.name"]
		isTargetProcess := len(p.config.TargetProcessors) == 0 // if empty, apply to all
		
		if !isTargetProcess {
			for _, targetName := range p.config.TargetProcessors {
				if processName == targetName {
					isTargetProcess = true
					break
				}
			}
		}
		
		if !isTargetProcess {
			continue // Skip this resource if not a target process
		}
		
		// Process all scope metrics
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				
				// Only process histograms
				if metric.Type() == pmetric.MetricTypeHistogram {
					p.optimizeHistogram(metric)
				}
			}
		}
	}

	return p.GetNext().ConsumeMetrics(ctx, md)
}

// optimizeHistogram reduces the number of buckets in a histogram to improve efficiency.
func (p *processorImpl) optimizeHistogram(metric pmetric.Metric) {
	histogram := metric.Histogram()
	
	// Check if we have custom boundaries for this metric
	customBoundaries, hasCustom := p.config.CustomBoundaries[metric.Name()]
	
	// Loop through all data points
	for i := 0; i < histogram.DataPoints().Len(); i++ {
		dp := histogram.DataPoints().At(i)
		
		// If we have custom boundaries for this metric, rebucket to those
		if hasCustom && len(customBoundaries) > 0 {
			p.rebucketHistogram(dp, customBoundaries)
			continue
		}
		
		// Otherwise, just reduce bucket count if needed
		bucketCount := dp.BucketCounts().Len()
		if bucketCount > p.config.MaxBuckets {
			p.reduceBucketCount(dp, p.config.MaxBuckets)
		}
	}
}

// rebucketHistogram rearranges histogram data to match custom bucket boundaries.
func (p *processorImpl) rebucketHistogram(dp pmetric.HistogramDataPoint, boundaries []float64) {
	// Get original bucket boundaries and counts
	origBounds := getBoundaries(dp)
	origCounts := dp.BucketCounts().AsRaw()
	
	if len(origBounds) == 0 || len(origCounts) == 0 {
		return // Nothing to do
	}
	
	// Create new buckets based on custom boundaries
	newCounts := make([]uint64, len(boundaries)+1)
	
	// Keep running count of the points seen so far
	var cumulative uint64 = 0
	var origIndex int = 0
	
	// For each new boundary, find where it falls in the original buckets
	for newIndex, newBound := range boundaries {
		// Find the original bucket this new boundary falls into
		for origIndex < len(origBounds) && origBounds[origIndex] <= newBound {
			cumulative += origCounts[origIndex]
			origIndex++
		}
		
		newCounts[newIndex] = cumulative
	}
	
	// Final bucket gets everything remaining
	newCounts[len(boundaries)] = dp.Count()
	
	// Update the histogram with new boundaries and counts
	dp.BucketCounts().FromRaw(newCounts)
	dp.ExplicitBounds().FromRaw(boundaries)
}

// reduceBucketCount compacts a histogram to have fewer buckets.
func (p *processorImpl) reduceBucketCount(dp pmetric.HistogramDataPoint, maxBuckets int) {
	origBounds := getBoundaries(dp)
	origCounts := dp.BucketCounts().AsRaw()
	
	// If we already have fewer buckets than the max, no work needed
	if len(origBounds) <= maxBuckets {
		return
	}
	
	// Calculate a step size to reduce buckets
	step := len(origBounds) / maxBuckets
	if step < 1 {
		step = 1
	}
	
	// Create new boundaries and counts
	newBounds := make([]float64, 0, maxBuckets)
	newCounts := make([]uint64, 0, maxBuckets+1)
	
	// Add the initial count
	newCounts = append(newCounts, origCounts[0])
	
	// Group buckets by step size
	for i := 0; i < len(origBounds); i += step {
		// Add this boundary
		newBounds = append(newBounds, origBounds[i])
		
		// Calculate count for this new bucket
		var count uint64
		if i+step >= len(origCounts) {
			// Last bucket gets everything remaining
			count = dp.Count()
		} else {
			count = origCounts[i+step]
		}
		
		newCounts = append(newCounts, count)
		
		// If we've reached our max bucket count, break
		if len(newBounds) >= maxBuckets {
			break
		}
	}
	
	// Update the histogram with the new boundaries and counts
	dp.BucketCounts().FromRaw(newCounts)
	dp.ExplicitBounds().FromRaw(newBounds)
}

// getBoundaries extracts explicit boundaries from a histogram data point.
func getBoundaries(dp pmetric.HistogramDataPoint) []float64 {
	if dp.ExplicitBounds().Len() == 0 {
		return []float64{}
	}
	return dp.ExplicitBounds().AsRaw()
}

// OnConfigPatch implements the UpdateableProcessor interface.
func (p *processorImpl) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.Lock()
	defer p.Unlock()

	switch patch.ParameterPath {
	case "enabled":
		if v, ok := patch.NewValue.(bool); ok {
			p.config.Enabled = v
		} else {
			return fmt.Errorf("invalid type for enabled")
		}
	case "max_buckets":
		if v, ok := patch.NewValue.(int); ok {
			p.config.MaxBuckets = v
		} else if v, ok := patch.NewValue.(float64); ok {
			p.config.MaxBuckets = int(v)
		} else {
			return fmt.Errorf("invalid type for max_buckets")
		}
	default:
		return fmt.Errorf("unknown parameter %s", patch.ParameterPath)
	}

	return nil
}

// GetName returns the processor name.
func (p *processorImpl) GetName() string {
	return "histogram_aggregator"
}

// GetConfigStatus implements the UpdateableProcessor interface.
func (p *processorImpl) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.RLock()
	defer p.RUnlock()

	return interfaces.ConfigStatus{
		Parameters: map[string]any{
			"max_buckets": p.config.MaxBuckets,
		},
		Enabled: p.config.Enabled,
	}, nil
}