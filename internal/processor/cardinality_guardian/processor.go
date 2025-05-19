package cardinality_guardian

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/pkg/util/hll"
)

const typeStr = "cardinality_guardian"

// processorImpl implements cardinality guarding.
type processorImpl struct {
	config *Config
	logger *zap.Logger
	next   consumer.Metrics
	hlls   map[string]*hll.HyperLogLog
	lock   sync.RWMutex
}

var _ processor.Metrics = (*processorImpl)(nil)
var _ interfaces.UpdateableProcessor = (*processorImpl)(nil)

func newProcessor(cfg *Config, settings processor.Settings, next consumer.Metrics) (*processorImpl, error) {
	return &processorImpl{
		config: cfg,
		logger: settings.TelemetrySettings.Logger,
		next:   next,
		hlls:   make(map[string]*hll.HyperLogLog),
	}, nil
}

func (p *processorImpl) Start(context.Context, component.Host) error { return nil }

func (p *processorImpl) Shutdown(context.Context) error { return nil }

func (p *processorImpl) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p *processorImpl) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if !p.config.Enabled {
		return p.next.ConsumeMetrics(ctx, md)
	}

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		resAttrs := rm.Resource().Attributes()

		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				h := p.getHLL(metric.Name())

				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					dps := metric.Gauge().DataPoints()
					for d := 0; d < dps.Len(); d++ {
						dp := dps.At(d)
						key := keyFrom(metric.Name(), dp.Attributes(), resAttrs)
						h.AddString(key)
					}
					if int(h.Count()) > p.config.MaxUnique {
						reduceDataPoints(metric, resAttrs, p.config.MaxUnique)
					}
				case pmetric.MetricTypeSum:
					dps := metric.Sum().DataPoints()
					for d := 0; d < dps.Len(); d++ {
						dp := dps.At(d)
						key := keyFrom(metric.Name(), dp.Attributes(), resAttrs)
						h.AddString(key)
					}
					if int(h.Count()) > p.config.MaxUnique {
						reduceDataPoints(metric, resAttrs, p.config.MaxUnique)
					}
				}
			}
		}
	}

	return p.next.ConsumeMetrics(ctx, md)
}

func (p *processorImpl) getHLL(name string) *hll.HyperLogLog {
	if h, ok := p.hlls[name]; ok {
		return h
	}
	h := hll.NewDefault()
	p.hlls[name] = h
	return h
}

func keyFrom(metricName string, attrs pcommon.Map, resAttrs pcommon.Map) string {
	parts := []string{metricName}
	attrs.Range(func(k string, v pcommon.Value) bool {
		parts = append(parts, k+"="+v.AsString())
		return true
	})
	resAttrs.Range(func(k string, v pcommon.Value) bool {
		parts = append(parts, "r."+k+"="+v.AsString())
		return true
	})
	sort.Strings(parts[1:])
	return strings.Join(parts, "|")
}

func reduceDataPoints(metric pmetric.Metric, resAttrs pcommon.Map, max int) {
	hash := fnv.New32a()
	bucketFor := func(attrs pcommon.Map) uint32 {
		key := keyFrom(metric.Name(), attrs, resAttrs)
		hash.Reset()
		hash.Write([]byte(key))
		return hash.Sum32() % uint32(max)
	}

	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			bucket := bucketFor(dp.Attributes())
			dp.Attributes().Clear()
			dp.Attributes().PutInt("cg_bucket", int64(bucket))
		}
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			bucket := bucketFor(dp.Attributes())
			dp.Attributes().Clear()
			dp.Attributes().PutInt("cg_bucket", int64(bucket))
		}
	}
}

func (p *processorImpl) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	switch patch.ParameterPath {
	case "max_unique":
		v, ok := patch.NewValue.(int)
		if !ok || v <= 0 {
			return fmt.Errorf("invalid max_unique value")
		}
		p.config.MaxUnique = v
		return nil
	case "enabled":
		b, ok := patch.NewValue.(bool)
		if !ok {
			return fmt.Errorf("invalid enabled value")
		}
		p.config.Enabled = b
		return nil
	default:
		return fmt.Errorf("unknown parameter: %s", patch.ParameterPath)
	}
}

func (p *processorImpl) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	params := map[string]any{
		"max_unique": p.config.MaxUnique,
	}
	return interfaces.ConfigStatus{Parameters: params, Enabled: p.config.Enabled}, nil
}
