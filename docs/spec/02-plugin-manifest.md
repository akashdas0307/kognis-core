# Plugin Manifest

> **Stability:** EVOLVING
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [04-handshake-protocols.md](04-handshake-protocols.md), [05-capability-registry.md](05-capability-registry.md), [08-plugin-lifecycle.md](08-plugin-lifecycle.md)

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