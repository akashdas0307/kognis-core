package router

import (
	"encoding/json"
	"testing"

	"github.com/kognis-framework/kognis-core/core/internal/registry"
)

func TestNew(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)
	if r == nil {
		t.Fatal("New() returned nil")
	}
	if len(r.pipelines) != 0 {
		t.Fatalf("expected no pipelines, got %d", len(r.pipelines))
	}
}

func TestLoadPipeline(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	spec := &PipelineSpec{
		Name: "PERCEPTION_TO_COGNITION",
		Slots: []SlotSpec{
			{Name: "perceive", Capability: "PERCEPTION", Required: true},
			{Name: "preprocess", Capability: "PERCEPTION", Required: false},
		},
	}

	if err := r.LoadPipeline(spec); err != nil {
		t.Fatalf("LoadPipeline() failed: %v", err)
	}

	got, ok := r.GetPipeline("PERCEPTION_TO_COGNITION")
	if !ok {
		t.Fatal("GetPipeline() returned false after LoadPipeline()")
	}
	if got.Name != "PERCEPTION_TO_COGNITION" {
		t.Fatalf("expected pipeline name PERCEPTION_TO_COGNITION, got %s", got.Name)
	}
	if len(got.Slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(got.Slots))
	}
}

func TestLoadPipelineDuplicate(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	spec := &PipelineSpec{Name: "dup-pipeline", Slots: []SlotSpec{}}
	r.LoadPipeline(spec)

	if err := r.LoadPipeline(spec); err == nil {
		t.Fatal("expected error for duplicate pipeline, got nil")
	}
}

func TestGetPipelineNotFound(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	_, ok := r.GetPipeline("nonexistent")
	if ok {
		t.Fatal("expected false for nonexistent pipeline, got true")
	}
}

func TestListPipelines(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	r.LoadPipeline(&PipelineSpec{Name: "pipeline-a", Slots: []SlotSpec{}})
	r.LoadPipeline(&PipelineSpec{Name: "pipeline-b", Slots: []SlotSpec{}})

	names := r.ListPipelines()
	if len(names) != 2 {
		t.Fatalf("expected 2 pipelines, got %d", len(names))
	}
}

func TestParsePipelineSpec(t *testing.T) {
	raw := map[string]interface{}{
		"name": "test-pipeline",
		"slots": []interface{}{
			map[string]interface{}{
				"name":       "think",
				"capability":  "COGNITION",
				"required":    true,
			},
		},
	}
	data, _ := json.Marshal(raw)

	spec, err := ParsePipelineSpec(data)
	if err != nil {
		t.Fatalf("ParsePipelineSpec() failed: %v", err)
	}
	if spec.Name != "test-pipeline" {
		t.Fatalf("expected name test-pipeline, got %s", spec.Name)
	}
	if len(spec.Slots) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(spec.Slots))
	}
	if spec.Slots[0].Name != "think" {
		t.Fatalf("expected slot name think, got %s", spec.Slots[0].Name)
	}
	if !spec.Slots[0].Required {
		t.Fatal("expected slot required=true")
	}
}