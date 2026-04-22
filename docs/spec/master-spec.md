# Kognis Framework — Master Specification Document

> **[SPLIT COMPLETE]** — This file has been split into individual spec files under `docs/spec/`.
> See the Table of Contents below for the list of split files.
> This file remains as a reading index only.

> **Stability:** EVOLVING (structures are settling; breaking changes will be versioned)
> **Version:** 0.1.0
> **Last updated:** April 2026
>
> **Instructions for AI agents reading this document:**
> This file is a CONSOLIDATED master document. Your first task during initialization
> is to split this into separate files under `docs/spec/` following the
> section markers (`# SPEC 01:`, `# SPEC 02:`, etc.). Preserve content exactly.
> Add cross-references between the split files. After splitting,
> leave this file in place as a reading index but mark it as `[SPLIT COMPLETE]`.

---

## Table of Contents — Split Plan

| Spec | Filename | Stability |
|---|---|---|
| 01 | `01-message-envelope.md` | EVOLVING |
| 02 | `02-plugin-manifest.md` | EVOLVING |
| 03 | `03-pipeline-templates.md` | EVOLVING |
| 04 | `04-handshake-protocols.md` | STABLE |
| 05 | `05-capability-registry.md` | STABLE |
| 06 | `06-state-broadcast.md` | STABLE |
| 07 | `07-error-taxonomy.md` | EVOLVING |
| 08 | `08-plugin-lifecycle.md` | STABLE |
| 09 | `09-mutation-semantics.md` | STABLE |
| 10 | `10-context-budget-manager.md` | EVOLVING |
| 11 | `11-tool-bridge.md` | STABLE |
| 12 | `12-durability-backup.md` | EVOLVING |
| 13 | `13-startup-dependency-order.md` | STABLE |
| 14 | `14-emergency-bypass.md` | STABLE |
| 15 | `15-emotional-state-vector.md` | STABLE |
| 16 | `16-sleep-stage-behaviors.md` | EVOLVING |
| 17 | `17-offspring-system.md` | EVOLVING |
| 18 | `18-health-pulse-schema.md` | STABLE |

---

# SPEC 01: Message Envelope

## 1.1 Purpose

The message envelope is the universal data format for anything flowing through the Kognis pipeline system. It is the lingua franca — every plugin speaks this.

## 1.2 Envelope Structure

```yaml
# Message envelope schema v1
envelope_version: 1

# Identity fields
id: string                        # Unique message ID (UUID v4)
created_at: string                 # ISO 8601 timestamp
origin_plugin: string              # Plugin ID that created this envelope
message_type: string               # See Message Types table below

# Payload — plugin-defined content
payload: object                    # Structure depends on message_type

# Routing state
routing:
  pipeline: string                 # Pipeline template ID
  completed_stages: array<string>  # Stages already processed
  current_stage: string | null     # Stage currently dispatched
  failed_stages: array<object>     # Stages that failed (with reason)
  hop_count: integer               # Increments on each dispatch (max 20 default)
  entry_slot: string               # Which slot this envelope entered at

# Enrichments — additive, never overwrites
enrichments:
  # Each plugin adds to its own namespace
  environment: object | null       # Added by EAL
  context: object | null           # Added by Prajna TLP
  memory: object | null            # Added by Memory
  # ... etc, namespace per plugin

# Metadata
metadata:
  priority: string                 # tier_1_immediate | tier_2_elevated | tier_3_normal
  trust_level: string              # tier_1_creator | tier_2_trusted | tier_3_external | internal
  trace_id: string                 # For distributed tracing
  revision_count: integer          # For action_review revision loops
  parent_envelope_id: string | null # If derived from another envelope
  correlation_id: string | null    # For request-response pairing
```

## 1.3 Message Types

| Type | Payload Shape | Source | Typical Pipeline |
|---|---|---|---|
| `user_text_input` | `{user_id, text, channel}` | Chat plugins | user_text_interaction |
| `voice_input` | `{user_id, transcript, audio_ref, emotional_tone}` | Voice plugin | user_voice_interaction |
| `ambient_audio` | `{sound_class, volume, duration, location_hint}` | Audio monitoring | background_monitoring |
| `visual_frame` | `{frame_ref, scene_summary, entities_detected}` | Visual plugin | background_monitoring |
| `assistant_response` | `{text, channel, emotional_tone, action_refs}` | Cognitive Core | any |
| `action_request` | `{action_type, params, constraints}` | Cognitive Core | action pipelines |
| `action_result` | `{action_request_id, success, output, error}` | Brainstem | feedback |
| `wake_up_handoff` | `{sleep_summary, trait_candidates, offspring_results, health_summary}` | Sleep/Dream | autonomous_cognition |
| `daydream_seed` | `{seed_type, seed_content, origin}` | Cognitive Core idle | autonomous_cognition |
| `eal_escalation` | `{deviation_type, significance, details}` | EAL | background_monitoring |
| `internal_trigger` | `{trigger_type, payload}` | Various | autonomous_cognition |
| `health_alert` | `{severity, plugin_id, issue, diagnostic}` | System Health | health_management |

This list is extensible — plugins can declare new message types in their manifest.

## 1.4 Envelope Constraints

- Envelopes are immutable once dispatched to a slot. Plugins do not modify envelopes in place. They produce new envelopes.
- Enrichments are additive only. A plugin adds to its own namespace. Does not touch others'.
- `hop_count` is enforced by router. If hop_count > 20, envelope is dead-lettered with `loop_detected` error.
- `revision_count` is incremented ONLY by action_review slot. Max 3 revisions.
- Payload serialization: JSON with optional binary refs (for audio/images — stored separately, referenced by ID).

## 1.5 Envelope Flow Example

```yaml
# Initial envelope from chat input
id: "msg-abc-001"
created_at: "2026-04-21T09:15:02Z"
origin_plugin: "chat-tui"
message_type: "user_text_input"
payload:
  user_id: "akash"
  text: "How's the rice yield data looking?"
  channel: "system_gui"
routing:
  pipeline: "user_text_interaction"
  completed_stages: ["input_reception"]
  current_stage: null
  hop_count: 1
  entry_slot: "input_reception"
enrichments: {}
metadata:
  priority: "tier_3_normal"
  trust_level: "tier_1_creator"
  trace_id: "trace-xyz-001"
  revision_count: 0

# After EAL enrichment
enrichments:
  environment:
    summary: "quiet morning, home office, baseline nominal"
    deviations: []
    timestamp: "2026-04-21T09:15:03Z"
routing:
  completed_stages: ["input_reception", "input_enrichment"]
  hop_count: 2

# After Prajna cognitive_processing (as result envelope)
id: "msg-abc-002"                  # NEW envelope
parent_envelope_id: "msg-abc-001"
message_type: "action_request"
payload:
  action_type: "send_chat"
  target: "akash"
  text: "Looking at the data..."
routing:
  pipeline: "user_text_interaction"
  completed_stages: ["input_reception", "input_enrichment", "cognitive_processing"]
  hop_count: 3
```

---

# SPEC 02: Plugin Manifest

## 2.1 Purpose

Every plugin declares its contract with the Kognis core through a `plugin.yaml` manifest at its root. This is the single source of truth for discovery, registration, routing, and capability management.

## 2.2 Complete Manifest Schema

```yaml
# plugin.yaml — Kognis plugin manifest v1
manifest_version: 1

# Identity
plugin_id: string                 # Unique identifier (lowercase-hyphenated)
plugin_name: string               # Human-readable name
version: string                   # Semantic version (e.g., 0.1.0)
author: string                    # Author/maintainer
license: string                   # SPDX license identifier
description: string               # Brief description

# Runtime
language: string                  # python | go | node | other
runtime:
  entrypoint: string              # How to start: e.g., "python -m my_plugin"
  working_directory: string       # Relative to plugin root
  environment_requirements:
    python_version: string        # If language is python
    system_packages: array        # OS-level dependencies
    external_commands: array      # Binaries that must be on PATH

# SDK compatibility
sdk:
  required_version: string        # e.g., ">=0.1.0"
  manifest_schema: integer        # This manifest's schema version

# Plugin handler mode — critical distinction
handler_mode: string              # "stateless" | "stateful_agent"

# Slot registrations — where this plugin fits in pipelines
slot_registrations:
  - pipeline: string              # Pipeline template ID
    slot: string                  # Slot name within that pipeline
    priority: integer             # Lower runs first (0-100)
    message_types_handled: array  # Which message_types this plugin processes
    message_types_produced: array # What it might emit
    timeout_seconds: integer      # Max processing time
    retry_attempts: integer       # On failure
    optional: boolean             # Can pipeline run without this plugin
    max_concurrent: integer       # Parallel dispatches allowed

# Capabilities provided
provides_capabilities:
  - capability_id: string         # e.g., "memory.retrieve_episodes"
    description: string
    params_schema:                # JSON schema for parameters
      type: object
      properties: {}
    response_schema:              # JSON schema for response
      type: object
      properties: {}
    latency_class: string         # "fast" | "medium" | "slow"
    llm_tool_description: string  # For Tool Bridge exposure (optional)
    llm_tool_expose_to:           # Which plugins' LLMs see this as a tool
      - cognitive_core
      - world_model
    authentication_required: boolean

# Capabilities required from others
requires_capabilities:
  - capability_id: string
    optional: boolean             # Plugin can run without this

# Events — pub/sub on event bus
event_subscriptions:
  - topic: string                 # e.g., "eal.deviation_detected"
    handler: string               # Name of handler function

event_publications:
  - topic: string
    schema_ref: string            # Reference to event schema

# State broadcast
state_broadcasts:
  - state_name: string            # e.g., "activity_state"
    description: string
    values: array                 # Possible values
    change_topic: string          # Topic to publish changes to

# Health reporting
health:
  pulse_interval_seconds: integer # How often to emit health pulse
  critical_metrics:               # Metrics included in pulses
    - name: string
      type: string                # "gauge" | "counter"
      unit: string
  alert_conditions:
    - condition: string           # e.g., "queue_depth > 1000"
      severity: string            # "warning" | "error" | "critical"

# Sleep stage behavior
sleep_behavior:
  stage_1_settling: string        # "continue_normal" | "reduced_activity" | "monitoring_only"
  stage_2_maintenance: string
  stage_3_deep_consolidation: string
  stage_4_pre_wake: string
  maintenance_jobs:               # What this plugin does during maintenance stage
    - name: string
      description: string
      estimated_duration_seconds: integer

# Permissions — what this plugin requires access to
permissions:
  filesystem:
    read: array                   # Paths this plugin reads
    write: array                  # Paths this plugin writes
  network:
    allowed_domains: array
    allowed_ports: array
  hardware:
    microphone: boolean
    camera: boolean
    gpu: boolean

# UI contribution
ui:
  type: string                    # "status_panel" | "interactive_view" | "background"
  launch_command: string          # For interactive_view: how to launch
  icon: string                    # Emoji or symbol
  default_shortcut: string        # Keyboard shortcut in dashboard
  summary_data_source: string     # Path or topic where summary data comes from
  summary_update_interval_seconds: integer

# Maturity requirements
maturity_gate:
  minimum_stage: string           # "infancy" | "childhood" | "adolescence" | "adult"
  minimum_age_days: integer

# Configuration schema
configuration:
  schema:                         # JSON schema for plugin config
    type: object
    properties: {}
  defaults:                       # Default values
    key: value
```

## 2.3 Manifest Example — EAL Plugin

```yaml
manifest_version: 1
plugin_id: eal
plugin_name: "Environmental Awareness Layer"
version: 0.1.0
author: "Kognis Core Team"
license: "MIT"
description: "Continuous ambient environmental monitoring with deviation detection"

language: python
runtime:
  entrypoint: "python -m kognis_eal"
  environment_requirements:
    python_version: ">=3.11"
    system_packages: []

sdk:
  required_version: ">=0.1.0"
  manifest_schema: 1

handler_mode: stateless

slot_registrations:
  - pipeline: user_text_interaction
    slot: input_enrichment
    priority: 30
    message_types_handled: [user_text_input, voice_input]
    timeout_seconds: 2
    optional: true
    max_concurrent: 4

provides_capabilities:
  - capability_id: eal.get_environment_summary
    description: "Returns current environmental baseline and any deviations"
    params_schema:
      type: object
      properties:
        time_window_seconds: {type: integer, default: 300}
    response_schema:
      type: object
      properties:
        baseline_status: {type: string}
        current_deviations: {type: array}
    latency_class: fast
    llm_tool_description: "Check what's happening in the physical environment"
    llm_tool_expose_to: [cognitive_core, world_model]

event_publications:
  - topic: eal.deviation_detected
    schema_ref: "schemas/eal_deviation_v1.yaml"
  - topic: eal.baseline_changed
    schema_ref: "schemas/eal_baseline_v1.yaml"

state_broadcasts:
  - state_name: monitoring_mode
    values: [active, sleep_mode, paused]
    change_topic: state.eal.monitoring_mode

health:
  pulse_interval_seconds: 10
  critical_metrics:
    - name: baseline_status
      type: gauge
      unit: categorical
    - name: deviations_per_minute
      type: counter
      unit: count

sleep_behavior:
  stage_1_settling: continue_normal
  stage_2_maintenance: reduced_activity
  stage_3_deep_consolidation: monitoring_only
  stage_4_pre_wake: continue_normal

permissions:
  filesystem:
    read: ["/tmp/audio_stream"]
    write: ["~/.kognis/eal/"]
  hardware:
    microphone: true

ui:
  type: status_panel
  icon: "🌍"
  summary_data_source: "/tmp/kognis_eal.summary.json"
  summary_update_interval_seconds: 30

maturity_gate:
  minimum_stage: infancy
  minimum_age_days: 0
```

## 2.4 Manifest Validation

The core validates manifests during plugin registration:

1. **Schema validation** — matches manifest_version=1 schema
2. **Identity uniqueness** — plugin_id not already registered
3. **SDK compatibility** — required_version matches installed SDK
4. **Pipeline references** — all pipelines and slots exist in catalog
5. **Capability conflicts** — provides_capabilities don't conflict with existing
6. **Permission reasonableness** — requested permissions are grantable
7. **Dependency satisfaction** — required_capabilities are provided (or declared optional)
8. **UI declaration consistency** — interactive_view plugins have launch_command

Invalid manifests cause registration rejection with specific error codes (see Error Taxonomy spec).

---

# SPEC 03: Pipeline Templates

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

---

# SPEC 04: Handshake Protocols

## 4.1 Purpose

Define the exact communication protocols between plugins and core, and (when needed) between plugins via the core.

## 4.2 Registration Handshake (4-step)

```
[Plugin Start]
    ↓
1. Plugin reads its own plugin.yaml
   Plugin connects to core via Unix socket / gRPC
   Plugin sends: REGISTER_REQUEST {manifest, pid, version}
    ↓
2. Core validates manifest (schema, version compat, conflicts)
   Core assigns plugin_id_runtime and credentials
   Core responds: REGISTER_ACK {plugin_id_runtime, event_bus_token, config_bundle, peer_capabilities_snapshot}
    ↓
3. Plugin receives config, connects to NATS using token
   Plugin subscribes to its declared topics
   Plugin sends: READY {subscribed_topics, health_endpoint}
    ↓
4. Core marks plugin as HEALTHY_ACTIVE
   Core recomputes dispatch tables including new plugin
   Core broadcasts: plugin.joined event
    ↓
[Plugin is now part of system]
```

Timeouts:
- Step 1 → 2: 5 seconds (core must validate quickly)
- Step 2 → 3: 10 seconds (plugin must connect to event bus)
- Step 3 → 4: 2 seconds (core marks active)

If any step times out, plugin is rejected and the core logs specific failure reason.

## 4.3 Graceful Shutdown Handshake

```
[Shutdown triggered by user or core]
    ↓
1. Core sends: SHUTDOWN_REQUEST {grace_period_seconds}
2. Plugin stops accepting new dispatches
   Plugin completes in-flight work
   Plugin persists state to disk
   Plugin sends: SHUTDOWN_READY
3. Core removes plugin from dispatch tables
   Core recomputes routing
   Core sends: SHUTDOWN_CONFIRMED
4. Plugin exits cleanly
   Core broadcasts: plugin.left event
```

Grace period default: 30 seconds. If exceeded, SIGTERM, then SIGKILL after another 10.

## 4.4 Single Handshake — Pipeline Dispatch

Used for normal pipeline flow. One-way with acknowledgments.

```
Core → Plugin:    DISPATCH {msg_id, envelope, deadline_ms, slot}
Plugin → Core:    ACK {msg_id, received_at, estimated_processing_ms}
                  [plugin processes]
Plugin → Core:    COMPLETE {msg_id, result_envelope, processing_duration}
   OR
Plugin → Core:    FAILED {msg_id, error_code, retry_safe: true/false}
   OR
Core notices:     TIMEOUT (no COMPLETE within deadline)
```

States tracked per in-flight message:
- AWAITING_ACK (0–500ms window)
- PROCESSING (ACK received)
- COMPLETE | FAILED | TIMEOUT

ACK requirement: <500ms. If no ACK, plugin is presumed unresponsive.

## 4.5 Double Handshake — Plugin Capability Query

Used when Plugin A needs to invoke Plugin B's capability.

```
Plugin A → Core:  QUERY {target_capability, params, await_response: true, correlation_id}
Core → Plugin B:  QUERY_DISPATCH {query_id, requester: A, params, correlation_id}
Plugin B → Core:  ACK {query_id}
Core → Plugin A:  ACK_FORWARDED {query_id}
                  [Plugin B processes]
Plugin B → Core:  RESPONSE {query_id, result}
Core → Plugin A:  RESPONSE_DELIVERED {query_id, result, correlation_id}
Plugin A → Core:  RECEIPT_ACK {query_id}
```

Four acknowledgments — two on each side — hence "double."

Use cases:
- Cognitive Core querying Memory
- World Model querying Persona
- Any plugin needing another's declared capability

## 4.6 Emergency Bypass (Exception)

Limited set of capabilities can bypass normal routing for genuine emergencies:

```
Plugin → Core:    EMERGENCY_BYPASS {bypass_type, payload}
Core:             Validates bypass authorization (plugin must have permission in manifest)
Core → Target:    EMERGENCY_DISPATCH (to Queue Zone emergency handler)
```

Only certain plugins (audio safety-sound detection, System Health critical alerts, emergency input from Creator) can invoke emergency bypass. Declared in manifest:

```yaml
emergency_bypass:
  - bypass_type: "safety_sound_detected"
  - bypass_type: "health_critical"
```

Core validates at registration that requested bypass types are allowed for this plugin type.

## 4.7 Heartbeat Protocol

Bidirectional heartbeats every N seconds (configurable per plugin, default 10):

```
Plugin → Core:    HEARTBEAT {plugin_id, timestamp, metrics, status}
Core → Plugin:    HEARTBEAT_ACK {server_time}
```

If core misses 3 consecutive heartbeats from a plugin → mark UNRESPONSIVE
If plugin misses 3 consecutive ACKs → attempt reconnection

## 4.8 Protocol Details

All protocols use:
- **Transport:** gRPC over Unix socket (low latency, local-only)
- **Serialization:** Protocol Buffers for type safety
- **Authentication:** Token-based, issued during registration
- **Versioning:** Protocol version in every message; mismatch triggers compatibility check

---

# SPEC 05: Capability Registry

## 5.1 Purpose

The capability registry is the core's live index of everything every plugin can do. It enables:
- Plugin-to-plugin discovery (double handshake queries)
- LLM tool exposure (Tool Bridge integration)
- Graceful degradation (checking if capability available before needing it)

## 5.2 Registry Structure

```
Capability Registry (in-memory, maintained by core):
├── By capability_id
│   ├── memory.retrieve_episodes
│   │   ├── providing_plugins: [memory]
│   │   ├── status: available
│   │   ├── schema: {params, response}
│   │   ├── latency_class: fast
│   │   └── llm_exposed_to: [cognitive_core, world_model]
│   ├── eal.get_environment_summary
│   │   └── ...
│   └── ...
├── By plugin_id
│   ├── memory
│   │   ├── provides: [memory.retrieve_episodes, memory.store_episode, ...]
│   │   └── requires: [inference.complete]
│   └── ...
└── By llm_tool_exposure
    ├── cognitive_core: [list of capabilities to expose in prompt]
    ├── world_model: [...]
    └── ...
```

## 5.3 Registry API

Core exposes these operations to plugins via control plane:

```
query_capability_available(capability_id) → boolean
list_capabilities_for_llm(requesting_plugin_id) → array of tool schemas
find_providers(capability_id) → array of plugin_ids
get_capability_schema(capability_id) → schema object
subscribe_to_capability_changes(capability_ids, callback)
```

## 5.4 Registry Lifecycle

- **On plugin registration:** Core reads provides_capabilities, adds entries
- **On plugin shutdown:** Core marks capabilities as `unavailable`
- **On plugin crash:** Capabilities marked `unavailable` immediately; restored when plugin healthy again
- **On registry change:** Core broadcasts `capability.changed` event; subscribers can react

## 5.5 Capability Namespacing

Capability IDs follow the convention: `<plugin_namespace>.<capability_name>`

Examples:
- `memory.retrieve_episodes`
- `memory.store_episode`
- `eal.get_environment_summary`
- `persona.get_current_emotional_state`
- `world_model.review_proposed_action`
- `inference.complete`

This prevents conflicts and makes ownership clear.

## 5.6 Conflicts

If two plugins declare the same capability_id:
- Registration rejects the second
- Error code: `CAPABILITY_CONFLICT`
- Solution: plugins must namespace properly

Exception: redundancy/failover (future feature) — capabilities can declare themselves as alternatives.

---

# SPEC 06: State Broadcast

## 6.1 Purpose

The State Broadcast channel carries semantic state information between plugins — distinct from health pulses (technical) and pipeline messages (data flow).

## 6.2 Broadcast Channel

- **Transport:** NATS pub/sub
- **Topic naming:** `state.<plugin_id>.<state_name>`
- **Payload format:** JSON with timestamp, previous value, new value, source

## 6.3 Example State Topics

| Topic | Publisher | Subscribers |
|---|---|---|
| `state.cognitive_core.activity_state` | Cognitive Core | Queue Zone, EAL, Health |
| `state.persona.emotional_state` | Persona | Cognitive Core (via context), dashboard |
| `state.system.stage` | Core daemon | All plugins (for sleep behavior) |
| `state.eal.monitoring_mode` | EAL | Thalamus, dashboard |
| `state.memory.consolidation_status` | Memory | Cognitive Core, Sleep/Dream |

## 6.4 State Value Types

Plugins declare in manifest what state values they publish:

```yaml
state_broadcasts:
  - state_name: activity_state
    values: [idle, reasoning, daydreaming, sleeping, emergency_wake]
    change_topic: state.cognitive_core.activity_state
```

## 6.5 Subscription Pattern

Plugins subscribe via SDK:

```python
@subscribe_state("cognitive_core.activity_state")
async def handle_cognitive_state_change(old_value, new_value, timestamp):
    if new_value == "reasoning":
        # Queue Zone adjusts injection thresholds
        ...
```

## 6.6 Differences From Health Pulses

| Aspect | Health Pulse | State Broadcast |
|---|---|---|
| Purpose | Technical health of plugin | Semantic state of being |
| Frequency | Every N seconds constantly | On change only |
| Audience | System Health, dashboard | Subscribers based on need |
| Content | Metrics, error counts | What plugin is doing semantically |
| Volume | High | Low |

---

# SPEC 07: Error Taxonomy

## 7.1 Purpose

Standardized error codes enable consistent handling across the framework.

## 7.2 Error Code Format

`KGN-<category>-<specific>-<severity>`

Example: `KGN-PIPELINE-TIMEOUT-ERROR`, `KGN-MANIFEST-INVALID_SCHEMA-FATAL`

## 7.3 Categories

### 7.3.1 MANIFEST errors

| Code | Description | Severity |
|---|---|---|
| `KGN-MANIFEST-INVALID_SCHEMA-FATAL` | Manifest fails schema validation | FATAL |
| `KGN-MANIFEST-VERSION_MISMATCH-FATAL` | SDK version mismatch | FATAL |
| `KGN-MANIFEST-DUPLICATE_ID-FATAL` | Plugin ID already registered | FATAL |
| `KGN-MANIFEST-CAPABILITY_CONFLICT-FATAL` | Capability ID conflicts | FATAL |
| `KGN-MANIFEST-SLOT_NOT_FOUND-ERROR` | Registered for nonexistent slot | ERROR |
| `KGN-MANIFEST-PIPELINE_NOT_FOUND-ERROR` | Registered for nonexistent pipeline | ERROR |

### 7.3.2 LIFECYCLE errors

| Code | Description | Severity |
|---|---|---|
| `KGN-LIFECYCLE-REGISTRATION_TIMEOUT-ERROR` | Plugin didn't complete registration in time | ERROR |
| `KGN-LIFECYCLE-UNRESPONSIVE-ERROR` | Plugin missed 3 heartbeats | ERROR |
| `KGN-LIFECYCLE-STARTUP_FAILED-ERROR` | Plugin process couldn't start | ERROR |
| `KGN-LIFECYCLE-SHUTDOWN_TIMEOUT-WARNING` | Plugin didn't confirm shutdown in time | WARNING |
| `KGN-LIFECYCLE-MAX_RESTARTS_EXCEEDED-CRITICAL` | Plugin crashed repeatedly | CRITICAL |

### 7.3.3 PIPELINE errors

| Code | Description | Severity |
|---|---|---|
| `KGN-PIPELINE-TIMEOUT-ERROR` | Slot processing exceeded timeout | ERROR |
| `KGN-PIPELINE-LOOP_DETECTED-ERROR` | Envelope hop_count exceeded max | ERROR |
| `KGN-PIPELINE-NO_HANDLER-ERROR` | Required slot has no registered plugin | ERROR |
| `KGN-PIPELINE-INVALID_ENTRY-ERROR` | Envelope tried to enter at non-entry-point slot | ERROR |
| `KGN-PIPELINE-REVISION_EXHAUSTED-ERROR` | Action review revisions exceeded 3 | ERROR |
| `KGN-PIPELINE-DEAD_LETTER-WARNING` | Message moved to dead-letter queue | WARNING |

### 7.3.4 CAPABILITY errors

| Code | Description | Severity |
|---|---|---|
| `KGN-CAPABILITY-NOT_FOUND-ERROR` | Requested capability does not exist | ERROR |
| `KGN-CAPABILITY-UNAVAILABLE-ERROR` | Provider is down | ERROR |
| `KGN-CAPABILITY-UNAUTHORIZED-ERROR` | Caller not permitted | ERROR |
| `KGN-CAPABILITY-INVALID_PARAMS-ERROR` | Params fail schema validation | ERROR |
| `KGN-CAPABILITY-TIMEOUT-ERROR` | Provider didn't respond in time | ERROR |

### 7.3.5 CONTEXT errors

| Code | Description | Severity |
|---|---|---|
| `KGN-CONTEXT-BUDGET_EXCEEDED-WARNING` | Context trim triggered | WARNING |
| `KGN-CONTEXT-TRIM_FAILED-ERROR` | Cannot fit even after trimming | ERROR |

### 7.3.6 PERMISSION errors

| Code | Description | Severity |
|---|---|---|
| `KGN-PERMISSION-DENIED-ERROR` | Plugin attempted unauthorized action | ERROR |
| `KGN-PERMISSION-SANDBOX_ESCAPE-CRITICAL` | Plugin tried to escape sandbox | CRITICAL |

### 7.3.7 CONSTITUTION errors

| Code | Description | Severity |
|---|---|---|
| `KGN-CONSTITUTION-VIOLATION_ATTEMPTED-CRITICAL` | Automated process tried to modify Constitutional Core | CRITICAL |
| `KGN-CONSTITUTION-UNAUTHORIZED_CHANGE-CRITICAL` | Change attempted without proper auth | CRITICAL |

## 7.4 Error Propagation

Errors flow:
1. Plugin detects → logs locally
2. Plugin reports via control plane → core logs
3. If ERROR+ severity → Health System notified
4. If CRITICAL → Immediate escalation to creator

All errors include:
- `error_code`
- `plugin_id` (if plugin-originated)
- `timestamp`
- `trace_id` (if pipeline-related)
- `message` (human-readable)
- `context` (relevant state)

---

# SPEC 08: Plugin Lifecycle

## 8.1 Lifecycle States

```
UNREGISTERED → REGISTERED → STARTING → HEALTHY_ACTIVE
                                ↓
                          UNHEALTHY ← → HEALTHY_ACTIVE
                                ↓
                          UNRESPONSIVE
                                ↓
                     (restart attempts)
                                ↓
                        CIRCUIT_OPEN (cooldown)
                                ↓
                      DEAD (max restarts exceeded)

Graceful path:
HEALTHY_ACTIVE → SHUTTING_DOWN → SHUT_DOWN
```

## 8.2 State Transitions

| From | To | Trigger |
|---|---|---|
| UNREGISTERED | REGISTERED | Manifest validated |
| REGISTERED | STARTING | Process spawning |
| STARTING | HEALTHY_ACTIVE | First heartbeat + READY received |
| HEALTHY_ACTIVE | UNHEALTHY | Degraded metrics but still responding |
| HEALTHY_ACTIVE | UNRESPONSIVE | 3 missed heartbeats |
| UNRESPONSIVE | STARTING | Restart initiated |
| UNRESPONSIVE | CIRCUIT_OPEN | Multiple restart failures |
| CIRCUIT_OPEN | STARTING | Cooldown elapsed, trying again |
| CIRCUIT_OPEN | DEAD | Max attempts exceeded |
| HEALTHY_ACTIVE | SHUTTING_DOWN | Shutdown requested |
| SHUTTING_DOWN | SHUT_DOWN | Shutdown confirmed |

## 8.3 State and Dispatch

- Only HEALTHY_ACTIVE plugins receive dispatches
- STARTING plugins: dispatches queue until ready (brief window only, hard timeout)
- UNHEALTHY plugins: still receive dispatches but with degraded expectations
- UNRESPONSIVE+: no dispatches, capabilities marked unavailable

## 8.4 Backoff Schedule for Restarts

Attempt 1: immediate
Attempt 2: 30 seconds
Attempt 3: 2 minutes
Attempt 4: 5 minutes
Attempt 5: 15 minutes
After 5 failures: CIRCUIT_OPEN with 1-hour cooldown
After 3 circuit-open cycles: DEAD, creator notified

## 8.5 State Visibility

- All plugin states visible in core's Plugin Registry
- Dashboard shows states as status indicators
- State transitions logged for diagnosis
- Health System uses state changes as input for Layer 2 (innate response)

---

# SPEC 09: Mutation Semantics

## 9.1 Purpose

Defines how state changes happen across the three classes of state: Constitutional (immutable), Developmental (slow), Dynamic (continuous).

## 9.2 Constitutional Changes

**Rules:**
- NEVER happen automatically
- Only via explicit creator command through authenticated channel
- Require explicit confirmation protocol
- Logged permanently (audit trail)
- Offspring System CANNOT modify

**Protocol:**
1. Creator issues change via System GUI
2. System shows diff
3. Creator must explicitly confirm
4. Cryptographic signature of change
5. Logged to `~/.kognis/audit/constitutional_changes.log`
6. Change takes effect on next plugin restart

## 9.3 Developmental Identity Changes

**Rules:**
- Happen during sleep cycles only
- Batched, never real-time
- Require evidence threshold (trait candidates need 70%+ consistency over 2+ weeks)
- Drift detection compares to 30/60/90 day baselines
- Creator informed of significant changes

**Protocol:**
1. Sleep/Dream plugin's Trait Discovery analyzes behavioral patterns
2. If pattern meets threshold, proposes trait candidate
3. Cognitive Core confirms/rejects on next wake
4. If confirmed, Persona plugin adds to Developmental Identity
5. Versioned — old state retained
6. Broadcast change on state channel

## 9.4 Dynamic State Changes

**Rules:**
- Real-time, continuous
- Eventually consistent (brief windows of stale reads acceptable)
- Small deltas accumulate
- Decay toward baseline

**Protocol:**
1. Any plugin reports event via `persona.update_dynamic_state` capability
2. Persona applies delta to in-memory state
3. Broadcasts change (if significant)
4. Persists to disk (durability)
5. Decay job runs every hour (or periodically)

## 9.5 Memory Changes

**Rules:**
- Real-time capture through gatekeeper
- Deduplication and importance filtering (no LLM)
- Consolidation during sleep
- Never silent loss — gatekeeper decisions logged

**Protocol:**
1. Cognitive Core reflection produces memory candidate
2. `memory.store_candidate` capability called
3. Memory gatekeeper: check importance, dedupe, contradictions
4. If accepted: write to SQLite+ChromaDB atomically
5. If contradiction: flag for sleep-time resolution
6. If reinforcement: update existing rather than duplicate

## 9.6 World Model Experiential Changes

**Rules:**
- Calibration happens during sleep only
- Small deltas per cycle
- Baseline constitution never changed
- Journal tracks evidence

**Protocol:**
1. During active operation, World Model Journal logs review outcomes
2. During sleep, Sleep/Dream's World Model Calibration job analyzes
3. Computes small delta (e.g., -0.02 on social_consequence_threshold)
4. Applies via `world_model.apply_calibration_delta`
5. Logs change with reasoning

---

# SPEC 10: Context Budget Manager

## 10.1 Purpose

Every LLM-using plugin needs disciplined context management. The Context Budget Manager is a required SDK component that prevents context overflow and ensures critical information always makes it into prompts.

## 10.2 Context Block Priority Tiers

| Tier | Examples |
|---|---|
| MUST | Identity block, Instruction block, Current input envelope |
| HIGH | Sidebar critical items, Recent memories (last 24h), Emotional state |
| MEDIUM | State block, Environmental summary, Older memories (days-weeks) |
| LOW | Background context, Historical memories (months+), Meta-information |

## 10.3 Budget Algorithm

Before every LLM call:

```
1. Get target model's context window from Inference Gateway capability
2. Reserve output budget (default: 4000 tokens, configurable)
3. Reserve safety margin (default: 500 tokens)
4. Available input budget = window - output - margin
5. Assemble context blocks with priority
6. If total > budget:
   a. Trim LOW tier first (summarize via cheap model OR drop with note)
   b. Then MEDIUM tier
   c. Never trim MUST or HIGH
   d. If MUST+HIGH > budget: raise error (KGN-CONTEXT-TRIM_FAILED)
7. Make LLM call with trimmed context
8. Log trim action if triggered (for adaptive feedback)
```

## 10.4 Adaptive Feedback

When trimming happens frequently, system adjusts:
- Memory plugin reduces default retrieval size
- Environmental summary shortens
- Cognitive Core may compact working memory

This prevents constant trimming — system self-tunes.

## 10.5 Long Session Compaction

For stateful agents in long reasoning sessions (e.g., daydream over 30 minutes):
- Periodically compact own working memory
- Take reasoning-so-far → summarize to half size → continue from summary
- Mirrors human memory fade during sustained thought

---

# SPEC 11: Tool Bridge

## 11.1 Purpose

Translates between the framework's internal capability system and LLM-facing tool-call protocols.

Two distinct layers that must not be conflated:
- **Layer 1:** Plugin-to-plugin capability queries (internal plumbing, no LLM involvement)
- **Layer 2:** LLM tool calls (model decides to use a tool based on prompt-exposed schema)

Tool Bridge is the translation layer that exists inside every LLM-using plugin.

## 11.2 Architecture

```
Inside a stateful agent plugin (e.g., Cognitive Core):

  ┌─────────────────────────────────────────┐
  │ Capability Registry Client              │
  │ (via core control plane)                 │
  └────────────────┬────────────────────────┘
                    │
                    ↓
  ┌─────────────────────────────────────────┐
  │ Tool Bridge                              │
  │   ↓ Prompt Assembly                      │
  │ Translate registry entries → OpenAI/     │
  │ Anthropic tool-call schema               │
  │   ↑ Tool Use Handling                    │
  │ Translate LLM tool_use blocks → capability│
  │ queries → execute → translate result     │
  │ back as tool_result                      │
  └────────────────┬────────────────────────┘
                    │
                    ↓
  ┌─────────────────────────────────────────┐
  │ LLM Interface                            │
  │ (talks to Inference Gateway plugin)      │
  └─────────────────────────────────────────┘
```

## 11.3 Prompt-Time Tool Assembly

Before each LLM call:

```python
# Pseudo-code
available_tools = []
for capability in capability_registry.list_for_llm("cognitive_core"):
    tool_schema = {
        "name": capability.id,
        "description": capability.llm_tool_description,
        "parameters": capability.params_schema
    }
    available_tools.append(tool_schema)

prompt_with_tools = assemble_prompt(context_blocks, available_tools)
response = inference_gateway.complete(prompt_with_tools)
```

## 11.4 Tool Use Handling

When LLM emits tool_use:

```python
for tool_use_block in response.tool_uses:
    capability_id = tool_use_block.name
    params = tool_use_block.params
    
    # Double-handshake capability query
    result = await capability_registry.query(
        target=capability_id,
        params=params,
        await_response=True
    )
    
    # Return result to LLM as tool_result
    tool_results.append({
        "tool_use_id": tool_use_block.id,
        "content": result
    })
```

## 11.5 Security Boundaries

- `llm_tool_expose_to` in manifest controls which plugins' LLMs see which capabilities
- Not all capabilities are LLM-exposed
- Capabilities marked `authentication_required: true` never exposed to LLM
- LLM cannot invoke capabilities outside its exposure list (enforced at registry query time)

## 11.6 New Tool Auto-Discovery

When a new plugin registers providing LLM-exposed capabilities:
- Capability Registry broadcasts change
- Tool Bridges in active plugins update their cache
- Next LLM call includes new tool
- LLM can start using it immediately (with appropriate description)

This is the mechanism that enables "plug in robotic arms, the being uses them" — described in the Foundation document.

---

# SPEC 12: Durability & Backup

## 12.1 Purpose

State loss is unacceptable. Every stateful plugin must persist state durably and support restart without loss.

## 12.2 Durability Requirements

Each stateful plugin must:
- Write every state change to disk synchronously (fsync) before acknowledging
- Support crash recovery from disk state on restart
- Maintain backup chain for rollback

## 12.3 Three-Layer Backup Chain

| Layer | Frequency | Storage | Purpose |
|---|---|---|---|
| **Layer 1 — Synchronous writes** | Every state change | `~/.kognis/<plugin>/state/` | Crash recovery |
| **Layer 2 — Periodic snapshots** | Every 30 minutes | `~/.kognis/backup/<plugin>_<timestamp>.tar.gz` | Recent corruption recovery |
| **Layer 3 — Daily external** | Once per day | Configurable external (NAS, cloud) | Disaster recovery |

## 12.4 Critical Plugins

Memory, Persona, World Model, Sleep/Dream must follow full 3-layer backup.
Stateless plugins may have lighter requirements (no ongoing state).

## 12.5 Restore Protocol

On plugin startup:
1. Read from Layer 1 (primary state)
2. If corrupt/missing, restore from most recent Layer 2
3. If Layer 2 also gone, restore from Layer 3
4. If no valid backup, raise CRITICAL alert — do not silently start with empty state
5. Creator notified of any restore beyond Layer 1

## 12.6 Backup Management

- Old Layer 2 snapshots pruned after 7 days (configurable)
- Old Layer 3 backups pruned after 30 days (configurable)
- Critical events (major milestones, identity changes) create permanent backups

---

# SPEC 13: Startup Dependency Order

## 13.1 Purpose

Plugins depend on each other. The core daemon must start them in correct order.

## 13.2 Dependency Declaration

In manifest:

```yaml
startup_dependencies:
  hard: [inference-gateway]  # Must be HEALTHY before this plugin starts
  soft: [memory, persona]     # Prefer HEALTHY but can start in parallel
```

## 13.3 Critical Startup Order

Based on the design, approximate order:

1. **Core daemon** (Go) — the orchestrator itself
2. **Inference Gateway** — LLM infrastructure
3. **Memory** — needed by most cognitive operations
4. **Persona Manager** — identity provider
5. **World Model** — review layer
6. **Thalamus** — input gateway
7. **EAL** — environmental awareness
8. **Prajna** — cognitive pipeline (Cognitive Core inside)
9. **Brainstem** — output gateway
10. **Communication plugins** (Chat TUI, Telegram, Voice)
11. **Sleep/Dream** — can start later
12. **System Health** — can start later
13. **Offspring** — starts last, only during Adolescence+

## 13.4 Topological Resolution

Core does:
1. Read all manifests, extract dependencies
2. Topologically sort
3. Detect cycles — if any, refuse to start, alert creator
4. Start in order
5. Wait for each HEALTHY before starting next wave
6. Timeout per plugin: 60 seconds default

## 13.5 Partial Startup

If a plugin fails to start:
- If hard dependency of others: others wait
- If soft dependency: others start anyway
- System continues with degraded capability
- Creator notified

---

# SPEC 14: Emergency Bypass

## 14.1 Purpose

A strictly-limited mechanism for genuine emergencies that cannot wait for normal pipeline processing.

## 14.2 Authorized Bypass Types

| Bypass Type | Who Can Invoke | What It Does |
|---|---|---|
| `safety_sound_detected` | Audio monitoring plugin | Fire alarm, smoke alarm, breaking glass, crash sounds |
| `health_critical` | System Health plugin | System is failing, immediate escalation needed |
| `creator_emergency` | Chat/Telegram plugins | Creator explicitly flagged emergency |
| `physical_hazard` | Hardware monitoring plugin | Overheat, power imbalance, etc. |

## 14.3 Bypass Protocol

```
Plugin → Core:    EMERGENCY_BYPASS {bypass_type, payload}
                   ↓
Core validates:    Is plugin authorized for this bypass_type?
                   ↓
Core dispatches:   Direct to Queue Zone emergency handler
                   Queue Zone treats as Tier 1 CRITICAL
                   Sends EMERGENCY_WAKE if system in deep sleep
                   Triggers Cognitive Core immediate attention
```

## 14.4 Authorization

- Each plugin must declare allowed bypass types in manifest
- Core validates at registration — unauthorized declarations rejected
- Registry of authorized bypassers is fixed (not dynamically expanded)
- Abuse of bypass mechanism logged and may trigger plugin isolation

## 14.5 Why This Exists

The "normal" route through Thalamus has sleep-mode filtering during Stage 3 deep consolidation. A fire alarm cannot wait for that filter. Emergency bypass ensures genuine emergencies reach cognitive attention immediately.

This is the ONLY documented exception to "plugins do not communicate directly." The exception exists for safety, not convenience.

---

# SPEC 15: Emotional State Vector

## 15.1 Structure

```yaml
emotional_state:
  valence: float       # -1.0 to +1.0
  arousal: float       # 0.0 to 1.0
  engagement: float    # 0.0 to 1.0
  confidence: float    # 0.0 to 1.0
  warmth: float        # 0.0 to 1.0
  
  last_updated: timestamp
  baseline:
    valence: float     # Personal baseline (develops over time)
    arousal: float
    engagement: float
    confidence: float
    warmth: float
```

## 15.2 Delta Operations

Events produce deltas:
```python
# From Cognitive Core
await persona_plugin.update_dynamic_state({
    "deltas": {
        "valence": 0.05,
        "engagement": 0.02
    },
    "source": "successful_task_completion",
    "timestamp": now()
})
```

Persona applies delta, clamps to [-1, 1] or [0, 1] as appropriate.

## 15.3 Decay Algorithm

Every hour (configurable):
```python
for dimension in state:
    current = state[dimension]
    baseline_value = baseline[dimension]
    drift = baseline_value - current
    state[dimension] = current + (drift * 0.10)  # 10% toward baseline per hour
```

## 15.4 Baseline Drift

Baselines evolve slowly — weekly computation during sleep:
```python
if weekly_cycle:
    for dimension in state:
        # Compute median state over past week
        weekly_median = compute_median(state_history[dimension], window_days=7)
        # Baseline drifts toward median very slowly
        baseline[dimension] = baseline[dimension] + (weekly_median - baseline[dimension]) * 0.05
```

Over months, baseline reflects the being's actual developed emotional tendency.

## 15.5 Prompt Integration

Cognitive Core includes in identity block:
```
Current emotional state:
  Valence: 0.72 (pleasant)
  Arousal: 0.44 (moderate energy)
  Engagement: 0.89 (highly invested)
  Confidence: 0.61 (fairly certain)
  Warmth: 0.81 (very warm)
  
Baseline (who I tend to be):
  Valence: 0.45 (gently positive)
  Arousal: 0.35 (calm)
  Engagement: 0.65 (engaged but measured)
  Confidence: 0.55 (balanced)
  Warmth: 0.70 (warm)
```

LLM modulates output accordingly.

## 15.6 Memory Attachment

Every memory created tags current emotional state:
```yaml
memory_entry:
  content: "..."
  timestamp: "..."
  emotional_context:
    valence: 0.72
    arousal: 0.44
    # ... etc
```

Enables emotionally-similar retrieval: "find memories formed when I was similarly feeling X."

---

# SPEC 16: Sleep Stage Behaviors

## 16.1 Stage Definitions

Four stages with adaptive duration:

| Stage | Duration | Interruptibility |
|---|---|---|
| Stage 1 — Settling | 30-60 min | HIGH (any Tier 1/2 wakes) |
| Stage 2 — Maintenance | 1-2 hrs | MEDIUM (Tier 1 wakes, maintenance pauses) |
| Stage 3 — Deep Consolidation | 3-6 hrs | LOW (sleepwalking mode for non-critical) |
| Stage 4 — Pre-Wake | 30-60 min | HIGH |

Total sleep duration: 6-12 hours, adaptive.

## 16.2 Plugin Sleep Behavior Declaration

Each plugin declares in manifest:

```yaml
sleep_behavior:
  stage_1_settling: continue_normal
  stage_2_maintenance: reduced_activity
  stage_3_deep_consolidation: monitoring_only
  stage_4_pre_wake: continue_normal
  
  maintenance_jobs:
    - name: database_optimize
      estimated_duration_seconds: 300
    - name: log_rotation
      estimated_duration_seconds: 60
```

## 16.3 Stage Transition Broadcasts

Core broadcasts system stage:

```
state.system.stage = "sleep_stage_3"
```

Plugins react per declared behavior.

## 16.4 Emergency Wake

On Tier 1 CRITICAL input during Stage 3:
1. Sleep/Dream checkpoints current consolidation job (2 seconds)
2. Stage 4 runs COMPRESSED (2-3 min)
3. Cognitive Core wakes with reduced context handoff
4. Emergency processed
5. If time remains in planned sleep window: re-enter Stage 3 from checkpoint
6. If not: stay awake, note sleep debt

## 16.5 Sleep Debt Tracking

System tracks incomplete consolidation:
```yaml
sleep_debt:
  episodes_unprocessed: int
  contradictions_unresolved: int
  skills_unrefined: int
  last_clean_sleep: timestamp
```

If debt accumulates, Sleep/Dream requests extended rest from creator.

## 16.6 Adaptive Duration Algorithm

Before sleep:
```python
estimated_work_time = (
    episodes_count * avg_consolidation_time +
    contradictions_count * avg_resolution_time +
    skills_with_new_data * avg_refinement_time +
    offspring_pending_tests * estimated_test_time +
    base_maintenance_time
)

planned_duration = min(
    max(
        estimated_work_time + STAGE_1_TIME + STAGE_4_TIME,
        MIN_SLEEP_HOURS * 3600
    ),
    MAX_SLEEP_HOURS * 3600
)
```

---

# SPEC 17: Offspring System

## 17.1 Components

1. **Improvement Identifier** — collects signals, prioritizes tickets
2. **Spawning Engine** — creates git branches, delegates code modification
3. **Testing Sandbox** — isolated execution environment
4. **Ancestry Tree Manager** — tracks evolutionary history
5. **Harness Integration** — delegates execution to OpenCode/Claude Code

## 17.2 Ancestry Tree Structure

Stored in `[name]-offspring` repo as `ancestry/tree.json`:

```json
{
  "ancestry_version": 1,
  "main_generation": 5,
  "branches": {
    "gen-5-daydream-novelty-threshold": {
      "parent_commit": "abc123",
      "created_at": "2026-04-21T07:00:00Z",
      "status": "PROMOTED",
      "target_improvement": "Daydream novelty threshold 0.75 → 0.65",
      "test_results": {
        "replay_improvement": "+22% novelty acceptance",
        "regression_check": "none detected"
      },
      "promoted_at": "2026-04-22T09:30:00Z"
    }
  }
}
```

## 17.3 One Improvement Per Branch

Critical discipline: each offspring branch modifies ONE thing. Enables clean attribution and rollback.

## 17.4 3-5 Generation Buffer

`main` branch runs 3-5 generations behind the experimental edge. A change must survive 3+ descendant generations before promoting to main.

## 17.5 Status Lifecycle

```
SPAWNED → TESTING → VALIDATED → PROMOTED
                  → ABANDONED
PROMOTED → ROLLED_BACK (if problems detected later)
```

## 17.6 Safety Boundaries

- Cannot modify Constitutional Core
- Cannot modify World Model baseline constitution
- Cannot modify manifest_version schema or protocol definitions
- Can modify thresholds, prompts, pipeline parameters, skill procedures
- Major changes (Cognitive Core reasoning loop) flagged for human review

## 17.7 Test Types

- **Replay Testing** — real inputs from past 24-48h replayed through variant
- **Synthetic Testing** — targeted edge cases
- **Limited Live Testing** — 10-20% of real inputs in parallel (canary pattern)

---

# SPEC 18: Health Pulse Schema

## 18.1 Pulse Format

```yaml
health_pulse:
  plugin_id: string
  timestamp: ISO8601
  status: string     # HEALTHY | DEGRADED | ERROR | CRITICAL | UNRESPONSIVE
  
  metrics:
    # Plugin-specific, defined in manifest
    queue_depth: integer
    processing_latency_p50_ms: number
    processing_latency_p99_ms: number
    error_count_last_60s: integer
    memory_usage_mb: number
    # ...
  
  current_activity: string  # Brief description
  last_dispatch_at: ISO8601 | null
  
  alerts:
    - severity: string  # warning | error | critical
      code: string
      message: string
```

## 18.2 Aggregation

Core aggregates pulses into Health Registry:
- Last N pulses per plugin (default 100)
- Current status
- Metric time-series (short window)
- Derived metrics (uptime %, error rate)

## 18.3 Visibility

Dashboard shows real-time health pulses.
System Health Layer 1 uses registry for anomaly detection.
System Health Layer 3 uses pulse history for root-cause analysis.

---

## Closing Note for the Splitting Agent

When splitting this document into `docs/spec/*.md`:

1. One file per `# SPEC NN:` section
2. Name pattern: `NN-topic-name.md` (leading zeros for sort order)
3. Add cross-references between related specs
4. Preserve content exactly
5. Add header to each file: stability level, version, related specs
6. Mark this master file as `[SPLIT COMPLETE]` when done
7. Commit with message: `docs(spec): split master-spec.md into individual spec files`

All content here is architectural bedrock. Preserve completely.
