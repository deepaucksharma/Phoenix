package process_context_learner

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

const typeStr = "process_context_learner"

// ProcessorImpl implements the process_context_learner processor.
type ProcessorImpl struct {
	config *Config
	logger *zap.Logger
	next   consumer.Metrics

	lock   sync.RWMutex
	edges  map[int][]int   // child -> parents (usually length 1)
	scores map[int]float64 // pid -> importance score
}

var _ processor.Metrics = (*ProcessorImpl)(nil)
var _ interfaces.UpdateableProcessor = (*ProcessorImpl)(nil)

func newProcessor(cfg *Config, settings processor.Settings, nextConsumer consumer.Metrics) (*ProcessorImpl, error) {
	p := &ProcessorImpl{
		config: cfg,
		logger: settings.TelemetrySettings.Logger,
		next:   nextConsumer,
		edges:  make(map[int][]int),
		scores: make(map[int]float64),
	}
	return p, nil
}

func (p *ProcessorImpl) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (p *ProcessorImpl) Shutdown(ctx context.Context) error {
	return nil
}

func (p *ProcessorImpl) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// ConsumeMetrics processes incoming metrics and updates process importance scores.
func (p *ProcessorImpl) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if !p.config.Enabled {
		return p.next.ConsumeMetrics(ctx, md)
	}

	// Track relationships
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		res := rm.Resource()

		pidAttr, ok1 := res.Attributes().Get("process.pid")
		ppidAttr, ok2 := res.Attributes().Get("process.parent_pid")
		if !ok1 || !ok2 {
			continue
		}
		pid := int(pidAttr.Int())
		ppid := int(ppidAttr.Int())
		if pid == 0 {
			continue
		}
		// record edge child->parent
		p.edges[pid] = []int{ppid}
	}

	p.computeScores()

	// Annotate metrics with scores
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		res := rm.Resource()
		pidAttr, ok := res.Attributes().Get("process.pid")
		if !ok {
			continue
		}
		pid := int(pidAttr.Int())
		if score, ok := p.scores[pid]; ok {
			res.Attributes().PutDouble("aemf.process.importance", score)
		}
	}

	return p.next.ConsumeMetrics(ctx, md)
}

func (p *ProcessorImpl) computeScores() {
	nodes := make(map[int]struct{})
	for pid, parents := range p.edges {
		nodes[pid] = struct{}{}
		for _, parent := range parents {
			if parent != 0 {
				nodes[parent] = struct{}{}
			}
		}
	}
	n := len(nodes)
	if n == 0 {
		return
	}

	// Initialize scores
	scores := make(map[int]float64, n)
	for id := range nodes {
		scores[id] = 1.0 / float64(n)
	}

	damping := p.config.DampingFactor
	iterations := p.config.Iterations

	for i := 0; i < iterations; i++ {
		newScores := make(map[int]float64, n)
		for id := range nodes {
			newScores[id] = (1 - damping) / float64(n)
		}

		sinkSum := 0.0
		for id := range nodes {
			if len(p.edges[id]) == 0 {
				sinkSum += scores[id]
			}
		}
		sinkContribution := damping * sinkSum / float64(n)
		for id := range nodes {
			newScores[id] += sinkContribution
		}

		for child, parents := range p.edges {
			if len(parents) == 0 {
				continue
			}
			share := damping * scores[child] / float64(len(parents))
			for _, parent := range parents {
				newScores[parent] += share
			}
		}
		scores = newScores
	}

	p.scores = scores
}

// GetConfigStatus implements the UpdateableProcessor interface.
func (p *ProcessorImpl) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return interfaces.ConfigStatus{
		Parameters: map[string]any{
			"damping_factor": p.config.DampingFactor,
			"iterations":     p.config.Iterations,
		},
		Enabled: p.config.Enabled,
	}, nil
}

// OnConfigPatch implements the UpdateableProcessor interface.
func (p *ProcessorImpl) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	switch patch.ParameterPath {
	case "enabled":
		enabled, ok := patch.NewValue.(bool)
		if !ok {
			return nil
		}
		p.config.Enabled = enabled
	case "damping_factor":
		v, ok := patch.NewValue.(float64)
		if !ok {
			return nil
		}
		p.config.DampingFactor = v
	case "iterations":
		v, ok := patch.NewValue.(int)
		if !ok {
			return nil
		}
		p.config.Iterations = v
	default:
		return nil
	}

	return p.config.Validate()
}

// GetScores returns the current importance scores.
func (p *ProcessorImpl) GetScores() map[int]float64 {
	p.lock.RLock()
	defer p.lock.RUnlock()
	out := make(map[int]float64, len(p.scores))
	for k, v := range p.scores {
		out[k] = v
	}
	return out
}
