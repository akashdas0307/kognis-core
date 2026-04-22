package router

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/akashdas0307/kognis-core/core/internal/registry"
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
		Name:            "PERCEPTION_TO_COGNITION",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "perceive", Capability: "PERCEPTION", Required: true, ValidEntryPoint: true, TimeoutSeconds: 5},
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

	spec := &PipelineSpec{
		Name:            "dup-pipeline",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", ValidEntryPoint: true, TimeoutSeconds: 5},
		},
	}
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

	r.LoadPipeline(&PipelineSpec{
		Name:            "pipeline-a",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", ValidEntryPoint: true, TimeoutSeconds: 5},
		},
	})
	r.LoadPipeline(&PipelineSpec{
		Name:            "pipeline-b",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", ValidEntryPoint: true, TimeoutSeconds: 5},
		},
	})

	names := r.ListPipelines()
	if len(names) != 2 {
		t.Fatalf("expected 2 pipelines, got %d", len(names))
	}
}

func TestParsePipelineSpec(t *testing.T) {
	raw := map[string]interface{}{
		"name":             "test-pipeline",
		"pipeline_version": 1,
		"slots": []interface{}{
			map[string]interface{}{
				"name":              "think",
				"capability":        "COGNITION",
				"required":          true,
				"valid_entry_point": true,
				"timeout_seconds":   30,
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
	if !spec.Slots[0].ValidEntryPoint {
		t.Fatal("expected slot valid_entry_point=true")
	}
	if spec.Slots[0].TimeoutSeconds != 30 {
		t.Fatalf("expected timeout_seconds=30, got %d", spec.Slots[0].TimeoutSeconds)
	}
}

// --- New tests for SPEC 03 full schema compliance ---

func TestParsePipelineSpecYAML(t *testing.T) {
	yamlData := []byte(`
pipeline_version: 1
pipeline_id: user_text_interaction
description: "User text message flows through perception, cognition, and response"
accepted_message_types:
  - user_text_input
slots:
  - slot_id: input_reception
    required: true
    allows_multiple_plugins: true
    execution_mode: parallel
    valid_entry_point: true
    timeout_seconds: 5
  - slot_id: cognitive_processing
    required: true
    allows_multiple_plugins: false
    execution_mode: sequential_by_priority
    valid_entry_point: false
    timeout_seconds: 60
    on_all_failed: fail
  - slot_id: output_delivery
    required: true
    allows_multiple_plugins: true
    execution_mode: by_channel_match
    valid_entry_point: false
    timeout_seconds: 10
`)

	spec, err := ParsePipelineSpecYAML(yamlData)
	if err != nil {
		t.Fatalf("ParsePipelineSpecYAML() failed: %v", err)
	}
	if spec.Name != "user_text_interaction" {
		t.Fatalf("expected pipeline_id user_text_interaction, got %s", spec.Name)
	}
	if spec.PipelineVersion != 1 {
		t.Fatalf("expected pipeline_version 1, got %d", spec.PipelineVersion)
	}
	if spec.Description != "User text message flows through perception, cognition, and response" {
		t.Fatalf("unexpected description: %s", spec.Description)
	}
	if len(spec.AcceptedMessageTypes) != 1 || spec.AcceptedMessageTypes[0] != "user_text_input" {
		t.Fatalf("expected accepted_message_types [user_text_input], got %v", spec.AcceptedMessageTypes)
	}
	if len(spec.Slots) != 3 {
		t.Fatalf("expected 3 slots, got %d", len(spec.Slots))
	}

	// Verify first slot
	s0 := spec.Slots[0]
	if s0.Name != "input_reception" {
		t.Fatalf("expected slot_id input_reception, got %s", s0.Name)
	}
	if !s0.Required {
		t.Fatal("expected required=true for input_reception")
	}
	if !s0.AllowsMultiplePlugins {
		t.Fatal("expected allows_multiple_plugins=true for input_reception")
	}
	if s0.ExecutionMode != "parallel" {
		t.Fatalf("expected execution_mode parallel, got %s", s0.ExecutionMode)
	}
	if !s0.ValidEntryPoint {
		t.Fatal("expected valid_entry_point=true for input_reception")
	}
	if s0.TimeoutSeconds != 5 {
		t.Fatalf("expected timeout_seconds=5, got %d", s0.TimeoutSeconds)
	}

	// Verify second slot
	s1 := spec.Slots[1]
	if s1.OnAllFailed != "fail" {
		t.Fatalf("expected on_all_failed=fail, got %s", s1.OnAllFailed)
	}
	if s1.AllowsMultiplePlugins {
		t.Fatal("expected allows_multiple_plugins=false for cognitive_processing")
	}

	// Verify third slot
	s2 := spec.Slots[2]
	if s2.ExecutionMode != "by_channel_match" {
		t.Fatalf("expected execution_mode by_channel_match, got %s", s2.ExecutionMode)
	}
}

func TestLoadPipelineFromYAML(t *testing.T) {
	yamlContent := []byte(`
pipeline_version: 1
pipeline_id: test-yaml-pipeline
description: "A test pipeline from YAML file"
accepted_message_types:
  - test_input
slots:
  - slot_id: entry_slot
    required: true
    valid_entry_point: true
    timeout_seconds: 10
    execution_mode: parallel
  - slot_id: process_slot
    required: false
    valid_entry_point: false
    timeout_seconds: 30
    on_empty: skip
`)

	// Create temp YAML file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-pipeline.yaml")
	if err := os.WriteFile(tmpFile, yamlContent, 0644); err != nil {
		t.Fatalf("failed to write temp YAML: %v", err)
	}

	spec, err := LoadPipelineFromYAML(tmpFile)
	if err != nil {
		t.Fatalf("LoadPipelineFromYAML() failed: %v", err)
	}
	if spec.Name != "test-yaml-pipeline" {
		t.Fatalf("expected pipeline_id test-yaml-pipeline, got %s", spec.Name)
	}
	if spec.FilePath != tmpFile {
		t.Fatalf("expected FilePath %s, got %s", tmpFile, spec.FilePath)
	}
	if len(spec.Slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(spec.Slots))
	}
	if spec.Slots[0].Name != "entry_slot" {
		t.Fatalf("expected first slot entry_slot, got %s", spec.Slots[0].Name)
	}
	if spec.Slots[1].OnEmpty != "skip" {
		t.Fatalf("expected on_empty=skip, got %s", spec.Slots[1].OnEmpty)
	}
}

func TestLoadPipelinesFromDir(t *testing.T) {
	pipeline1 := []byte(`
pipeline_version: 1
pipeline_id: pipeline-alpha
description: "Alpha pipeline"
slots:
  - slot_id: entry
    required: true
    valid_entry_point: true
    timeout_seconds: 5
`)

	pipeline2 := []byte(`
pipeline_version: 1
pipeline_id: pipeline-beta
description: "Beta pipeline"
slots:
  - slot_id: start
    required: true
    valid_entry_point: true
    timeout_seconds: 10
`)

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "alpha.yaml"), pipeline1, 0644)
	os.WriteFile(filepath.Join(tmpDir, "beta.yml"), pipeline2, 0644)
	os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("not yaml"), 0644) // should be ignored

	specs, err := LoadPipelinesFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadPipelinesFromDir() failed: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("expected 2 pipeline specs, got %d", len(specs))
	}

	names := map[string]bool{}
	for _, s := range specs {
		names[s.Name] = true
	}
	if !names["pipeline-alpha"] || !names["pipeline-beta"] {
		t.Fatalf("expected pipeline-alpha and pipeline-beta, got names: %v", names)
	}
}

// --- Validation tests ---

func TestValidatePipelineAcceptsValid(t *testing.T) {
	spec := &PipelineSpec{
		Name:            "valid-pipeline",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{
				Name:           "entry",
				Required:       true,
				ValidEntryPoint: true,
				TimeoutSeconds: 10,
				ExecutionMode:  "parallel",
			},
			{
				Name:           "process",
				Required:       false,
				ValidEntryPoint: false,
				ExecutionMode:  "sequential_by_priority",
				OnEmpty:        "skip",
			},
		},
	}

	if err := ValidatePipeline(spec); err != nil {
		t.Fatalf("ValidatePipeline() rejected valid pipeline: %v", err)
	}
}

func TestValidatePipelineRejectsMissingName(t *testing.T) {
	spec := &PipelineSpec{
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", ValidEntryPoint: true, TimeoutSeconds: 5},
		},
	}

	if err := ValidatePipeline(spec); err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestValidatePipelineRejectsNoEntryPoint(t *testing.T) {
	spec := &PipelineSpec{
		Name:            "no-entry",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "process", Required: false, ValidEntryPoint: false},
		},
	}

	if err := ValidatePipeline(spec); err == nil {
		t.Fatal("expected error for no entry point, got nil")
	}
}

func TestValidatePipelineRejectsInvalidExecutionMode(t *testing.T) {
	spec := &PipelineSpec{
		Name:            "bad-mode",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{
				Name:           "entry",
				ValidEntryPoint: true,
				TimeoutSeconds:  5,
				ExecutionMode:   "invalid_mode",
			},
		},
	}

	if err := ValidatePipeline(spec); err == nil {
		t.Fatal("expected error for invalid execution_mode, got nil")
	}
}

func TestValidatePipelineRejectsInvalidOnEmpty(t *testing.T) {
	spec := &PipelineSpec{
		Name:            "bad-on-empty",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{
				Name:           "entry",
				ValidEntryPoint: true,
				TimeoutSeconds:  5,
				OnEmpty:        "explode",
			},
		},
	}

	if err := ValidatePipeline(spec); err == nil {
		t.Fatal("expected error for invalid on_empty, got nil")
	}
}

func TestValidatePipelineRejectsInvalidOnAllFailed(t *testing.T) {
	spec := &PipelineSpec{
		Name:            "bad-on-all-failed",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{
				Name:           "entry",
				ValidEntryPoint: true,
				TimeoutSeconds:  5,
				OnAllFailed:    "panic",
			},
		},
	}

	if err := ValidatePipeline(spec); err == nil {
		t.Fatal("expected error for invalid on_all_failed, got nil")
	}
}

func TestValidatePipelineRejectsDuplicateSlotNames(t *testing.T) {
	spec := &PipelineSpec{
		Name:            "dup-slots",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", ValidEntryPoint: true, TimeoutSeconds: 5},
			{Name: "entry", ValidEntryPoint: false},
		},
	}

	if err := ValidatePipeline(spec); err == nil {
		t.Fatal("expected error for duplicate slot names, got nil")
	}
}

func TestValidatePipelineRejectsRequiredSlotWithoutTimeout(t *testing.T) {
	spec := &PipelineSpec{
		Name:            "no-timeout",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", Required: true, ValidEntryPoint: true, TimeoutSeconds: 0},
		},
	}

	if err := ValidatePipeline(spec); err == nil {
		t.Fatal("expected error for required slot without timeout, got nil")
	}
}

func TestValidatePipelineRejectsBadVersion(t *testing.T) {
	spec := &PipelineSpec{
		Name:            "bad-version",
		PipelineVersion: 2,
		Slots: []SlotSpec{
			{Name: "entry", ValidEntryPoint: true, TimeoutSeconds: 5},
		},
	}

	if err := ValidatePipeline(spec); err == nil {
		t.Fatal("expected error for unsupported pipeline_version, got nil")
	}
}

func TestValidatePipelineDefaultsZeroVersion(t *testing.T) {
	spec := &PipelineSpec{
		Name: "default-version",
		Slots: []SlotSpec{
			{Name: "entry", ValidEntryPoint: true, Required: true, TimeoutSeconds: 5},
		},
	}

	if err := ValidatePipeline(spec); err != nil {
		t.Fatalf("expected zero version to default to 1, got: %v", err)
	}
	if spec.PipelineVersion != 1 {
		t.Fatalf("expected PipelineVersion to be defaulted to 1, got %d", spec.PipelineVersion)
	}
}

// --- Dispatch table test ---

func TestCompileDispatchTable(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	// Register a plugin with slot registrations
	plugin1 := &registry.PluginEntry{
		ID: "plugin-1",
		SlotRegistrations: []registry.SlotRegistration{
			{Pipeline: "my-pipeline", Slot: "entry", Priority: 1},
			{Pipeline: "my-pipeline", Slot: "process", Priority: 2},
		},
	}
	plugin2 := &registry.PluginEntry{
		ID: "plugin-2",
		SlotRegistrations: []registry.SlotRegistration{
			{Pipeline: "my-pipeline", Slot: "entry", Priority: 3},
		},
	}
	reg.Register(plugin1)
	reg.Register(plugin2)

	r.LoadPipeline(&PipelineSpec{
		Name:            "my-pipeline",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", Required: true, ValidEntryPoint: true, TimeoutSeconds: 5},
			{Name: "process", Required: false},
		},
	})

	dt := r.CompileDispatchTable()

	slotMap, ok := dt["my-pipeline"]
	if !ok {
		t.Fatal("expected my-pipeline in dispatch table")
	}

	entryPlugins := slotMap["entry"]
	if len(entryPlugins) != 2 {
		t.Fatalf("expected 2 plugins for entry slot, got %d", len(entryPlugins))
	}

	processPlugins := slotMap["process"]
	if len(processPlugins) != 1 {
		t.Fatalf("expected 1 plugin for process slot, got %d", len(processPlugins))
	}
	if processPlugins[0] != "plugin-1" {
		t.Fatalf("expected plugin-1 for process slot, got %s", processPlugins[0])
	}
}

// --- SlotSpec fields test ---

func TestSlotSpecFields(t *testing.T) {
	slot := SlotSpec{
		Name:                 "cognitive_processing",
		Capability:           "COGNITION",
		Required:             true,
		AllowsMultiplePlugins: false,
		ExecutionMode:        "sequential_by_priority",
		ValidEntryPoint:      false,
		TimeoutSeconds:       60,
		OnEmpty:              "fail",
		OnAllFailed:          "retry",
	}

	if slot.Name != "cognitive_processing" {
		t.Fatalf("expected Name cognitive_processing, got %s", slot.Name)
	}
	if slot.Capability != "COGNITION" {
		t.Fatalf("expected Capability COGNITION, got %s", slot.Capability)
	}
	if slot.AllowsMultiplePlugins {
		t.Fatal("expected AllowsMultiplePlugins=false")
	}
	if slot.ExecutionMode != "sequential_by_priority" {
		t.Fatalf("expected ExecutionMode sequential_by_priority, got %s", slot.ExecutionMode)
	}
	if slot.OnEmpty != "fail" {
		t.Fatalf("expected OnEmpty fail, got %s", slot.OnEmpty)
	}
	if slot.OnAllFailed != "retry" {
		t.Fatalf("expected OnAllFailed retry, got %s", slot.OnAllFailed)
	}
}

// --- Backward compat: JSON with old fields still works ---

func TestParsePipelineSpecBackwardCompat(t *testing.T) {
	raw := map[string]interface{}{
		"name": "old-style-pipeline",
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
		t.Fatalf("ParsePipelineSpec() backward compat failed: %v", err)
	}
	if spec.Name != "old-style-pipeline" {
		t.Fatalf("expected name old-style-pipeline, got %s", spec.Name)
	}
	if len(spec.Slots) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(spec.Slots))
	}
	if spec.Slots[0].Capability != "COGNITION" {
		t.Fatalf("expected capability COGNITION, got %s", spec.Slots[0].Capability)
	}
}

func TestLoadPipelineRejectsInvalidSpec(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	// Pipeline with no entry point should be rejected by LoadPipeline
	spec := &PipelineSpec{
		Name:            "invalid-no-entry",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "process", Required: false, ValidEntryPoint: false},
		},
	}

	if err := r.LoadPipeline(spec); err == nil {
		t.Fatal("expected LoadPipeline to reject invalid spec, got nil")
	}
}