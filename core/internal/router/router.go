package router

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/nats-io/nats.go"
	"gopkg.in/yaml.v3"

	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// PipelineSpec describes a pipeline template loaded from YAML or JSON.
type PipelineSpec struct {
	Name                 string     `json:"name"                       yaml:"pipeline_id"`
	Description          string     `json:"description"                yaml:"description"`
	PipelineVersion      int        `json:"pipeline_version"           yaml:"pipeline_version"`
	AcceptedMessageTypes []string   `json:"accepted_message_types"     yaml:"accepted_message_types"`
	Slots                []SlotSpec `json:"slots"                      yaml:"slots"`
	FilePath             string     `json:"-"                          yaml:"-"`
}

// SlotSpec describes a single slot within a pipeline.
type SlotSpec struct {
	Name                 string `json:"name"                       yaml:"slot_id"`
	Capability           string `json:"capability"                  yaml:"capability"`
	Required             bool   `json:"required"                   yaml:"required"`
	AllowsMultiplePlugins bool   `json:"allows_multiple_plugins"    yaml:"allows_multiple_plugins"`
	ExecutionMode        string `json:"execution_mode"              yaml:"execution_mode"`
	ValidEntryPoint      bool   `json:"valid_entry_point"           yaml:"valid_entry_point"`
	TimeoutSeconds       int    `json:"timeout_seconds"             yaml:"timeout_seconds"`
	OnEmpty              string `json:"on_empty"                   yaml:"on_empty"`
	OnAllFailed          string `json:"on_all_failed"              yaml:"on_all_failed"`
}

// DispatchTable maps pipeline_id -> slot_id -> []plugin_ids.
type DispatchTable map[string]map[string][]string

// Router dispatches messages through pipeline slots to registered plugins.
type Router struct {
	mu                sync.RWMutex
	registry          *registry.Registry
	bus               *eventbus.Bus
	pipelines         map[string]*PipelineSpec
	lastDispatchTable map[string]*CompiledDispatchTable
}

// New creates a new pipeline router.
func New(reg *registry.Registry, bus *eventbus.Bus) *Router {
	return &Router{
		registry:          reg,
		bus:               bus,
		pipelines:         make(map[string]*PipelineSpec),
		lastDispatchTable: make(map[string]*CompiledDispatchTable),
	}
}

// LoadPipeline registers a pipeline template after validation.
func (r *Router) LoadPipeline(spec *PipelineSpec) error {
	if err := ValidatePipeline(spec); err != nil {
		return fmt.Errorf("pipeline %s validation failed: %w", spec.Name, err)
	}

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

// CompileDispatchTable builds a dispatch table from loaded pipelines and the registry.
func (r *Router) CompileDispatchTable() DispatchTable {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dt := make(DispatchTable)
	for pipelineName, spec := range r.pipelines {
		slotMap := make(map[string][]string)
		for _, slot := range spec.Slots {
			plugins := r.registry.FindByPipelineSlot(pipelineName, slot.Name)
			ids := make([]string, 0, len(plugins))
			for _, p := range plugins {
				ids = append(ids, p.ID)
			}
			slotMap[slot.Name] = ids
		}
		dt[pipelineName] = slotMap
	}
	return dt
}

// ParsePipelineSpec parses a JSON pipeline specification.
func ParsePipelineSpec(data []byte) (*PipelineSpec, error) {
	var spec PipelineSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse pipeline spec: %w", err)
	}
	return &spec, nil
}

// ParsePipelineSpecYAML parses a YAML pipeline specification.
func ParsePipelineSpecYAML(data []byte) (*PipelineSpec, error) {
	var spec PipelineSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse pipeline spec YAML: %w", err)
	}
	return &spec, nil
}

// LoadPipelineFromYAML reads a YAML file and parses it as a PipelineSpec.
func LoadPipelineFromYAML(path string) (*PipelineSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pipeline YAML %s: %w", path, err)
	}

	spec, err := ParsePipelineSpecYAML(data)
	if err != nil {
		return nil, err
	}

	spec.FilePath = path
	return spec, nil
}

// LoadPipelinesFromDir loads all .yaml/.yml pipeline files from a directory.
func LoadPipelinesFromDir(dir string) ([]*PipelineSpec, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read pipelines dir %s: %w", dir, err)
	}

	var specs []*PipelineSpec
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		spec, err := LoadPipelineFromYAML(path)
		if err != nil {
			return nil, fmt.Errorf("load pipeline from %s: %w", path, err)
		}
		specs = append(specs, spec)
	}

	return specs, nil
}

// ValidatePipeline checks a PipelineSpec for spec compliance.
func ValidatePipeline(spec *PipelineSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("pipeline name must not be empty")
	}

	if spec.PipelineVersion != 0 && spec.PipelineVersion != 1 {
		return fmt.Errorf("pipeline %s: unsupported pipeline_version %d (must be 1)", spec.Name, spec.PipelineVersion)
	}

	// Set default version if zero
	if spec.PipelineVersion == 0 {
		spec.PipelineVersion = 1
	}

	hasEntryPoint := false
	slotNames := make(map[string]bool)

	validExecutionModes := map[string]bool{
		"sequential_by_priority": true,
		"parallel":              true,
		"by_action_type":        true,
		"by_channel_match":      true,
		"":                      true, // empty is allowed (default)
	}
	validOnEmpty := map[string]bool{
		"skip": true, "fail": true, "buffer": true, "": true,
	}
	validOnAllFailed := map[string]bool{
		"skip": true, "fail": true, "retry": true, "": true,
	}

	for _, slot := range spec.Slots {
		if slot.Name == "" {
			return fmt.Errorf("pipeline %s: slot name must not be empty", spec.Name)
		}

		if slotNames[slot.Name] {
			return fmt.Errorf("pipeline %s: duplicate slot name %s", spec.Name, slot.Name)
		}
		slotNames[slot.Name] = true

		if slot.ValidEntryPoint {
			hasEntryPoint = true
		}

		if !validExecutionModes[slot.ExecutionMode] {
			return fmt.Errorf("pipeline %s slot %s: invalid execution_mode %q", spec.Name, slot.Name, slot.ExecutionMode)
		}

		if !validOnEmpty[slot.OnEmpty] {
			return fmt.Errorf("pipeline %s slot %s: invalid on_empty %q", spec.Name, slot.Name, slot.OnEmpty)
		}

		if !validOnAllFailed[slot.OnAllFailed] {
			return fmt.Errorf("pipeline %s slot %s: invalid on_all_failed %q", spec.Name, slot.Name, slot.OnAllFailed)
		}

		if slot.Required && slot.TimeoutSeconds <= 0 {
			return fmt.Errorf("pipeline %s slot %s: required slot must have timeout_seconds > 0", spec.Name, slot.Name)
		}
	}

	if !hasEntryPoint {
		return fmt.Errorf("pipeline %s: must have at least one slot with valid_entry_point=true", spec.Name)
	}

	return nil
}