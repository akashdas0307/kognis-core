# Phase 4 Summary — Python SDK Core (M-001 through M-012)

> **Phase:** 4 — Python SDK Core
> **Status:** COMPLETE — Awaiting Human Review (Hard Milestone Boundary)
> **Date:** 2026-04-22

## Phase Goal

Implement the Python Plugin SDK as a series of 12 sequential milestones, providing all client-side components a Kognis plugin needs to register with the core daemon, exchange messages, manage state, and participate in the cognitive framework. Every module references a specific SPEC contract.

## Milestones Completed

| Milestone | Module | Spec Reference | Key Deliverable |
|---|---|---|---|
| M-001 | `manifest.py` | SPEC 02 | Manifest parser with from_yaml/from_dict, validation |
| M-002 | `envelope.py` | SPEC 01 | Immutable envelope pattern, derive, enrich, hop counting |
| M-003 | `control_plane.py` | SPEC 04 | gRPC client stubs, handshake protocols, registration, dispatch, heartbeat |
| M-004 | `eventbus.py` | SPEC 06 | NATS pub/sub wrapper, topic helpers |
| M-005 | `plugin.py` | SPEC 02/08 | Abstract Plugin base class, lifecycle hooks, stateless handler mode |
| M-006 | `stateful_agent.py` | SPEC 02/08 | StatefulAgent with cognition cycle, memory compaction |
| M-007 | `capability.py` | SPEC 05 | CapabilityRegistryClient, double handshake queries, LLM tool listing |
| M-008 | `tool_bridge.py` | SPEC 11 | ToolBridge for LLM tool-call translation |
| M-009 | `context_budget.py` | SPEC 10 | ContextBudgetManager with priority tiers, trimming, adaptive feedback |
| M-010 | `health.py` | SPEC 18/06 | HealthPulseEmitter + StateBroadcaster |
| M-011 | `state_store.py` | SPEC 12 | Three-layer durability with crash recovery |
| M-012 | `testing/__init__.py` | — | TestCore fixture for plugin unit testing |

## Source Files (12 modules + public API)

```
sdk/python/kognis_sdk/
  __init__.py          — Full public API exports, __version__ = "0.1.0"
  manifest.py          — M-001
  envelope.py          — M-002
  control_plane.py     — M-003
  eventbus.py          — M-004
  plugin.py            — M-005
  stateful_agent.py    — M-006
  capability.py        — M-007
  tool_bridge.py       — M-008
  context_budget.py    — M-009
  health.py            — M-010
  state_store.py       — M-011
  testing/__init__.py  — M-012
```

## Test Coverage

| Test File | Cases | Milestones Covered |
|---|---|---|
| `test_manifest.py` | 37 | M-001 |
| `test_envelope.py` | 63 | M-002 |
| `test_control_plane.py` | 12 | M-003 |
| `test_eventbus.py` | 14 | M-004 |
| `test_plugin_and_agent.py` | 17 | M-005, M-006 |
| `test_capability_toolbridge_budget.py` | 26 | M-007, M-008, M-009 |
| `test_health_statestore_testing.py` | 35 | M-010, M-011, M-012 |
| **Total** | **204** | |

**All 204 tests passing.** Plus 59 integration tests from schemas/pipelines phases (263 total).

## Specs Referenced

- SPEC 01 (Message Envelope) — `envelope.py`: immutable pattern, enrichments, hop count, revision tracking
- SPEC 02 (Plugin Manifest) — `manifest.py`: parsing, validation; `plugin.py`: handler mode
- SPEC 04 (Handshake Protocols) — `control_plane.py`: 4-step registration, dispatch, heartbeat
- SPEC 05 (Capability Registry) — `capability.py`: double handshake queries, LLM tool exposure
- SPEC 06 (State Broadcast) — `health.py`: StateBroadcaster, on-change semantics; `eventbus.py`: topic naming
- SPEC 08 (Plugin Lifecycle) — `plugin.py`: UNREGISTERED→SHUT_DOWN transitions; `stateful_agent.py`: wake/sleep
- SPEC 10 (Context Budget Manager) — `context_budget.py`: MUST/HIGH/MEDIUM/LOW tiers, trim algorithm
- SPEC 11 (Tool Bridge) — `tool_bridge.py`: registry→OpenAI/Anthropic schema translation
- SPEC 12 (Durability & Backup) — `state_store.py`: 3-layer backup chain, crash recovery
- SPEC 18 (Health Pulse Schema) — `health.py`: HealthPulseEmitter, valid statuses, alerts

## Known Limitations

1. **All clients are stubs** — ControlPlaneClient and EventBusClient simulate protocol behavior in-memory. They do not connect to real gRPC or NATS servers. This is by design for Phase 4 (SDK-side implementation); real wire integration happens in Phase 5 (Go core) and Phase 6 (E2E tests).

2. **No real gRPC/protobuf** — The control plane client mimics the handshake protocol state machine but has no generated protobuf stubs. The Go core (Phase 5) will define the `.proto` files and generate both Go and Python stubs.

3. **StatefulAgent cognition loop is skeletal** — The `cognition_cycle()` increments a counter and compacts memory at a 100-cycle interval. Real cognitive processing (LLM calls, perception, decision-making) will be implemented by plugin authors using the SDK.

4. **ToolBridge produces schemas but doesn't call LLMs** — It translates capability registry entries to OpenAI/Anthropic tool-call format but does not invoke any LLM API. That's the plugin author's responsibility.

5. **StateStore uses filesystem, not SQLite** — SPEC 12 specifies SQLite+FTS5+ChromaDB for the Go core. The Python SDK's StateStore provides a simpler filesystem-based durability layer suitable for plugin-local state. The core's memory subsystem (Phase 5) will provide the full memory API.

6. **No async runtime integration** — Plugin start/stop methods are async but don't manage their own event loops. A plugin runner utility (future work) would handle loop lifecycle.

7. **TestCore dispatches synchronously** — The test harness dispatches to registered handlers directly. It doesn't simulate pipeline routing (multiple slots in sequence) or hop counting.

## Recommended Next Phase Work

Phase 5 — Core Daemon in Go (M-013 through M-025):

1. **M-013**: Plugin registry — verify existing Go scaffolding from M-001, extend with full SPEC 04/08 compliance
2. **M-014**: Capability registry — SPEC 05 implementation in Go
3. **M-015**: NATS event bus embedding — real embedded NATS server
4. **M-016**: gRPC control plane server — protobuf definitions, generated stubs
5. **M-017**: Registration handshake — wire SPEC 04 4-step protocol
6. **M-018**: Pipeline loader — YAML template loading
7. **M-019**: Dispatch table compiler — slot→plugin mapping
8. **M-020**: Message router — hop counting, timeouts, routing info
9. **M-021**: Supervisor with backoff — process management, exponential restart
10. **M-022**: Health aggregator — collect pulses from all plugins
11. **M-023**: Emergency bypass channel — SPEC 14
12. **M-024**: TUI dashboard — bubbletea integration
13. **M-025**: Main entry point — all components wired together

Key dependencies: Go toolchain must be available for compilation/verification.

## Spec Clarifications Needed

1. **SPEC 04**: The handshake protocol specifies a 4-step registration (REGISTER→ACK→READY→ACTIVE). The current SDK client simulates this as state transitions. When the Go gRPC server is built, we need to confirm the exact protobuf message schemas match what the SDK expects.

2. **SPEC 12**: The Python SDK's StateStore uses filesystem durability (sync writes + tar.gz snapshots). The Go core's memory subsystem will use SQLite+FTS5+ChromaDB. These serve different purposes — plugin-local state vs. shared memory. Confirm this dual approach is intentional.

3. **SPEC 10**: ContextBudgetManager raises `KGN-CONTEXT-TRIM_FAILED` when MUST+HIGH blocks exceed budget. The spec says this should trigger "alert to cognitive core for model switch." The SDK just raises the error; the core should handle the model-switch logic. Confirm this separation is correct.

---

**This is a HARD MILESTONE BOUNDARY.** Awaiting human review before proceeding to Phase 5.