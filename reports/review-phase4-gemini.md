# Phase 4 Spec Compliance Review

> **Reviewer:** Gemini (GLM-5.1)
> **Date:** 2026-04-22
> **Scope:** Python SDK Core (M-001 through M-012) vs. referenced SPECs
> **Verdict:** COMPLIANT with minor gaps and 3 valid clarification requests

---

## 1. Module-by-Module SPEC Compliance

### 1.1 envelope.py vs. SPEC 01 (Message Envelope)

**Verdict: COMPLIANT**

| SPEC Requirement | Implementation | Status |
|---|---|---|
| Envelope structure: id, created_at, origin_plugin, message_type, payload, routing, enrichments, metadata | `Envelope` dataclass with all fields present | PASS |
| `envelope_version: 1` | Default `envelope_version: int = 1` | PASS |
| Routing: pipeline, completed_stages, current_stage, failed_stages, hop_count, entry_slot | `RoutingInfo` dataclass with all 6 fields | PASS |
| Metadata: priority, trust_level, trace_id, revision_count, parent_envelope_id, correlation_id | `EnvelopeMetadata` dataclass with all 6 fields | PASS |
| Priority values: tier_1_immediate, tier_2_elevated, tier_3_normal | Validated in `validate_envelope()` | PASS |
| Trust level values: tier_1_creator, tier_2_trusted, tier_3_external, internal | Validated in `validate_envelope()` | PASS |
| Immutability: plugins produce new envelopes, never modify in place | All mutation methods (`with_enrichment`, `with_hop_increment`, `with_completed_stage`, `with_failed_stage`, `with_revision`, `derive`) return NEW Envelope instances | PASS |
| Enrichments additive only, per namespace | `with_enrichment(namespace, data)` adds only to specified namespace | PASS |
| hop_count max 20, loop_detected on exceed | `MAX_HOP_COUNT = 20`, `with_hop_increment()` raises `EnvelopeError("loop_detected")` | PASS |
| revision_count max 3, incremented only by action_review | `MAX_REVISION_COUNT = 3`, `with_revision()` raises on exceed. Note: enforcement of "only action_review increments" is a runtime concern, not SDK-level. | PASS |
| Derived envelopes get new ID, set parent_envelope_id | `derive()` generates new UUID, sets `parent_envelope_id=self.id` | PASS |

**Minor gap:** The spec lists 12 specific message types (user_text_input, voice_input, etc.) and states "this list is extensible." The SDK does not define these as constants or enums, leaving them as free-form strings. This is acceptable for an SDK -- the core daemon validates message types at dispatch time -- but documenting the canonical types as SDK constants would reduce integration errors.

---

### 1.2 manifest.py vs. SPEC 02 (Plugin Manifest)

**Verdict: COMPLIANT with gaps**

| SPEC Requirement | Implementation | Status |
|---|---|---|
| manifest_version: 1 | Validated: `manifest_version != 1` raises error | PASS |
| plugin_id, plugin_name, version, author, license, description | All required fields in Manifest dataclass | PASS |
| language: python, go, node, other | Validated in `validate_manifest()` | PASS |
| handler_mode: stateless, stateful_agent | Validated in `validate_manifest()` | PASS |
| runtime with entrypoint, working_directory, environment_requirements | `RuntimeSpec` with entrypoint, working_directory, python_version, system_packages, external_commands | PASS |
| sdk.required_version, sdk.manifest_schema | Only `sdk_required_version` parsed; `manifest_schema` field MISSING | **GAP** |
| slot_registrations with all 9 fields | `SlotRegistration` has pipeline, slot, priority, message_types_handled/produced, timeout_seconds, retry_attempts, optional, max_concurrent | PASS |
| provides_capabilities with all 8 fields | `CapabilitySpec` has capability_id, description, latency_class, llm_tool_description, llm_tool_expose_to, authentication_required, params_schema, response_schema | PASS |
| requires_capabilities | `RequiredCapability` with capability_id, optional | PASS |
| event_subscriptions, event_publications | `EventSubscription` and `EventPublication` implemented | PASS |
| state_broadcasts | `StateBroadcast` with state_name, values, change_topic, description | PASS |
| health section (pulse_interval_seconds, critical_metrics, alert_conditions) | Only `health_pulse_interval` in LifecycleSpec; no critical_metrics or alert_conditions parsing | **GAP** |
| sleep_behavior (4 stages + maintenance_jobs) | Not parsed; LifecycleSpec has single `sleep_behavior: str = "suspend"` instead of the 4-stage spec | **GAP** |
| permissions (filesystem, network, hardware) | Stored as `list[str]` instead of structured dict with filesystem/network/hardware sub-objects | **GAP** |
| ui section | Not parsed | **GAP** |
| maturity_gate | Not parsed | **GAP** |
| configuration schema/defaults | Not parsed | **GAP** |
| Validation: 8 rules from Section 2.4 | Only 4 of 8 implemented (schema, identity, slot, capability). Missing: pipeline reference validation, capability conflict detection, permission reasonableness, UI consistency | PARTIAL |

**Assessment:** The manifest parser covers the structural core needed for Phase 4 operation (identity, runtime, slots, capabilities, events, state broadcasts). However, it omits several fields that the spec declares as part of the manifest contract: `sdk.manifest_schema`, structured health declarations, the 4-stage sleep behavior model, structured permissions, UI contribution, maturity gate, and configuration schema. These are not currently consumed by any SDK module, so the gap does not cause runtime failures, but it does mean `manifest.from_yaml()` would silently discard valid spec-compliant manifest data. This should be tracked as tech debt for when the Go core sends full manifests to Python plugins.

---

### 1.3 control_plane.py vs. SPEC 04 (Handshake Protocols)

**Verdict: COMPLIANT (stub-level)**

| SPEC Requirement | Implementation | Status |
|---|---|---|
| 4-step registration: REGISTER_REQUEST, REGISTER_ACK, READY, HEALTHY_ACTIVE | `register()` (steps 1-2), `send_ready()` (step 3), state set to HEALTHY_ACTIVE (step 4) | PASS |
| RegisterRequest fields: manifest, pid, version | `RegisterRequest` dataclass with all 3 fields | PASS |
| RegisterAck fields: plugin_id_runtime, event_bus_token, config_bundle, peer_capabilities_snapshot | `RegisterAck` with all 4 fields | PASS |
| Registration timeouts: 5s (ack), 10s (event bus), 2s (ready) | Constants defined: `REGISTRATION_ACK_TIMEOUT=5.0`, `EVENT_BUS_CONNECT_TIMEOUT=10.0`, `READY_CONFIRM_TIMEOUT=2.0` | PASS |
| Graceful shutdown: SHUTDOWN_REQUEST, SHUTTING_DOWN, SHUT_DOWN | `shutdown()` transitions SHUTTING_DOWN then SHUT_DOWN | PASS |
| Grace period default 30s, SIGTERM+10s | `GRACEFUL_SHUTDOWN_DEFAULT=30`, `SIGTERM_ADDITIONAL=10` defined | PASS |
| Dispatch lifecycle: DISPATCH, ACK, COMPLETE/FAILED/TIMEOUT | `DispatchMessage`, `DispatchAck`, `DispatchComplete`, `DispatchFailed` + `DispatchStatus` enum | PASS |
| ACK requirement <500ms | `DISPATCH_ACK_TIMEOUT=0.5` | PASS |
| AWAITING_ACK (0-500ms), PROCESSING, COMPLETE/FAILED/TIMEOUT | `DispatchStatus` enum with all 5 states | PASS |
| Double handshake: QUERY, QUERY_DISPATCH, ACK, ACK_FORWARDED, RESPONSE, RESPONSE_DELIVERED, RECEIPT_ACK | `CapabilityQuery` and `CapabilityResponse` present; 4-step ACK flow not explicitly modeled | **PARTIAL** |
| Heartbeat: bidirectional, 10s default, 3 misses = UNRESPONSIVE | `Heartbeat`, `HeartbeatAck` dataclasses; `_heartbeat_interval=10`, `_max_missed_heartbeats=3` | PASS |
| Emergency bypass (SPEC 04 Section 4.6) | No EmergencyBypass message type or handler | **MISSING** |
| PluginState matches SPEC 08 | All 10 states present in enum | PASS |

**Assessment:** The control plane client correctly models the protocol state machine and message types. The double-handshake query flow is simplified -- the full 7-message exchange (QUERY, QUERY_DISPATCH, ACK, ACK_FORWARDED, RESPONSE, RESPONSE_DELIVERED, RECEIPT_ACK) is compressed into a single `query_capability()` call. This is acceptable for an SDK stub (the Go core will manage the full exchange), but the SDK should document that `query_capability()` is a high-level abstraction over the full double-handshake.

The emergency bypass protocol (SPEC 04 Section 4.6 / SPEC 14) is entirely absent. This is a known omission that the Phase 4 summary does not flag. Emergency bypass is critical for the safety-sound and health-critical pathways.

---

### 1.4 capability.py vs. SPEC 05 (Capability Registry)

**Verdict: COMPLIANT (client-side)**

| SPEC Requirement | Implementation | Status |
|---|---|---|
| Registry structure: by capability_id, by plugin_id, by llm_tool_exposure | `RegistryEntry` has providing_plugins, status, schemas, latency_class, llm_exposed_to | PASS |
| query_capability_available(capability_id) -> boolean | `is_available()` | PASS |
| list_capabilities_for_llm(requesting_plugin_id) -> array of tool schemas | `list_for_llm()` | PASS |
| find_providers(capability_id) -> array of plugin_ids | `find_providers()` | PASS |
| get_capability_schema(capability_id) -> schema object | `get_schema()` returns params + response schemas | PASS |
| subscribe_to_capability_changes | Not implemented (event-based; deferred to event bus integration) | **GAP** |
| Lifecycle: mark unavailable on shutdown/crash | `remove_from_cache()` sets status to "unavailable" | PASS |
| Capability namespacing: `<plugin_namespace>.<capability_name>` | Not enforced in SDK; convention only | PASS (runtime concern) |
| CAPABILITY_CONFLICT on duplicate registration | Not client-side; core concern | N/A |

**Minor gap:** `list_for_llm()` returns `description` as `capability_id` instead of `llm_tool_description` from the CapabilitySpec. The spec's Tool Bridge integration (SPEC 11 Section 11.3) shows `capability.llm_tool_description` as the tool description. The current implementation in `capability.py` line 86 uses `entry.capability_id` as the description, which means the LLM will see the capability ID as the tool description rather than the human-readable description. The `ToolBridge.assemble_tools()` at line 67 does pull `raw.get("description", "")` which would receive whatever `list_for_llm()` returns, so the bug cascades.

---

### 1.5 health.py vs. SPEC 18 (Health Pulse) + SPEC 06 (State Broadcast)

**Verdict: COMPLIANT**

| SPEC 18 Requirement | Implementation | Status |
|---|---|---|
| Pulse format: plugin_id, timestamp, status, metrics, current_activity, last_dispatch_at, alerts | `HealthPulse` dataclass with all fields | PASS |
| Status values: HEALTHY, DEGRADED, ERROR, CRITICAL, UNRESPONSIVE | Constants + `VALID_STATUSES` tuple | PASS |
| Alerts with severity, code, message | `add_alert(severity, code, message)` | PASS |
| Periodic emission | `_emit_loop()` with configurable `interval_seconds` | PASS |
| Wrapped in `health_pulse:` key in serialization | `to_dict()` wraps under `"health_pulse"` key | PASS |

| SPEC 06 Requirement | Implementation | Status |
|---|---|---|
| Transport: NATS pub/sub | Uses `EventBusClient.publish()` | PASS |
| Topic naming: `state.<plugin_id>.<state_name>` | Uses `make_state_topic()` from eventbus | PASS |
| Payload: timestamp, previous value, new value, source | `StateChange` with all 4 fields | PASS |
| On-change only (not periodic) | `broadcast_change()` returns early if old == new (but still returns a StateChange object -- see note) | PASS |
| Subscription pattern: `@subscribe_state` decorator | Not implemented (SDK provides programmatic subscription via event bus, not decorator) | **GAP** |

**Subtle issue in StateBroadcaster:** At line 217-225, when `old_value == new_value`, the method returns a `StateChange` object without publishing it to the event bus. This correctly implements the on-change-only semantics. However, it silently returns a StateChange that was never published, which could confuse callers who expect the returned object to represent a published event. The method should either return `None` or document that the returned StateChange was not published.

---

### 1.6 plugin.py + stateful_agent.py vs. SPEC 08 (Plugin Lifecycle)

**Verdict: MOSTLY COMPLIANT**

| SPEC 08 Requirement | Implementation | Status |
|---|---|---|
| Lifecycle states: UNREGISTERED through SHUT_DOWN | `PluginState` enum with all 10 states | PASS |
| Transition: UNREGISTERED -> REGISTERED -> STARTING -> HEALTHY_ACTIVE | `start()` goes UNREGISTERED -> REGISTERED (in register()) -> HEALTHY_ACTIVE (in send_ready()); STARTING is skipped | **GAP** |
| Transition: HEALTHY_ACTIVE -> SHUTTING_DOWN -> SHUT_DOWN | `stop()` sets SHUTTING_DOWN then SHUT_DOWN | PASS |
| UNHEALTHY, UNRESPONSIVE, CIRCUIT_OPEN, DEAD states | Enum values present but no transition logic in Plugin or StatefulAgent | **GAP** |
| Backoff schedule: 5 attempts then CIRCUIT_OPEN | Not implemented (core-side concern) | N/A (deferred) |
| Only HEALTHY_ACTIVE receives dispatches | `dispatch()` checks `self.state == PluginState.HEALTHY_ACTIVE` | PASS |
| Stateful agents have continuous internal loop | `StatefulAgent.run()` loops `cognition_cycle()` | PASS |
| Stateful agents: working memory, sidebar events, idle behavior | `working_memory` dict present; `on_wake()`, `on_sleep()` hooks; no sidebar event injection mechanism | PARTIAL |

**Foundation 04 consistency:** The foundation document Section 4.4 specifies that StatefulAgent should have `inner_loop()`, `handle_dispatch()`, `handle_sidebar_event()`, `handle_idle()`, and `on_shutdown()`. The SDK provides `cognition_cycle()` (equivalent to inner_loop), `on_dispatch()` (equivalent to handle_dispatch), `on_shutdown()`, and `on_wake()`/`on_sleep()`. Missing: `handle_sidebar_event()` and `handle_idle()`. The `cognition_cycle()` itself serves as the idle handler (it runs continuously), but sidebar event injection during cognition is not modeled.

**Missing STARTING transition:** SPEC 08 defines the path REGISTERED -> STARTING -> HEALTHY_ACTIVE. The current implementation jumps directly from REGISTERED to HEALTHY_ACTIVE in `send_ready()`. The STARTING state represents "process spawning" and is more relevant to the core daemon's view. However, from the plugin's perspective, the SDK should transition through STARTING during the `connect()` -> `register()` phase before `send_ready()`. This is a semantic gap, not a functional one.

---

### 1.7 context_budget.py vs. SPEC 10 (Context Budget Manager)

**Verdict: COMPLIANT with algorithmic concern**

| SPEC 10 Requirement | Implementation | Status |
|---|---|---|
| Priority tiers: MUST, HIGH, MEDIUM, LOW | `PriorityTier` enum with all 4 values | PASS |
| Budget algorithm: reserve output (4000) + margin (500) | `BudgetConfig` defaults match exactly | PASS |
| Available = window - output - margin | `calculate_available_budget()` implements this exactly | PASS |
| Trim LOW first, then MEDIUM | `assemble()` processes medium_blocks THEN low_blocks in the result list, but both are trimmed based on remaining budget | **CONCERN** |
| Never trim MUST or HIGH | Code adds must_blocks and high_blocks unconditionally, raises KGN-CONTEXT-TRIM_FAILED if they exceed budget | PASS |
| Error code KGN-CONTEXT-TRIM_FAILED when MUST+HIGH > budget | Raises `ContextBudgetError("KGN-CONTEXT-TRIM_FAILED", ...)` | PASS |
| Adaptive feedback logging | `_log_trim()` records block name, tier, reason, token count; `frequent_trimming()` checks threshold | PASS |
| Long session compaction | Implemented in `StatefulAgent.compact_working_memory()`, not in ContextBudgetManager itself | PASS (correct separation) |

**Algorithmic concern:** The spec says "Trim LOW tier first (summarize via cheap model OR drop with note), then MEDIUM tier." The implementation at lines 130-144 processes MEDIUM blocks first (adding them to the result before LOW), then processes LOW blocks. This means MEDIUM blocks are preferentially included over LOW blocks, which IS the correct priority order -- MEDIUM should be kept before LOW. However, the spec's wording "trim LOW first, then MEDIUM" means "when you need to trim, drop LOW blocks before dropping MEDIUM blocks." The implementation achieves the correct outcome (MEDIUM is preserved over LOW) but the code structure makes this subtle. The assembly order could be clearer by processing LOW first for inclusion and only including MEDIUM if budget remains after LOW, but the current approach (MEDIUM then LOW) produces the same result because MEDIUM blocks are added greedily first.

Wait -- re-reading more carefully: the spec says when total > budget, "Trim LOW tier first." This means in the trimming phase, LOW blocks should be the first to be dropped. The implementation does this correctly: it adds all MUST and HIGH unconditionally, then tries to fit MEDIUM blocks, then tries to fit LOW blocks with whatever budget remains. LOW blocks are the last to be included and thus the first to be trimmed. **The algorithm is correct.**

---

### 1.8 tool_bridge.py vs. SPEC 11 (Tool Bridge)

**Verdict: COMPLIANT with minor gap**

| SPEC 11 Requirement | Implementation | Status |
|---|---|---|
| Two layers: plugin-to-plugin (internal) and LLM tool calls | Docstring and class comments reference both layers | PASS |
| Prompt-time tool assembly: registry -> tool-call schema | `assemble_tools()` queries `list_for_llm()` and builds `ToolSchema` objects | PASS |
| Tool use handling: tool_use -> capability query -> tool_result | `handle_tool_uses()` iterates tool_use blocks, queries capability_client, returns ToolResult | PASS |
| Security: llm_tool_expose_to filtering | Filtering happens in `CapabilityRegistryClient.list_for_llm()` | PASS |
| authentication_required: never exposed to LLM | NOT enforced -- `list_for_llm()` does not check `authentication_required` | **GAP** |
| Auto-discovery on capability.changed events | `refresh_tools()` available; no automatic event subscription | PARTIAL |

**Security gap:** SPEC 11 Section 11.5 states "Capabilities marked authentication_required: true never exposed to LLM." The `CapabilityRegistryClient.list_for_llm()` at capability.py line 83 does not filter out entries with `authentication_required=True`. The `RegistryEntry` dataclass does not even carry an `authentication_required` field. This is a security boundary violation that must be fixed before the system handles real LLM tool calls.

---

### 1.9 state_store.py vs. SPEC 12 (Durability & Backup)

**Verdict: COMPLIANT with caveats**

| SPEC 12 Requirement | Implementation | Status |
|---|---|---|
| Synchronous writes with fsync before ack | `_sync_write()` does flush + os.fsync, atomic rename via tmp file | PASS |
| Layer 1: `~/.kognis/<plugin>/state/` | `state_dir = ~/.kognis/<plugin>/state/`, file `current.json` | PASS |
| Layer 2: every 30 min, `~/.kognis/backup/<plugin>_<timestamp>.tar.gz` | `create_snapshot()` creates tar.gz in backup_dir with correct naming; `LAYER2_INTERVAL_SECONDS = 1800` | PASS |
| Layer 2 retention: 7 days | `LAYER2_RETENTION_DAYS = 7`, `_prune_old_snapshots()` implemented | PASS |
| Layer 3: daily external backup | Not implemented; no external backup target support | **GAP** |
| Layer 3 retention: 30 days | `LAYER3_RETENTION_DAYS = 30` constant defined but not used | PARTIAL |
| Restore protocol: L1 -> L2 -> L3 -> CRITICAL alert | `load()` tries L1 then L2, raises StateStoreError if no valid backup | PASS (2 of 3 layers) |
| Critical plugins: full 3-layer backup | Not distinguished; no plugin criticality flag | **GAP** |
| Permanent backups for critical events | Not implemented | **GAP** |
| Backup management: pruning old snapshots | Layer 2 pruning implemented; Layer 3 not applicable (no Layer 3) | PASS (partial) |

**Caveat on L1 fsync:** The implementation does `f.flush()` + `os.fsync(f.fileno())` then `tmp_path.replace(self.state_file)`. On Linux with ext4, `replace()` (which calls `os.rename()`) is atomic on the same filesystem. However, `os.fsync()` on the file does not guarantee the directory entry is persisted. A fully robust implementation would also fsync the directory after the rename. This is a known limitation of the current approach and is acceptable for Phase 4, but should be hardened in production.

---

## 2. Evaluation of Phase 4 Spec Clarifications

The Phase 4 summary flags 3 spec clarifications needed. Evaluation of each:

### 2.1 Clarification 1: SPEC 04 Protobuf Schema Alignment

> "When the Go gRPC server is built, we need to confirm the exact protobuf message schemas match what the SDK expects."

**Validity: VALID CONCERN.** This is a legitimate integration risk. The SDK defines Python-side message dataclasses (RegisterRequest, RegisterAck, etc.) that will need to match the Go-generated protobuf stubs. The concern is well-founded because:
- The SDK's `RegisterRequest.to_dict()` only serializes a subset of manifest fields (manifest_version, plugin_id, plugin_name, version) while the full manifest is much richer.
- The spec says "Serialization: Protocol Buffers" but the SDK uses JSON-serializable dicts.
- The Go core will need to define `.proto` files that produce Python stubs compatible with the current SDK dataclasses.

**Recommendation:** The `.proto` definitions should be treated as a shared contract. When Phase 5 begins, the Go core should generate Python protobuf stubs and the SDK should either wrap them or migrate to them. The current dataclass-based approach is a valid interim.

### 2.2 Clarification 2: StateStore (filesystem) vs. Core Memory (SQLite+FTS5+ChromaDB)

> "The Python SDK's StateStore uses filesystem durability. The Go core's memory subsystem will use SQLite+FTS5+ChromaDB. Confirm this dual approach is intentional."

**Validity: VALID CONCERN -- and the answer is YES, it is intentional.** The spec (SPEC 12) addresses plugin-local state durability, which is distinct from the shared memory subsystem. The CLAUDE.md technology stack lists "Memory storage: SQLite + FTS5 (metadata), ChromaDB (embeddings)" for the core daemon's memory subsystem. The StateStore in the SDK serves a different purpose: plugin-local state that a stateful plugin needs to persist across restarts (e.g., Cognitive Core's working state, Persona's emotional state). These are not the same as episodic memories or embedding-indexed knowledge.

The dual approach is architecturally correct:
- **SDK StateStore (filesystem):** Plugin-local key-value state, crash recovery, 3-layer backup chain. Used by individual plugins for their own state.
- **Core Memory (SQLite+ChromaDB):** Shared episodic memory, semantic search, cross-plugin knowledge retrieval. Used by the Memory plugin on behalf of all plugins.

**Recommendation:** This should be documented explicitly in a spec amendment or a clarifying note in SPEC 12, distinguishing "plugin-local state durability" from "shared memory subsystem."

### 2.3 Clarification 3: ContextBudgetManager error handling

> "The SDK just raises the error; the core should handle the model-switch logic. Confirm this separation is correct."

**Validity: VALID CONCERN -- and the separation IS correct.** SPEC 10 Section 10.3 step 6d says "If MUST+HIGH > budget: raise error (KGN-CONTEXT-TRIM_FAILED)." The SDK correctly raises this error. The spec's phrase "alert to cognitive core for model switch" describes what the system should do in response to the error, not what the budget manager itself should do. The ContextBudgetManager is an SDK component running inside a plugin; it cannot switch models. The Cognitive Core (a separate plugin) or the core daemon should catch this error and decide whether to:
1. Switch to a model with a larger context window
2. Reduce the input further
3. Alert the creator

The SDK's responsibility is to raise the error reliably. The system's responsibility is to handle it. This is clean separation of concerns.

**Recommendation:** The spec could be clarified to explicitly state that KGN-CONTEXT-TRIM_FAILED is a signal to the broader system, not an action the ContextBudgetManager takes. Add a note: "The budget manager raises this error; the receiving plugin or core daemon decides on remediation."

---

## 3. Architectural Consistency with Kognis Framework Vision

### 3.1 Foundation 01 (Vision) Alignment

The SDK implementation is consistent with the vision of a "continuously-conscious digital being":

- **StatefulAgent with continuous loop:** Implements the "thinks even when no one is interacting" requirement via the `cognition_cycle()` loop.
- **Stateless handlers for nervous system plugins:** Plugin base class supports handler-only mode.
- **State durability:** StateStore ensures state survives restarts, supporting "persistent identity."
- **Environmental awareness:** The capability system and event bus enable plugins to share perception data.

### 3.2 Foundation 04 (Nervous System + Brain Regions) Alignment

The two-mode architecture is correctly implemented:

- **Mode A (Stateless):** `Plugin` base class with `register_slot_handler()` for handler-based dispatches.
- **Mode B (Stateful Agent):** `StatefulAgent` with continuous `cognition_cycle()`, working memory, wake/sleep hooks.

**Gaps vs. Foundation 04 Section 4.4:**

The foundation specifies these StatefulAgent methods:
- `inner_loop()` -- SDK has `cognition_cycle()` (equivalent, renamed)
- `handle_dispatch()` -- SDK has `on_dispatch()` (equivalent)
- `handle_sidebar_event()` -- NOT IMPLEMENTED. This is a channel for injecting mid-cognition events (e.g., a new perception arriving while the agent is thinking). The current architecture has no mechanism for this.
- `handle_idle()` -- NOT IMPLEMENTED as a separate method. `cognition_cycle()` serves as both the idle and active loop.
- `on_shutdown()` -- Implemented

**Recommendation:** Add `handle_sidebar_event()` to StatefulAgent. This is architecturally important because the foundation document explicitly identifies it as necessary for the "subscribes to sidebar events mid-cognition" characteristic of brain-region plugins. Without it, Cognitive Core cannot process urgent inputs (like emergency wake) during a cognition cycle.

---

## 4. Spec Violations and Drift Summary

| # | Severity | Module | Issue |
|---|---|---|---|
| V1 | **HIGH** | capability.py | `list_for_llm()` does not filter `authentication_required=True` capabilities, violating SPEC 11 Section 11.5 security boundary |
| V2 | **HIGH** | capability.py | `list_for_llm()` returns `capability_id` as tool description instead of `llm_tool_description`, causing incorrect LLM tool schemas |
| V3 | **MEDIUM** | control_plane.py | Emergency bypass protocol (SPEC 04 Section 4.6 / SPEC 14) not implemented |
| V4 | **MEDIUM** | stateful_agent.py | `handle_sidebar_event()` missing, breaking Foundation 04 mid-cognition event injection |
| V5 | **MEDIUM** | manifest.py | Sleep behavior parsed as single string instead of 4-stage structure from SPEC 02 |
| V6 | **MEDIUM** | manifest.py | Structured permissions (filesystem/network/hardware) flattened to `list[str]` |
| V7 | **LOW** | manifest.py | Several manifest sections not parsed: ui, maturity_gate, configuration schema, sdk.manifest_schema |
| V8 | **LOW** | state_store.py | Layer 3 (daily external backup) not implemented |
| V9 | **LOW** | control_plane.py | STARTING state skipped in plugin lifecycle transitions |
| V10 | **LOW** | health.py | StateBroadcaster returns unpublished StateChange when old==new |

---

## 5. SPEC 08 Lifecycle State Machine Compliance

The PluginState enum correctly defines all 10 states from SPEC 08. The transition logic in Plugin and StatefulAgent covers:

- UNREGISTERED -> REGISTERED -> HEALTHY_ACTIVE -> SHUTTING_DOWN -> SHUT_DOWN

**Missing transitions:**
- REGISTERED -> STARTING -> HEALTHY_ACTIVE (STARTING is skipped)
- HEALTHY_ACTIVE -> UNHEALTHY (no degraded-metrics detection)
- HEALTHY_ACTIVE -> UNRESPONSIVE (no missed-heartbeat detection)
- UNRESPONSIVE -> STARTING (no restart initiation)
- UNRESPONSIVE -> CIRCUIT_OPEN (no restart-failure tracking)
- CIRCUIT_OPEN -> STARTING (no cooldown logic)
- CIRCUIT_OPEN -> DEAD (no max-attempts tracking)

These are primarily core-daemon responsibilities. The SDK client should at minimum provide hooks for receiving state transition commands from the core (e.g., core tells plugin it is now UNHEALTHY, plugin adjusts behavior). Currently, the plugin can only set its own state, not receive external state transitions.

---

## 6. SPEC 04 Handshake Protocol Compliance

The 4-step registration is correctly modeled. Specific protocol observations:

**Registration (Section 4.2):** Steps 1-4 are present with correct field names and timeout constants. The implementation is a synchronous simulation -- `register()` immediately returns a synthetic `RegisterAck` rather than waiting for a real gRPC response.

**Graceful Shutdown (Section 4.3):** Steps 1-4 are present. The implementation transitions states correctly but does not implement the full flow of "core sends SHUTDOWN_REQUEST, plugin completes in-flight work, persists state, sends SHUTDOWN_READY, core sends SHUTDOWN_CONFIRMED." The `shutdown()` method is a self-initiated shutdown, not a response to a core-requested one.

**Dispatch (Section 4.4):** The dispatch lifecycle (DISPATCH -> ACK -> PROCESSING -> COMPLETE/FAILED) is correctly modeled with the `DispatchStatus` enum and message dataclasses. The `dispatch()` method validates the result envelope.

**Double Handshake (Section 4.5):** The `CapabilityQuery` and `CapabilityResponse` dataclasses represent the query/response pair. The full 7-message exchange (4 acknowledgments) is abstracted into a single call. This is an acceptable simplification for the SDK side, but the Go core must implement the full protocol.

**Heartbeat (Section 4.7):** `Heartbeat` and `HeartbeatAck` dataclasses are present. The periodic emission logic is in the Plugin/StatefulAgent run loops rather than in the ControlPlaneClient itself.

---

## 7. SPEC 10 Context Budget Trim Algorithm Assessment

The trim algorithm in `ContextBudgetManager.assemble()` is **correct**:

1. Calculate available budget = window - output_budget - safety_margin
2. Add all MUST blocks (unconditional)
3. Add all HIGH blocks (unconditional)
4. If MUST + HIGH > budget, raise KGN-CONTEXT-TRIM_FAILED
5. Add MEDIUM blocks greedily (each fits if remaining budget allows)
6. Add LOW blocks greedily (each fits if remaining budget allows)
7. Log any trimmed blocks

The effect is that when budget is tight, LOW blocks are trimmed first (they are added last, so they are the first not to fit), then MEDIUM blocks. MUST and HIGH are never trimmed. This matches the spec.

**One nuance:** The spec says for LOW trimming, "summarize via cheap model OR drop with note." The implementation simply drops the block and logs it. The "summarize via cheap model" option is not implemented. This is acceptable for Phase 4 but should be noted as a future enhancement -- summarization requires LLM access that the ContextBudgetManager does not have.

---

## 8. SPEC 12 Three-Layer Durability Assessment

The StateStore implements a credible 3-layer durability model:

**Layer 1 (Synchronous writes):** Correctly implemented with:
- Atomic write via temp file + `os.rename()`
- `f.flush()` + `os.fsync()` before rename
- JSON serialization of full state dict

**Layer 2 (Periodic snapshots):** Correctly implemented with:
- tar.gz snapshots at `~/.kognis/backup/<plugin>_<timestamp>.tar.gz`
- 30-minute interval constant
- 7-day retention with pruning
- Restore from latest snapshot on corruption

**Layer 3 (Daily external):** NOT IMPLEMENTED. The constant `LAYER3_RETENTION_DAYS = 30` exists but no backup-to-external logic. This requires a configurable external target (NAS, cloud), which is a deployment concern rather than an SDK concern. The SDK should provide an interface for Layer 3 backup targets.

**Restore protocol:** Implements L1 -> L2 correctly. If both fail, raises `StateStoreError("no_valid_backup")`. The spec says "If no valid backup, raise CRITICAL alert -- do not silently start with empty state." The error is raised, but the "CRITICAL alert" part (notifying the creator) requires the health/alert system, which is not wired into the StateStore. The error propagation is correct; the alert escalation is a system-level concern.

**Missing from spec:** The spec says "Critical events (major milestones, identity changes) create permanent backups." Not implemented. This requires a `create_permanent_backup()` method and a separate storage path for permanent snapshots.

---

## 9. Summary of Findings

### Critical (must fix before Phase 5 integration):
1. **SPEC 11 security violation:** `authentication_required=True` capabilities not filtered from LLM exposure
2. **SPEC 11 tool description bug:** `list_for_llm()` returns capability_id instead of llm_tool_description

### Important (should fix during Phase 5):
3. Emergency bypass protocol (SPEC 04/14) absent from control plane
4. `handle_sidebar_event()` missing from StatefulAgent (Foundation 04 requirement)
5. Manifest parser silently drops several spec-defined fields (sleep_behavior, permissions, health, ui, maturity_gate)
6. Plugin does not receive external state transitions (UNHEALTHY, UNRESPONSIVE) from core

### Acceptable for Phase 4 (track as tech debt):
7. Layer 3 external backup not implemented
8. STARTING state skipped in lifecycle
9. StateBroadcaster returns unpublished StateChange on no-op
10. Manifest validation covers 4 of 8 spec rules
11. Double-handshake 7-message exchange simplified to 1 call
12. "Summarize via cheap model" in context trim not implemented

### Spec Clarifications (all 3 are valid):
1. **SPEC 04 protobuf alignment** -- Legitimate integration risk; addressed by treating .proto files as shared contract in Phase 5
2. **StateStore vs. Core Memory dual approach** -- Intentional and correct; plugin-local state != shared memory subsystem. Recommend spec amendment to clarify.
3. **ContextBudgetManager error vs. system action** -- Correct separation; SDK raises error, system handles remediation. Recommend spec note.

### Architectural Consistency:
The SDK implementation is well-aligned with the Kognis framework vision. The two-mode architecture (Stateless Plugin + StatefulAgent) correctly implements the nervous-system/brain-regions insight from Foundation 04. The continuous cognition loop, durable state, and capability-based inter-plugin communication all support the vision of a continuously-conscious being. The identified gaps are fixable without architectural changes.

---

*Review completed. 10 spec violations/gaps identified (2 high, 4 medium, 4 low). 3 spec clarifications validated. Architecture consistent with framework vision.*