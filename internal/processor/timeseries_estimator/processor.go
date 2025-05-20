package timeseries_estimator

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/base"
	"github.com/deepaucksharma/Phoenix/pkg/util/hll"
)

type processorImpl struct {
	*base.UpdateableProcessor
	config *Config

	uniqueTimeSeries map[string]struct{}
	hllEstimator     *hll.HyperLogLog
	exact            bool

	memoryGauge interface{}
}

var _ processor.Metrics = (*processorImpl)(nil)
var _ interfaces.UpdateableProcessor = (*processorImpl)(nil)

func newProcessor(cfg *Config, settings processor.Settings, next consumer.Metrics) (*processorImpl, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	up := base.NewUpdateableProcessor(settings.TelemetrySettings.Logger, next, Type, settings.ID, cfg)

	p := &processorImpl{
		UpdateableProcessor: up,
		config:              cfg,
		exact:               strings.ToLower(cfg.EstimatorType) != "hll",
	}

	if p.exact {
		p.uniqueTimeSeries = make(map[string]struct{})
	} else {
		p.hllEstimator = hll.NewDefault()
	}

	return p, nil
}

func (p *processorImpl) Start(ctx context.Context, host component.Host) error {
	if err := p.UpdateableProcessor.Start(ctx, host); err != nil {
		return err
	}
	if emitter := p.GetMetricsEmitter(); emitter != nil {
		p.memoryGauge, _ = emitter.RegisterGauge("phoenix.timeseries_estimator.memory_bytes", "Memory used by the time series estimator")
	}
	return nil
}

func (p *processorImpl) record(key string) {
	if p.exact {
		if p.uniqueTimeSeries == nil {
			p.uniqueTimeSeries = make(map[string]struct{})
		}
		p.uniqueTimeSeries[key] = struct{}{}
		if len(p.uniqueTimeSeries) >= p.config.MaxUniqueTimeSeries {
			p.hllEstimator = hll.NewDefault()
			for k := range p.uniqueTimeSeries {
				p.hllEstimator.AddString(k)
			}
			p.uniqueTimeSeries = nil
			p.exact = false
			p.config.EstimatorType = "hll"
			p.GetLogger().Warn("max unique time series reached, switching to probabilistic estimator")
		}
	} else {
		if p.hllEstimator == nil {
			p.hllEstimator = hll.NewDefault()
		}
		p.hllEstimator.AddString(key)
	}
}

func buildKey(name string, resAttrs, attrs pcommon.Map) string {
	parts := []string{name}

	if resAttrs.Len() > 0 {
		keys := make([]string, 0, resAttrs.Len())
		resAttrs.Range(func(k string, _ pcommon.Value) bool {
			keys = append(keys, k)
			return true
		})
		sort.Strings(keys)
		for _, k := range keys {
			v := resAttrs.Get(k)
			parts = append(parts, fmt.Sprintf("%s=%v", k, v.AsString()))
		}
	}

	if attrs.Len() > 0 {
		keys := make([]string, 0, attrs.Len())
		attrs.Range(func(k string, _ pcommon.Value) bool {
			keys = append(keys, k)
			return true
		})
		sort.Strings(keys)
		for _, k := range keys {
			v := attrs.Get(k)
			parts = append(parts, fmt.Sprintf("%s=%v", k, v.AsString()))
		}
	}

	return strings.Join(parts, "|")
}

func (p *processorImpl) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.Lock()
	defer p.Unlock()

	if !p.config.Enabled {
		return p.GetNext().ConsumeMetrics(ctx, md)
	}

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		res := rm.Resource().Attributes()
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				switch m.Type() {
				case pmetric.MetricTypeGauge:
					g := m.Gauge()
					for x := 0; x < g.DataPoints().Len(); x++ {
						dp := g.DataPoints().At(x)
						p.record(buildKey(m.Name(), res, dp.Attributes()))
					}
				case pmetric.MetricTypeSum:
					s := m.Sum()
					for x := 0; x < s.DataPoints().Len(); x++ {
						dp := s.DataPoints().At(x)
						p.record(buildKey(m.Name(), res, dp.Attributes()))
					}
				case pmetric.MetricTypeHistogram:
					h := m.Histogram()
					for x := 0; x < h.DataPoints().Len(); x++ {
						dp := h.DataPoints().At(x)
						p.record(buildKey(m.Name(), res, dp.Attributes()))
					}
				}
			}
		}
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	if emitter := p.GetMetricsEmitter(); emitter != nil {
		emitter.AddMetrics(map[string]any{
			"phoenix.timeseries_estimator.memory_bytes": float64(ms.Alloc),
		})
	}

	return p.GetNext().ConsumeMetrics(ctx, md)
}
