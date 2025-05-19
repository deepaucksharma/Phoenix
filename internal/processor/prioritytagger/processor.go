// Package prioritytagger implements a processor that tags resources with priority levels.
package prioritytagger

import (
	"context"
	"regexp"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/yourorg/sa-omf/internal/interfaces"
	"github.com/yourorg/sa-omf/pkg/metrics"
)

const (
	typeStr = "priority_tagger"
)

// Config defines the configuration for the priority_tagger processor.
type Config struct {
	Rules   []Rule `mapstructure:"rules"`
	Enabled bool   `mapstructure:"enabled"`
}

// Rule defines a matching rule for assigning priority.
type Rule struct {
	Match    string `mapstructure:"match"`    // Regex pattern to match against process.name
	Priority string `mapstructure:"priority"` // Priority value: critical, high, medium, low
}

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid.
func (cfg *Config) Validate() error {
	// Validate each rule's regular expression
	for _, rule := range cfg.Rules {
		if _, err := regexp.Compile(rule.Match); err != nil {
			return err
		}
	}
	return nil
}

// processorImp is the implementation of the priority_tagger processor.
type processorImp struct {
	config    Config
	logger    *zap.Logger
	next      consumer.Metrics
	rules     []*regexp.Regexp
	lock      sync.RWMutex
	metrics   *metrics.MetricsEmitter
}

// Ensure the processor implements the required interfaces.
var _ processor.Metrics = (*processorImp)(nil)
var _ interfaces.UpdateableProcessor = (*processorImp)(nil)

// newProcessor creates a new priority_tagger processor.
func newProcessor(cfg *Config, settings processor.CreateSettings, nextConsumer consumer.Metrics) (*processorImp, error) {
	p := &processorImp{
		config: *cfg,
		logger: settings.Logger,
		next:   nextConsumer,
		rules:  make([]*regexp.Regexp, len(cfg.Rules)),
	}

	// Compile regular expressions
	for i, rule := range cfg.Rules {
		re, err := regexp.Compile(rule.Match)
		if err != nil {
			return nil, err
		}
		p.rules[i] = re
	}

	return p, nil
}

// Start implements the Component interface.
func (p *processorImp) Start(ctx context.Context, host component.Host) error {
	// Set up metrics if available
	metricProvider := host.GetExtensions()[component.MustNewID("prometheus")]
	if metricProvider != nil {
		// This would need a concrete implementation of metric.MeterProvider
		// p.metrics = metrics.NewMetricsEmitter(metricProvider.(metric.MeterProvider).Meter("priority_tagger"), 
		//                                      "priority_tagger", component.MustNewID(typeStr))
	}
	return nil
}

// Shutdown implements the Component interface.
func (p *processorImp) Shutdown(ctx context.Context) error {
	return nil
}

// Capabilities implements the processor.Metrics interface.
func (p *processorImp) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// ConsumeMetrics implements the consumer.Metrics interface.
func (p *processorImp) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if !p.config.Enabled {
		// If disabled, pass through without modification
		return p.next.ConsumeMetrics(ctx, md)
	}

	// Iterate through all resource metrics
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		resource := rm.Resource()

		// Find process.name attribute
		var processName string
		if val, ok := resource.Attributes().Get("process.name"); ok {
			processName = val.AsString()
		} else {
			// Skip resources without process.name
			continue
		}

		// Apply matching rules
		for j, re := range p.rules {
			if re.MatchString(processName) {
				// Add priority attribute
				resource.Attributes().PutStr("aemf.process.priority", p.config.Rules[j].Priority)
				break // Stop at first match
			}
		}
	}

	// Pass the modified metrics to the next consumer
	return p.next.ConsumeMetrics(ctx, md)
}

// OnConfigPatch implements the UpdateableProcessor interface.
func (p *processorImp) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	switch patch.ParameterPath {
	case "enabled":
		// Update enabled flag
		enabled, ok := patch.NewValue.(bool)
		if !ok {
			return nil
		}
		p.config.Enabled = enabled
		return nil

	case "rules":
		// Update rules
		rules, ok := patch.NewValue.([]Rule)
		if !ok {
			return nil
		}

		// Reset rules
		p.config.Rules = rules
		p.rules = make([]*regexp.Regexp, len(rules))

		// Compile new regular expressions
		for i, rule := range rules {
			re, err := regexp.Compile(rule.Match)
			if err != nil {
				return err
			}
			p.rules[i] = re
		}
		return nil
	}

	return nil
}

// GetConfigStatus implements the UpdateableProcessor interface.
func (p *processorImp) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return interfaces.ConfigStatus{
		Parameters: map[string]any{
			"rules": p.config.Rules,
		},
		Enabled: p.config.Enabled,
	}, nil
}
