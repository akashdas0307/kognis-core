# Kognis YAML Examples — Canonical Reference

> **Purpose:** Complete, validated YAML examples for every structure in Kognis
> **Stability:** EVOLVING (examples update as specs evolve)
> **Usage:** Copy these as starting points. All examples are valid against schemas.

---

## Table of Contents

1. [Complete Plugin Manifest](#1-complete-plugin-manifest)
2. [Minimal Plugin Manifest](#2-minimal-plugin-manifest)
3. [Stateful Agent Plugin Manifest](#3-stateful-agent-plugin-manifest)
4. [Pipeline Templates](#4-pipeline-templates)
5. [Message Envelope Examples](#5-message-envelope-examples)
6. [Health Pulse](#6-health-pulse)
7. [State Broadcast](#7-state-broadcast)
8. [Registry Entry](#8-registry-entry)
9. [OMC Team Configuration](#9-omc-team-configuration)
10. [Milestone Definition](#10-milestone-definition)
11. [Handoff Document](#11-handoff-document)

---

## 1. Complete Plugin Manifest

A fully-populated manifest showing all possible fields. Use as reference when authoring plugins.

```yaml
# plugin.yaml — Kognis Plugin Manifest v1
manifest_version: 1

# ============================================================
# IDENTITY
# ============================================================
plugin_id: eal
plugin_name: "Environmental Awareness Layer"
version: 0.1.0
author: "Kognis Core Team"
license: "MIT"
description: "Continuous ambient environmental monitoring with baseline learning and deviation detection"
homepage: "https://github.com/kognis-framework/kognis-registry/tree/main/official/eal"

# ============================================================
# RUNTIME
# ============================================================
language: python
runtime:
  entrypoint: "python -m kognis_eal"
  working_directory: "."
  environment_requirements:
    python_version: ">=3.11"
    system_packages:
      - "portaudio19-dev"   # Linux
    external_commands: []
  resource_limits:
    max_memory_mb: 500
    max_cpu_percent: 20

# ============================================================
# SDK COMPATIBILITY
# ============================================================
sdk:
  required_version: ">=0.1.0,<1.0.0"
  manifest_schema: 1

# ============================================================
# HANDLER MODE
# ============================================================
handler_mode: stateless   # or "stateful_agent"

# ============================================================
# SLOT REGISTRATIONS
# ============================================================
slot_registrations:
  - pipeline: user_text_interaction
    slot: input_enrichment
    priority: 30
    message_types_handled:
      - user_text_input
      - voice_input
    message_types_produced: []
    timeout_seconds: 2
    retry_attempts: 1
    optional: true
    max_concurrent: 4

  - pipeline: background_monitoring
    slot: ambient_assessment
    priority: 10
    message_types_handled:
      - ambient_audio
      - visual_frame
    timeout_seconds: 3
    optional: false
    max_concurrent: 2

# ============================================================
# PROVIDED CAPABILITIES
# ============================================================
provides_capabilities:
  - capability_id: eal.get_environment_summary
    description: "Returns current environmental baseline status and any active deviations"
    params_schema:
      type: object
      properties:
        time_window_seconds:
          type: integer
          default: 300
          minimum: 10
          maximum: 3600
      required: []
    response_schema:
      type: object
      properties:
        baseline_status:
          type: string
          enum: [nominal, learning, degraded]
        baseline_summary:
          type: string
        current_deviations:
          type: array
          items:
            type: object
        ambient_profile:
          type: object
      required: [baseline_status, baseline_summary, current_deviations]
    latency_class: fast
    llm_tool_description: "Check what's currently happening in the physical environment around the being"
    llm_tool_expose_to:
      - cognitive_core
      - world_model
    authentication_required: false

  - capability_id: eal.reset_baseline
    description: "Reset environmental baseline (used when moving to new environment)"
    params_schema:
      type: object
      properties:
        reason:
          type: string
      required: [reason]
    response_schema:
      type: object
      properties:
        reset_at:
          type: string
          format: date-time
    latency_class: fast
    llm_tool_expose_to: []   # Not exposed to LLM — control operation only
    authentication_required: true

# ============================================================
# REQUIRED CAPABILITIES (from other plugins)
# ============================================================
requires_capabilities:
  - capability_id: inference.complete
    optional: true   # Only needed for curiosity exploration
  - capability_id: memory.store_episode
    optional: true

# ============================================================
# EVENT PUB/SUB
# ============================================================
event_subscriptions:
  - topic: audio.frame_processed
    handler: handle_audio_frame
    queue_group: eal_audio_workers

event_publications:
  - topic: eal.deviation_detected
    schema_ref: "schemas/eal_deviation_v1.yaml"
    description: "Published when environmental baseline deviation detected"
  - topic: eal.baseline_changed
    schema_ref: "schemas/eal_baseline_v1.yaml"
    description: "Published when baseline profile updates"

# ============================================================
# STATE BROADCASTS
# ============================================================
state_broadcasts:
  - state_name: monitoring_mode
    description: "Current operational mode"
    values:
      - active
      - sleep_mode
      - paused
      - learning_baseline
    change_topic: state.eal.monitoring_mode

  - state_name: baseline_confidence
    description: "Confidence level in current baseline (0.0-1.0)"
    values: numeric
    change_topic: state.eal.baseline_confidence

# ============================================================
# HEALTH REPORTING
# ============================================================
health:
  pulse_interval_seconds: 10
  critical_metrics:
    - name: baseline_status
      type: gauge
      unit: categorical
    - name: deviations_per_minute
      type: counter
      unit: count
    - name: audio_buffer_depth
      type: gauge
      unit: count
    - name: processing_latency_p99_ms
      type: gauge
      unit: milliseconds
  alert_conditions:
    - condition: "audio_buffer_depth > 100"
      severity: warning
      message: "Audio input backing up"
    - condition: "processing_latency_p99_ms > 1000"
      severity: error
      message: "EAL processing too slow"

# ============================================================
# SLEEP STAGE BEHAVIOR
# ============================================================
sleep_behavior:
  stage_1_settling: continue_normal
  stage_2_maintenance: reduced_activity
  stage_3_deep_consolidation: monitoring_only
  stage_4_pre_wake: continue_normal
  maintenance_jobs:
    - name: baseline_optimization
      description: "Reorganize baseline model for efficiency"
      estimated_duration_seconds: 60
    - name: event_log_pruning
      description: "Remove old deviation event logs"
      estimated_duration_seconds: 30

# ============================================================
# PERMISSIONS
# ============================================================
permissions:
  filesystem:
    read:
      - "/tmp/audio_stream"
    write:
      - "~/.kognis/eal/"
    deny:
      - "~/.ssh/"
      - "/etc/"
  network:
    allowed_domains: []      # EAL doesn't need network
    allowed_ports: []
  hardware:
    microphone: true
    camera: false
    gpu: false

# ============================================================
# EMERGENCY BYPASS AUTHORIZATION
# ============================================================
emergency_bypass:
  - bypass_type: safety_sound_detected
    rationale: "EAL's audio subsystem detects fire alarms, smoke detectors, glass breaking, etc."

# ============================================================
# UI CONTRIBUTION
# ============================================================
ui:
  type: status_panel
  icon: "🌍"
  default_shortcut: "2"
  summary_data_source: "/tmp/kognis_eal.summary.json"
  summary_update_interval_seconds: 30
  panel_width: medium
  panel_priority: 20     # Higher = more prominent in dashboard

# ============================================================
# MATURITY GATE
# ============================================================
maturity_gate:
  minimum_stage: infancy   # Available from day one
  minimum_age_days: 0

# ============================================================
# STARTUP DEPENDENCIES
# ============================================================
startup_dependencies:
  hard: []                 # No hard dependencies
  soft:
    - memory
    - inference-gateway

# ============================================================
# CONFIGURATION
# ============================================================
configuration:
  schema:
    type: object
    properties:
      audio_sample_rate:
        type: integer
        default: 16000
      baseline_learning_duration_minutes:
        type: integer
        default: 30
      deviation_threshold:
        type: number
        default: 0.3
        minimum: 0.0
        maximum: 1.0
      curiosity_enabled:
        type: boolean
        default: true
  defaults:
    audio_sample_rate: 16000
    baseline_learning_duration_minutes: 30
    deviation_threshold: 0.3
    curiosity_enabled: true
```

---

## 2. Minimal Plugin Manifest

The smallest viable manifest for a stateless handler plugin:

```yaml
manifest_version: 1

plugin_id: echo-chat
plugin_name: "Echo Chat"
version: 0.1.0
author: "Example"
license: "MIT"
description: "Minimal chat plugin that echoes input"

language: python
runtime:
  entrypoint: "python -m echo_chat"

sdk:
  required_version: ">=0.1.0"
  manifest_schema: 1

handler_mode: stateless

slot_registrations:
  - pipeline: user_text_interaction
    slot: input_reception
    priority: 50
    message_types_produced: [user_text_input]
  - pipeline: user_text_interaction
    slot: output_delivery
    priority: 50
    message_types_handled: [assistant_response]

ui:
  type: interactive_view
  icon: "💬"
  launch_command: "python -m echo_chat --interactive"
  default_shortcut: "1"

permissions:
  filesystem:
    write: ["~/.kognis/echo-chat/"]

startup_dependencies:
  hard: []
  soft: []
```

---

## 3. Stateful Agent Plugin Manifest

Example for Cognitive Core — a stateful agent:

```yaml
manifest_version: 1

plugin_id: cognitive-core
plugin_name: "Cognitive Core"
version: 0.1.0
author: "Kognis Core Team"
license: "MIT"
description: "Central stateful agent responsible for inner monologue and reasoning"

language: python
runtime:
  entrypoint: "python -m kognis_cognitive_core"
  environment_requirements:
    python_version: ">=3.11"
  resource_limits:
    max_memory_mb: 2000

sdk:
  required_version: ">=0.1.0"
  manifest_schema: 1

handler_mode: stateful_agent   # <-- Key difference

slot_registrations:
  - pipeline: user_text_interaction
    slot: cognitive_processing
    priority: 100
    timeout_seconds: 60
    max_concurrent: 1   # Stateful agent serializes

  - pipeline: user_voice_interaction
    slot: cognitive_processing
    priority: 100
    timeout_seconds: 60
    max_concurrent: 1

  - pipeline: autonomous_cognition
    slot: cognitive_processing
    priority: 100
    timeout_seconds: 300   # Daydream sessions can be long
    max_concurrent: 1

provides_capabilities:
  - capability_id: cognitive_core.get_current_activity
    description: "Get description of what the being is currently thinking about"
    params_schema:
      type: object
    response_schema:
      type: object
      properties:
        activity_state: {type: string}
        current_focus: {type: string}
        started_at: {type: string}
    latency_class: fast
    llm_tool_expose_to: []   # Internal query only

requires_capabilities:
  - capability_id: inference.complete
    optional: false
  - capability_id: memory.retrieve_episodes
    optional: false
  - capability_id: persona.get_identity_block
    optional: false
  - capability_id: world_model.review_proposed_action
    optional: true
  - capability_id: eal.get_environment_summary
    optional: true

event_publications:
  - topic: cognitive_core.monologue_chunk
    description: "Published for observability — inner monologue as it happens"
  - topic: cognitive_core.decision_made
    description: "Published when a decision is finalized"

state_broadcasts:
  - state_name: activity_state
    values:
      - idle
      - reasoning
      - daydreaming
      - processing_input
      - awaiting_capability
      - emergency_response
    change_topic: state.cognitive_core.activity_state

  - state_name: current_focus
    values: text
    change_topic: state.cognitive_core.current_focus

health:
  pulse_interval_seconds: 5
  critical_metrics:
    - name: reasoning_sessions_active
      type: gauge
    - name: inference_calls_per_minute
      type: counter
    - name: context_tokens_used
      type: gauge
    - name: context_trim_events
      type: counter

sleep_behavior:
  stage_1_settling: continue_normal
  stage_2_maintenance: reduced_activity
  stage_3_deep_consolidation: monitoring_only   # Cognitive Core pauses
  stage_4_pre_wake: continue_normal

permissions:
  filesystem:
    write: ["~/.kognis/cognitive-core/"]
  network:
    allowed_domains: []   # Network goes through Inference Gateway

maturity_gate:
  minimum_stage: infancy
  minimum_age_days: 0

startup_dependencies:
  hard:
    - inference-gateway
    - memory
    - persona-manager
  soft:
    - world-model
    - eal
```

---

## 4. Pipeline Templates

### 4.1 User Text Interaction

```yaml
# pipelines/user_text_interaction.yaml
pipeline_version: 1
pipeline_id: user_text_interaction
description: "User text message flows through perception, cognition, and response"

accepted_message_types:
  - user_text_input

slots:
  - slot_id: input_reception
    description: "Where input originates"
    required: true
    allows_multiple_plugins: true
    execution_mode: parallel
    valid_entry_point: true
    timeout_seconds: 5
    on_empty: fail
    on_all_failed: fail

  - slot_id: input_enrichment
    description: "Add context to input"
    required: false
    allows_multiple_plugins: true
    execution_mode: sequential_by_priority
    valid_entry_point: false
    timeout_seconds: 10
    on_empty: skip
    on_all_failed: skip

  - slot_id: cognitive_processing
    description: "Core reasoning about input"
    required: true
    allows_multiple_plugins: false
    execution_mode: sequential_by_priority
    valid_entry_point: false
    timeout_seconds: 60
    on_empty: fail
    on_all_failed: fail

  - slot_id: action_review
    description: "Review proposed actions before execution"
    required: false
    allows_multiple_plugins: false
    execution_mode: sequential_by_priority
    valid_entry_point: false
    timeout_seconds: 15
    on_empty: skip
    on_all_failed: skip

  - slot_id: action_execution
    description: "Execute approved actions"
    required: true
    allows_multiple_plugins: true
    execution_mode: by_action_type
    valid_entry_point: false
    timeout_seconds: 30
    on_empty: fail
    on_all_failed: fail

  - slot_id: output_delivery
    description: "Deliver response to user"
    required: true
    allows_multiple_plugins: true
    execution_mode: by_channel_match
    valid_entry_point: false
    timeout_seconds: 10
    on_empty: fail
    on_all_failed: fail
```

### 4.2 Autonomous Cognition

```yaml
pipeline_version: 1
pipeline_id: autonomous_cognition
description: "System-initiated cognition without external input (daydream, wake handoff, internal triggers)"

accepted_message_types:
  - wake_up_handoff
  - daydream_seed
  - internal_trigger

slots:
  - slot_id: internal_trigger
    required: false
    allows_multiple_plugins: true
    execution_mode: parallel
    valid_entry_point: true
    timeout_seconds: 5

  - slot_id: context_assembly
    required: false
    allows_multiple_plugins: false
    execution_mode: sequential_by_priority
    valid_entry_point: true
    timeout_seconds: 15

  - slot_id: cognitive_processing
    required: true
    allows_multiple_plugins: false
    execution_mode: sequential_by_priority
    valid_entry_point: true
    timeout_seconds: 300   # Daydreams can be long

  - slot_id: action_review
    required: false
    allows_multiple_plugins: false
    execution_mode: sequential_by_priority
    valid_entry_point: false
    timeout_seconds: 15
    on_empty: skip

  - slot_id: action_execution
    required: false
    allows_multiple_plugins: true
    execution_mode: by_action_type
    valid_entry_point: false
    timeout_seconds: 30
    on_empty: skip

  - slot_id: output_delivery
    required: false
    allows_multiple_plugins: true
    execution_mode: by_channel_match
    valid_entry_point: false
    timeout_seconds: 10
    on_empty: skip
```

### 4.3 Background Monitoring

```yaml
pipeline_version: 1
pipeline_id: background_monitoring
description: "Ambient observations — may escalate to cognitive attention or just log"

accepted_message_types:
  - ambient_audio
  - visual_frame
  - eal_escalation

slots:
  - slot_id: ambient_assessment
    required: true
    allows_multiple_plugins: true
    execution_mode: parallel
    valid_entry_point: true
    timeout_seconds: 3

  - slot_id: significance_gate
    required: true
    allows_multiple_plugins: false
    execution_mode: sequential_by_priority
    valid_entry_point: false
    timeout_seconds: 2
    # If not significant, pipeline terminates here (logs to env context)

  - slot_id: cognitive_processing
    required: false   # Only if significance_gate escalates
    allows_multiple_plugins: false
    execution_mode: sequential_by_priority
    valid_entry_point: false
    timeout_seconds: 60

  - slot_id: output_delivery
    required: false
    allows_multiple_plugins: true
    execution_mode: by_channel_match
    valid_entry_point: false
    timeout_seconds: 10
```

---

## 5. Message Envelope Examples

### 5.1 User Text Input (Initial)

```yaml
envelope_version: 1
id: "msg-abc-001"
created_at: "2026-04-21T09:15:02.123Z"
origin_plugin: "chat-tui"
message_type: "user_text_input"

payload:
  user_id: "akash"
  text: "How's the rice yield data looking?"
  channel: "system_gui"
  input_metadata:
    input_method: "keyboard"
    typing_duration_ms: 1200

routing:
  pipeline: "user_text_interaction"
  completed_stages:
    - "input_reception"
  current_stage: null
  failed_stages: []
  hop_count: 1
  entry_slot: "input_reception"

enrichments: {}

metadata:
  priority: "tier_3_normal"
  trust_level: "tier_1_creator"
  trace_id: "trace-xyz-001"
  revision_count: 0
  parent_envelope_id: null
  correlation_id: null
```

### 5.2 After EAL Enrichment

```yaml
envelope_version: 1
id: "msg-abc-001"   # Same ID
created_at: "2026-04-21T09:15:02.123Z"
origin_plugin: "chat-tui"
message_type: "user_text_input"

payload:
  user_id: "akash"
  text: "How's the rice yield data looking?"
  channel: "system_gui"

routing:
  pipeline: "user_text_interaction"
  completed_stages:
    - "input_reception"
    - "input_enrichment"
  current_stage: null
  failed_stages: []
  hop_count: 2
  entry_slot: "input_reception"

enrichments:
  environment:
    summary: "quiet morning, home office, baseline nominal"
    deviations: []
    ambient_sound_level_db: 32
    timestamp: "2026-04-21T09:15:03.456Z"
    confidence: 0.92

metadata:
  priority: "tier_3_normal"
  trust_level: "tier_1_creator"
  trace_id: "trace-xyz-001"
  revision_count: 0
```

### 5.3 Action Request (from Cognitive Core)

```yaml
envelope_version: 1
id: "msg-abc-002"                 # NEW envelope
created_at: "2026-04-21T09:15:07.891Z"
origin_plugin: "cognitive-core"
message_type: "action_request"

payload:
  action_type: "send_chat"
  target: "akash"
  channel: "system_gui"
  text: "Looking at the data — yield seems 12% above last season..."
  emotional_tone:
    valence: 0.65
    arousal: 0.40
    engagement: 0.80
    confidence: 0.72
    warmth: 0.75

routing:
  pipeline: "user_text_interaction"
  completed_stages:
    - "input_reception"
    - "input_enrichment"
    - "cognitive_processing"
  current_stage: null
  failed_stages: []
  hop_count: 3
  entry_slot: "input_reception"

enrichments:
  environment:
    summary: "quiet morning, home office, baseline nominal"
  context:
    related_episodes_referenced: 3
    reasoning_chain_length: 4

metadata:
  priority: "tier_3_normal"
  trust_level: "tier_1_creator"
  trace_id: "trace-xyz-001"
  revision_count: 0
  parent_envelope_id: "msg-abc-001"
  correlation_id: "chat-exchange-001"
```

### 5.4 Internal Trigger (Daydream Seed)

```yaml
envelope_version: 1
id: "msg-internal-042"
created_at: "2026-04-21T11:32:00.000Z"
origin_plugin: "cognitive-core"
message_type: "daydream_seed"

payload:
  seed_type: "random_memory_sample"
  seed_content:
    memory_id: "mem-x-4823"
    summary: "Discussion about bamboo irrigation 2 days ago"
  trigger_reason: "idle_timeout_exceeded"
  idle_duration_minutes: 5

routing:
  pipeline: "autonomous_cognition"
  completed_stages: []
  current_stage: null
  failed_stages: []
  hop_count: 0
  entry_slot: "cognitive_processing"   # Enters at cognitive_processing slot

enrichments: {}

metadata:
  priority: "tier_3_normal"
  trust_level: "internal"
  trace_id: "trace-daydream-001"
```

---

## 6. Health Pulse

```yaml
pulse_version: 1
plugin_id: eal
timestamp: "2026-04-21T09:15:00.000Z"
sequence_number: 4523
status: HEALTHY

metrics:
  queue_depth: 3
  processing_latency_p50_ms: 145
  processing_latency_p99_ms: 380
  error_count_last_60s: 0
  memory_usage_mb: 187
  cpu_percent: 4.2
  audio_buffer_depth: 2
  baseline_status: "nominal"
  deviations_per_minute: 0
  last_deviation_at: "2026-04-21T04:12:15.000Z"

current_activity: "Monitoring ambient audio, baseline stable"
last_dispatch_at: "2026-04-21T09:14:58.123Z"

alerts: []

uptime_seconds: 54000
restart_count: 0
```

---

## 7. State Broadcast

```yaml
# Event published to: state.cognitive_core.activity_state
broadcast_version: 1
plugin_id: cognitive-core
state_name: activity_state
timestamp: "2026-04-21T09:15:04.567Z"

previous_value: "idle"
new_value: "reasoning"

context:
  triggered_by: "dispatch:msg-abc-001"
  estimated_duration_seconds: 10
  
source_sequence: 8912
```

---

## 8. Registry Entry

Used by `kognis-registry` repo.

```yaml
# Entry in registry.yaml — one per plugin
name: eal
display_name: "Environmental Awareness Layer"
version: 0.1.0

source:
  type: bundled                      # "bundled" | "github"
  path: official/eal                 # For bundled
  # For github:
  # repo: https://github.com/someuser/plugin-name
  # tag: v1.0.0
  # subpath: src/plugin              # Optional

maintainer: kognis-core-team
verified: true
verification_date: "2026-04-15"

category: perception
tags:
  - environmental
  - audio
  - monitoring

description: "Continuous ambient environmental monitoring with baseline learning and deviation detection"
short_description: "Environmental awareness"

documentation_url: "https://github.com/kognis-framework/kognis-registry/blob/main/official/eal/README.md"
issue_tracker: "https://github.com/kognis-framework/kognis-registry/issues"

versions_tested:
  - "0.1.0"

required_for_pipelines: []
optional_for_pipelines:
  - user_text_interaction
  - background_monitoring

permissions_requested:
  - hardware.microphone
  - filesystem.write:~/.kognis/eal

size_estimate:
  install_size_mb: 45
  memory_usage_mb: 200

license: MIT

security_review:
  status: passed
  last_reviewed: "2026-04-15"
  reviewer: "core-team"
```

---

## 9. OMC Team Configuration

Used in `.claude/omc.jsonc`:

```jsonc
{
  "team": {
    "roleRouting": {
      // Orchestrator inherits the session's model
      "orchestrator": { "model": "inherit" },

      // Deep reasoning tasks → GLM-5.1
      "planner":       { "provider": "claude", "model": "glm-5.1:cloud" },
      "architect":     { "provider": "claude", "model": "glm-5.1:cloud" },
      "analyst":       { "provider": "claude", "model": "glm-5.1:cloud" },
      "debugger":      { "provider": "claude", "model": "glm-5.1:cloud" },

      // Execution → MiniMax M2.7 (cost-effective)
      "executor":      { "provider": "claude", "model": "minimax-m2.7:cloud" },
      "test-engineer": { "provider": "claude", "model": "minimax-m2.7:cloud" },
      "writer":        { "provider": "claude", "model": "minimax-m2.7:cloud" },
      "explore":       { "provider": "claude", "model": "minimax-m2.7:cloud" },

      // Review → Kimi K2.6 (thorough, visual)
      "code-reviewer":     { "provider": "claude", "model": "kimi-k2.6:cloud" },
      "security-reviewer": { "provider": "claude", "model": "kimi-k2.6:cloud" },
      "designer":          { "provider": "claude", "model": "kimi-k2.6:cloud" },
      "critic":            { "provider": "claude", "model": "kimi-k2.6:cloud" }
    },

    "ops": {
      "maxAgents": 15,
      "defaultAgentType": "claude",
      "monitorIntervalMs": 30000,
      "shutdownTimeoutMs": 15000
    }
  },

  "routing": {
    "tierModels": {
      "HIGH": "claude-opus-4-20250514",
      "MEDIUM": "claude-sonnet-4-20250514",
      "LOW": "claude-haiku-4-20250514"
    }
  }
}
```

---

## 10. Milestone Definition

```yaml
# milestones/M-001-manifest-parser.yaml (or .md with YAML frontmatter)
milestone:
  id: M-001
  name: "Plugin Manifest Parser"
  priority: P0
  effort: M                    # S/M/L/XL
  type: soft                   # soft (auto-merge) | hard (human review)
  target_branch: feature/M-001-manifest-parser
  dependencies: []

  goal: |
    Implement a Python library that parses plugin.yaml files, validates
    them against manifest schema v1, and produces typed Manifest dataclass
    objects suitable for plugin registration.

  context_files:
    - docs/spec/02-plugin-manifest.md
    - docs/spec/07-error-taxonomy.md
    - docs/YAML_EXAMPLES.md
    - docs/foundations/01-vision.md

  deliverables:
    - sdk/python/kognis_sdk/manifest.py
    - sdk/python/tests/test_manifest_parser.py
    - Updates to docs/YAML_EXAMPLES.md
    - CHANGELOG entry

  constraints:
    language: python
    python_version: ">=3.11"
    dependencies_allowed:
      - pyyaml
      - dataclasses (stdlib)
      - typing (stdlib)
    not_allowed:
      - pydantic    # Use dataclasses instead
      - custom YAML parsers

  success_criteria:
    - "25+ unit tests covering valid and invalid manifests"
    - "All manifest_version:1 fields represented in Manifest dataclass"
    - "All KGN-MANIFEST-* error codes raised appropriately"
    - "Parse time p99 < 10ms for typical manifest"
    - "mypy strict mode passes"
    - "ruff clean"
    - "Documentation includes usage example"

  out_of_scope:
    - Plugin loading logic (separate milestone)
    - Capability registration (separate milestone)
    - CLI wrapper (separate milestone)
    - JSON format support (YAML only)

  review_gates:
    - after_spec_read: confirm understanding
    - before_merge: verify error codes match taxonomy
```

---

## 11. Handoff Document

Used during multi-stage work:

```markdown
## Handoff: team-plan → team-exec

- **Milestone**: M-001 (Plugin Manifest Parser)
- **Timestamp**: 2026-04-21T14:30:00Z
- **Stage**: team-plan → team-exec

### Decided

- Use dataclasses (stdlib), not pydantic
- Parser validates in two passes: schema check, then semantic check
- Error codes follow KGN-MANIFEST-* taxonomy exactly
- Will provide both parse_file() and parse_string() entry points

### Rejected

- **pydantic**: dependency not allowed per milestone constraints
- **Schema libraries (cerberus, jsonschema)**: adds complexity; manual validation is sufficient for manifest scope
- **Lazy parsing**: all fields parsed upfront for clearer error messages

### Risks

- Some manifest fields have complex nested structures (permissions, UI). Need careful dataclass design.
- Backward compatibility strategy for manifest v2 not yet defined — out of scope but note for later.

### Files Created/Modified

- `docs/spec/02-plugin-manifest.md` — re-read, no changes needed
- `sdk/python/kognis_sdk/manifest.py` — new
- `sdk/python/tests/test_manifest_parser.py` — new
- `sdk/python/pyproject.toml` — added dependencies

### Remaining for Execution Stage

- Write comprehensive test cases
- Implement Manifest dataclass hierarchy
- Implement validation passes
- Update CHANGELOG
- Generate milestone report
```

---

## 12. Tips for Authoring YAML

1. **Always include `manifest_version` / `envelope_version` / similar** — schema evolution depends on this
2. **Use lowercase-with-hyphens** for IDs (not camelCase)
3. **Provide `description` everywhere** — future readers thank you
4. **Use enums instead of free-text** where values are constrained
5. **Nested objects for grouped fields** — don't flatten
6. **Comment liberally** in config files (YAML supports `#` comments)
7. **Validate against schema** before committing — use `scripts/validate-manifest.py`

---

*When writing new YAML, start from an example in this document. Do not improvise structure.*
