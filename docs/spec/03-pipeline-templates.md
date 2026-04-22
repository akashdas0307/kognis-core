# Pipeline Templates

> **Stability:** EVOLVING
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [01-message-envelope.md](01-message-envelope.md), [04-handshake-protocols.md](04-handshake-protocols.md), [08-plugin-lifecycle.md](08-plugin-lifecycle.md)

## 3.1 Purpose

Pipeline templates are the canonical processing flows the framework ships with. They define the ordered slots that messages flow through for different types of processing. Plugins do not create pipelines — they register for slots in existing pipelines.

## 3.2 Template Schema

```yaml
# pipelines/<pipeline_id>.yaml
pipeline_version: 1
pipeline_id: string
description: string

# Message types that can enter this pipeline
accepted_message_types: array

# Slots in order
slots:
  - slot_id: string
    required: boolean              # Must have at least one plugin
    allows_multiple_plugins: boolean
    execution_mode: string         # "sequential_by_priority" | "parallel" | "by_action_type" | "by_channel_match"
    valid_entry_point: boolean     # Can envelope enter pipeline here
    timeout_seconds: integer       # Total time budget for this slot
    on_empty: string               # "skip" | "fail" | "buffer"
    on_all_failed: string          # "skip" | "fail" | "retry"
```

## 3.3 Canonical Pipeline Catalog

Kognis ships with these canonical pipelines:

### 3.3.1 `user_text_interaction`

User sends text, system processes and responds.

```yaml
pipeline_id: user_text_interaction
description: "User text message flows through perception, cognition, and response"
accepted_message_types: [user_text_input]

slots:
  - slot_id: input_reception
    required: true
    allows_multiple_plugins: true
    execution_mode: parallel
    valid_entry_point: true
    timeout_seconds: 5

  - slot_id: input_enrichment
    required: false
    allows_multiple_plugins: true
    execution_mode: sequential_by_priority
    valid_entry_point: false
    timeout_seconds: 10
    on_empty: skip

  - slot_id: cognitive_processing
    required: true
    allows_multiple_plugins: false
    execution_mode: sequential_by_priority
    valid_entry_point: false
    timeout_seconds: 60
    on_all_failed: fail

  - slot_id: action_review
    required: false
    allows_multiple_plugins: false
    execution_mode: sequential_by_priority
    valid_entry_point: false
    timeout_seconds: 15
    on_empty: skip

  - slot_id: action_execution
    required: true
    allows_multiple_plugins: true
    execution_mode: by_action_type
    valid_entry_point: false
    timeout_seconds: 30

  - slot_id: output_delivery
    required: true
    allows_multiple_plugins: true
    execution_mode: by_channel_match
    valid_entry_point: false
    timeout_seconds: 10
```

### 3.3.2 `user_voice_interaction`

Voice input variant. Same structure, different input types.

### 3.3.3 `background_monitoring`

Ambient events. Low priority, may or may not escalate to cognitive processing.

```yaml
pipeline_id: background_monitoring
description: "Ambient observations that may or may not require cognitive attention"
accepted_message_types: [ambient_audio, visual_frame, eal_escalation]

slots:
  - slot_id: ambient_assessment
    required: true
    valid_entry_point: true

  - slot_id: significance_gate
    required: true
    # If significant, continues; else, logs and exits

  - slot_id: cognitive_processing
    required: false  # only if significance_gate says so

  - slot_id: action_execution
    required: false

  - slot_id: output_delivery
    required: false
```

### 3.3.4 `autonomous_cognition`

Internal triggers — daydream, wake handoff, self-initiated thought.

```yaml
pipeline_id: autonomous_cognition
description: "System-initiated cognition without external input"
accepted_message_types: [wake_up_handoff, daydream_seed, internal_trigger]

slots:
  - slot_id: internal_trigger
    required: false
    valid_entry_point: true

  - slot_id: input_reception
    required: false
    valid_entry_point: true

  - slot_id: cognitive_processing
    required: true
    valid_entry_point: true

  - slot_id: action_review
    required: false

  - slot_id: action_execution
    required: false

  - slot_id: output_delivery
    required: false
```

### 3.3.5 `sleep_consolidation`

Sleep-time processing. Runs during Sleep Stage 3.

### 3.3.6 `health_management`

Health alerts and diagnostic flows.

### 3.3.7 `offspring_evaluation`

Offspring system uses this for isolated testing.

## 3.4 How the Router Uses Templates

On startup, and whenever plugins change:

1. Router reads all pipeline templates from `pipelines/*.yaml`
2. Router reads all plugin manifests
3. For each pipeline × slot combination:
   - Find all plugins registered for that slot
   - Sort by priority
   - Validate required slots have at least one plugin
   - Compile dispatch table: `{pipeline_id -> {slot_id -> [plugin_ids]}}`
4. Store dispatch table in memory
5. On message dispatch: consult dispatch table, route to next eligible plugin

When a plugin registers or dies:
- Only affected pipelines recompile
- Messages currently in flight use pre-change dispatch table
- New messages use updated dispatch table

## 3.5 Adding New Pipelines

Plugins can contribute new pipeline templates — but this is a heavyweight extension:

- Manifest declares `contributes_pipeline_template: <path>`
- Template validated against schema
- Template ID cannot conflict with existing
- Template added to catalog at runtime
- All plugins can then register for its slots

Use sparingly. Default to using canonical pipelines.