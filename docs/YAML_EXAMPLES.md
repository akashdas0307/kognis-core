# YAML Examples — Kognis Framework

> **Purpose:** Canonical YAML templates referenced by specs and implementations
> **Stability:** EVOLVING

---

## 1. Plugin Manifest (from SPEC 02)

```yaml
# plugin.yaml — Every plugin's root declaration
api_version: 1
plugin_id: "com.kognis.prajna"
name: "Prajna"
version: "0.1.0"
description: "Intelligence Core — input processing, routing, cognitive dispatch"
author: "Kognis Framework"

capabilities:
  - id: "input_processing"
    description: "Normalize and tag incoming inputs"
    input_schema:
      type: object
      properties:
        raw_input: { type: string }
        channel: { type: string }
    output_schema:
      type: object
      properties:
        tagged_input: { type: object }
        priority: { type: string }

  - id: "cognitive_dispatch"
    description: "Route processed inputs to cognitive core"
    input_schema:
      type: object
      properties:
        enriched_input: { type: object }
    output_schema:
      type: object

slots:
  - pipeline: "user_text_interaction"
    slot: "input_reception"
    priority: 10

  - pipeline: "user_text_interaction"
    slot: "input_enrichment"
    priority: 5

permissions:
  - "event_bus:publish"
  - "event_bus:subscribe:user_text_interaction.*"
  - "capability:query:com.kognis.memory.recall"
  - "llm:inference:ollama"

lifecycle:
  startup_order: 2
  health_pulse_interval: 30
  state_broadcast: true
  sleep_behavior: "suspend"

ui:
  has_dashboard: false
  has_settings: false
```

---

## 2. Message Envelope (from SPEC 01)

```yaml
# Example message envelope
envelope_version: 1
id: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
created_at: "2026-04-22T14:30:00Z"
origin_plugin: "com.kognis.thalamus"
message_type: "user_text_input"

payload:
  user_id: "creator"
  text: "Hello, how are you today?"
  channel: "telegram"

routing:
  pipeline: "user_text_interaction"
  completed_stages: ["input_reception"]
  current_stage: "input_enrichment"
  failed_stages: []
  hop_count: 2
  entry_slot: "input_reception"

enrichments:
  environment:
    time_of_day: "afternoon"
    ambient_noise_level: "low"

metadata:
  priority: "tier_2_elevated"
  trust_level: "tier_1_creator"
  trace_id: "trace-abc123"
  revision_count: 0
  parent_envelope_id: null
  correlation_id: null
```

---

## 3. Pipeline Template (from SPEC 03)

```yaml
# Pipeline template — framework-defined processing flow
api_version: 1
pipeline_id: "user_text_interaction"
description: "Standard processing flow for text-based user interactions"
version: "1.0"

slots:
  - name: "input_reception"
    description: "Receive and normalize incoming text input"
    required: true
    max_plugins: 1

  - name: "input_enrichment"
    description: "Add context, memory, environmental data"
    required: true
    max_plugins: 3

  - name: "cognitive_processing"
    description: "Core reasoning and decision-making"
    required: true
    max_plugins: 1

  - name: "action_review"
    description: "Review proposed actions for safety and coherence"
    required: true
    max_plugins: 1

  - name: "action_execution"
    description: "Execute approved actions"
    required: true
    max_plugins: 2

  - name: "output_delivery"
    description: "Format and deliver response to user"
    required: true
    max_plugins: 1

error_handling:
  on_slot_failure: "skip_and_log"
  on_pipeline_failure: "notify_and_queue"
  max_revisions: 3

timeout:
  slot_timeout: 30s
  pipeline_timeout: 120s
```

---

## 4. Health Pulse (from SPEC 18)

```yaml
# Health pulse — technical heartbeat
plugin_id: "com.kognis.prajna"
timestamp: "2026-04-22T14:30:30Z"
pulse_interval: 30

status: "healthy"

metrics:
  cpu_usage: 0.12
  memory_mb: 128
  goroutines: 4
  messages_processed: 142
  errors_since_last: 0
  last_message_at: "2026-04-22T14:30:25Z"

lifecycle_state: "running"
```

---

## 5. State Broadcast (from SPEC 06)

```yaml
# State broadcast — semantic state change
plugin_id: "com.kognis.cognitive_core"
timestamp: "2026-04-22T14:30:00Z"

state: "reasoning"
previous_state: "idle"

details:
  reason: "processing_user_input"
  pipeline: "user_text_interaction"
  estimated_duration: 5s
```

---

## 6. Registry Entry (for kognis-registry)

```yaml
# Registry entry — plugin marketplace metadata
plugin_id: "com.kognis.prajna"
name: "Prajna"
version: "0.1.0"
author: "Kognis Framework"
description: "Intelligence Core plugin"
repository: "https://github.com/kognis-framework/kognis-prajna"
license: "MIT"
verified: true

capabilities:
  - "input_processing"
  - "cognitive_dispatch"

pipelines:
  - "user_text_interaction"
  - "user_voice_interaction"

min_framework_version: "0.1.0"
tags: ["official", "core", "cognitive"]
```

---

*These examples are canonical. Implementations should validate their YAML against these structures and the schemas in `schemas/`.*