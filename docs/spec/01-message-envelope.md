# Message Envelope

> **Stability:** EVOLVING
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [03-pipeline-templates.md](03-pipeline-templates.md), [04-handshake-protocols.md](04-handshake-protocols.md), [07-error-taxonomy.md](07-error-taxonomy.md)

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