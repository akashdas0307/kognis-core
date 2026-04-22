package router

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go"

	"github.com/kognis-framework/kognis-core/core/internal/eventbus"
	"github.com/kognis-framework/kognis-core/core/internal/registry"
)

// PipelineSpec describes a pipeline template loaded from YAML.
type PipelineSpec struct {
	Name  string     `json:"name"`
	Slots []SlotSpec `json:"slots"`
}

// SlotSpec describes a single slot within a pipeline.
type SlotSpec struct {
	Name       string `json:"name"`
	Capability string `json:"capability"`
	Required   bool   `json:"required"`
}

// Router dispatches messages through pipeline slots to registered plugins.
type Router struct {
	mu        sync.RWMutex
	registry  *registry.Registry
	bus       *eventbus.Bus
	pipelines map[string]*PipelineSpec
}

// New creates a new pipeline router.
func New(reg *registry.Registry, bus *eventbus.Bus) *Router {
	return &Router{
		registry:  reg,
		bus:       bus,
		pipelines: make(map[string]*PipelineSpec),
	}
}

// LoadPipeline registers a pipeline template.
func (r *Router) LoadPipeline(spec *PipelineSpec) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pipelines[spec.Name]; exists {
		return fmt.Errorf("pipeline %s already loaded", spec.Name)
	}

	r.pipelines[spec.Name] = spec
	log.Printf("router: loaded pipeline %s (%d slots)", spec.Name, len(spec.Slots))
	return nil
}

// GetPipeline returns a loaded pipeline by name.
func (r *Router) GetPipeline(name string) (*PipelineSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.pipelines[name]
	return p, ok
}

// ListPipelines returns all loaded pipeline names.
func (r *Router) ListPipelines() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.pipelines))
	for name := range r.pipelines {
		names = append(names, name)
	}
	return names
}

// Dispatch routes a message through a pipeline, invoking plugins for each slot.
func (r *Router) Dispatch(pipelineName string, envelope []byte) error {
	r.mu.RLock()
	spec, ok := r.pipelines[pipelineName]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("pipeline %s not found", pipelineName)
	}

	subject := fmt.Sprintf("kognis.pipeline.%s", pipelineName)

	// Broadcast the envelope on the pipeline subject
	if err := r.bus.Publish(subject, envelope); err != nil {
		return fmt.Errorf("publish to pipeline %s: %w", pipelineName, err)
	}

	// Route to slot-specific subjects for plugins registered to each slot
	for _, slot := range spec.Slots {
		plugins := r.registry.FindByPipelineSlot(pipelineName, slot.Name)
		if slot.Required && len(plugins) == 0 {
			return fmt.Errorf("required slot %s in pipeline %s has no registered plugins", slot.Name, pipelineName)
		}

		slotSubject := fmt.Sprintf("kognis.pipeline.%s.slot.%s", pipelineName, slot.Name)
		if err := r.bus.Publish(slotSubject, envelope); err != nil {
			log.Printf("router: failed to dispatch to slot %s/%s: %v", pipelineName, slot.Name, err)
			continue
		}
	}

	return nil
}

// SubscribePipeline subscribes to all messages on a pipeline.
func (r *Router) SubscribePipeline(pipelineName string, handler nats.MsgHandler) (*nats.Subscription, error) {
	subject := fmt.Sprintf("kognis.pipeline.%s", pipelineName)
	return r.bus.Subscribe(subject, handler)
}

// SubscribeSlot subscribes to messages on a specific pipeline slot.
func (r *Router) SubscribeSlot(pipelineName, slotName string, handler nats.MsgHandler) (*nats.Subscription, error) {
	subject := fmt.Sprintf("kognis.pipeline.%s.slot.%s", pipelineName, slotName)
	return r.bus.Subscribe(subject, handler)
}

// ParsePipelineSpec parses a JSON pipeline specification.
func ParsePipelineSpec(data []byte) (*PipelineSpec, error) {
	var spec PipelineSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse pipeline spec: %w", err)
	}
	return &spec, nil
}