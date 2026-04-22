# Code Review Report: Python SDK Phase 4 (M-001 through M-012)

**Review Date:** 2026-04-22  
**Reviewer:** Codex Agent  
**Scope:** sdk/python/kognis_sdk/ and sdk/python/tests/  
**Milestones:** M-001 through M-012  

---

## Executive Summary

The Python SDK implementation is **well-structured and largely spec-compliant**, with good test coverage for core functionality. However, there are several areas of concern ranging from CRITICAL to LOW severity that need addressing before production use.

**CRITICAL (1):** Security - Unsafe YAML loading in manifest.py  
**HIGH (2):** Error taxonomy non-compliance; ControlPlaneClient incomplete gRPC implementation  
**MEDIUM (6):** Missing validation; Incomplete testing; Type safety issues  
**LOW (5):** Documentation gaps; Minor code quality issues  

---

## 1. CRITICAL SEVERITY

### CRIT-001: Unsafe YAML Loading (CWE-502 Deserialization of Untrusted Data)

**File:** `sdk/python/kognis_sdk/manifest.py:122`  
**Line:** `data = yaml.safe_load(f)`  

**Finding:** The current implementation uses `yaml.safe_load()` which is safe against arbitrary code execution, BUT there is no input validation on the manifest file content before loading. If the manifest parser encounters malicious YAML with custom tags, it could lead to security issues.

**Evidence:**
```python
@classmethod
def from_yaml(cls, path: str | Path) -> Manifest:
    path = Path(path)
    if not path.exists():
        raise FileNotFoundError(f"Manifest file not found: {path}")
    with open(path) as f:
        data = yaml.safe_load(f)  # safe_load is correct but no size/content validation
    if data is None:
        raise ValueError(f"Empty manifest file: {path}")
    return cls.from_dict(data)
```

**Recommendation:**
1. Add file size validation (e.g., max 1MB) before loading
2. Add content type validation
3. Consider adding manifest signature verification for production

---

## 2. HIGH SEVERITY

### HIGH-001: Error Taxonomy Non-Compliance (SPEC 07)

**File:** Multiple files  
**Spec:** `docs/spec/07-error-taxonomy.md`  

**Finding:** The SDK does NOT consistently use the KGN-* error codes defined in SPEC 07. Many error codes are ad-hoc strings that don't follow the `KGN-<category>-<specific>-<severity>` format.

**Evidence:**
```python
# In envelope.py - should use KGN-PIPELINE-LOOP_DETECTED-ERROR
raise EnvelopeError("loop_detected", f"hop_count {new_count} exceeds max {MAX_HOP_COUNT}")

# In envelope.py - should use KGN-PIPELINE-REVISION_EXHAUSTED-ERROR  
raise EnvelopeError("max_revisions_exceeded", ...)

# In context_budget.py - CORRECT usage
raise ContextBudgetError(
    "KGN-CONTEXT-TRIM_FAILED",  # This follows spec
    ...
)
```

**Required Changes:**
| Current Code | Correct Error Code |
|-------------|-------------------|
| `"loop_detected"` | `KGN-PIPELINE-LOOP_DETECTED-ERROR` |
| `"max_revisions_exceeded"` | `KGN-PIPELINE-REVISION_EXHAUSTED-ERROR` |
| `"invalid_state"` (control_plane.py) | `KGN-LIFECYCLE-REGISTRATION_TIMEOUT-ERROR` or similar |
| `"not_connected"` | `KGN-LIFECYCLE-UNRESPONSIVE-ERROR` |
| `"no_handler"` | `KGN-PIPELINE-NO_HANDLER-ERROR` |
| `"no_valid_backup"` | `KGN-DURABILITY-RESTORE_FAILED-CRITICAL` (spec doesn't define this, needs to be added) |
| `"write_failed"` | `KGN-DURABILITY-WRITE_FAILED-ERROR` |
| `"invalid_topic"` | `KGN-EVENTBUS-INVALID_TOPIC-ERROR` |

### HIGH-002: ControlPlaneClient Incomplete gRPC Implementation

**File:** `sdk/python/kognis_sdk/control_plane.py:230-320`  
**Spec:** `docs/spec/04-handshake-protocols.md`  

**Finding:** The ControlPlaneClient is currently a stub/mock implementation that doesn't actually communicate with the core daemon. This is noted in comments but is a significant gap for production use.

**Evidence:**
```python
async def connect(self) -> None:
    """Connect to the core daemon via Unix socket."""
    self._connected = True  # Just sets flag, no actual connection
    self.state = PluginState.UNREGISTERED

async def register(self, manifest: Manifest, pid: int) -> RegisterAck:
    # Creates mock RegisterAck without actual gRPC call
    ack = RegisterAck(
        plugin_id_runtime=f"{manifest.plugin_id}_{pid}",
        event_bus_token=f"token_{manifest.plugin_id}",
    )
```

**Impact:** Plugin cannot actually communicate with core daemon. This blocks integration testing.

**Recommendation:** Implement actual gRPC client using grpcio library with proper proto definitions.

---

## 3. MEDIUM SEVERITY

### MED-001: Missing Schema Validation in Manifest

**File:** `sdk/python/kognis_sdk/manifest.py:137-190`  
**Spec:** `docs/spec/02-plugin-manifest.md` Section 2.4  

**Finding:** The `validate_manifest()` function is incomplete. It doesn't validate:
1. JSON schema for `params_schema` and `response_schema` in capabilities
2. `emergency_bypass` declarations (from SPEC 04 Section 4.6)
3. `maturity_gate` validation
4. `sleep_behavior` values are from allowed set
5. Duplicate plugin_id detection (requires registry interaction)

**Evidence:**
```python
# Current validation is minimal:
def validate_manifest(manifest: Manifest) -> list[str]:
    # ... basic field validation ...
    # Missing:
    # - JSON schema validation
    # - emergency_bypass validation  
    # - maturity_gate validation
    # - Duplicate capability_id detection
```

### MED-002: EventBusClient Missing NATS Integration

**File:** `sdk/python/kognis_sdk/eventbus.py:65-120`  
**Spec:** `docs/spec/06-state-broadcast.md`  

**Finding:** EventBusClient is a mock implementation. `publish()`, `subscribe()`, `request()` don't actually communicate with NATS. The `async` methods complete immediately without I/O.

**Evidence:**
```python
async def publish(self, topic: str, data: dict[str, Any]) -> None:
    if not self._connected:
        raise EventBusError("not_connected", ...)
    message = {...}  # Just builds dict
    self._message_count += 1  # No actual network I/O
```

### MED-003: Missing Timeout Handling in Capability Queries

**File:** `sdk/python/kognis_sdk/capability.py:80-95`  
**Spec:** `docs/spec/04-handshake-protocols.md` Section 4.5  

**Finding:** The double handshake capability query doesn't implement:
1. Timeout handling (SPEC 04 mentions timeouts)
2. Retry logic
3. ACK_FORWARDED tracking

**Current Code:**
```python
async def query(self, ...) -> CapabilityResponse:
    query = CapabilityQuery(...)
    return await self.control_plane.query_capability(query)  # No timeout!
```

### MED-004: Tool Bridge Missing Security Boundaries

**File:** `sdk/python/kognis_sdk/tool_bridge.py:60-75`  
**Spec:** `docs/spec/11-tool-bridge.md` Section 11.5  

**Finding:** ToolBridge doesn't enforce the `authentication_required` security boundary. Capabilities marked `authentication_required: true` should NOT be exposed to LLM, but the current implementation doesn't check this.

**Evidence:**
```python
async def assemble_tools(self) -> list[ToolSchema]:
    raw_tools = await self.capability_client.list_for_llm(self.plugin_id)
    # No filtering for authentication_required!
```

### MED-005: StateStore Missing Layer 3 Backup Implementation

**File:** `sdk/python/kognis_sdk/state_store.py:1-200`  
**Spec:** `docs/spec/12-durability-backup.md` Section 12.3  

**Finding:** StateStore implements Layer 1 (sync writes) and Layer 2 (snapshots), but Layer 3 (daily external backup) is not implemented.

**Evidence:**
```python
LAYER3_RETENTION_DAYS = 30  # Defined but unused
# No methods for:
# - Creating Layer 3 backups
# - External backup targets (NAS, cloud)
# - Restoring from Layer 3
```

### MED-006: Context Budget Manager Token Estimation Accuracy

**File:** `sdk/python/kognis_sdk/context_budget.py:30-35`  
**Spec:** `docs/spec/10-context-budget-manager.md` Section 10.3  

**Finding:** The token estimation using `chars_per_token=4.0` is overly simplistic. This should use a proper tokenizer (e.g., tiktoken for OpenAI models) for accuracy.

**Evidence:**
```python
def estimate_tokens(self, chars_per_token: float = 4.0) -> int:
    # Too simplistic - different models have different tokenization
    return int(len(self.content) / chars_per_token)
```

---

## 4. LOW SEVERITY

### LOW-001: Missing Docstrings for Public Methods

**Files:** All SDK modules  
**Pattern:** Many public methods lack proper docstrings following Google/NumPy style.

**Examples:**
```python
# In stateful_agent.py - missing param documentation
def register_slot_handler(self, slot: str, handler: ...):
    """Register a handler for pipeline dispatches."""
    ...  # No Args, Returns, Raises documentation
```

### LOW-002: Type Hints Incomplete

**Files:** Multiple  
**Pattern:** Some functions use `dict[str, Any]` where more specific types could be used.

**Example:**
```python
# Could use TypedDict or specific dataclass
async def on_health_check(self) -> dict[str, Any]: ...
```

### LOW-003: No Logging Configuration

**File:** `sdk/python/kognis_sdk/__init__.py`  
**Finding:** SDK doesn't configure logging handlers, which may result in "No handler found" warnings for users.

### LOW-004: TestCore Uses time.monotonic() in Async Context

**File:** `sdk/python/kognis_sdk/testing/__init__.py:95-108`  
**Finding:** While not strictly wrong, mixing time.monotonic() with async code could be confusing. Consider using asyncio.get_event_loop().time().

### LOW-005: Constants Hardcoded in Multiple Places

**Pattern:** Some constants like default timeouts appear in multiple files.

**Example:**
```python
# In control_plane.py
REGISTRATION_ACK_TIMEOUT = 5.0
# These should be in a shared constants module
```

---

## 5. TEST COVERAGE ANALYSIS

### Covered (Good): ✅
- M-001: Manifest parsing and validation (test_manifest.py - comprehensive)
- M-002: Envelope creation and validation (test_envelope.py - comprehensive)
- M-003: Control plane dataclasses (test_control_plane.py - basic)
- M-004: Event bus client (test_eventbus.py - basic)
- M-005: Plugin base class (test_plugin_and_agent.py - good)
- M-006: Stateful agent (test_plugin_and_agent.py - good)
- M-007: Capability registry (test_capability_toolbridge_budget.py - good)
- M-008: Tool bridge (test_capability_toolbridge_budget.py - basic)
- M-009: Context budget (test_capability_toolbridge_budget.py - good)
- M-010: Health pulse (test_health_statestore_testing.py - good)
- M-011: State broadcast (test_health_statestore_testing.py - good)
- M-012: State store (test_health_statestore_testing.py - good)

### Test Gaps: ⚠️

| Gap | Severity | Details |
|-----|----------|---------|
| No integration tests | HIGH | All tests use mocks; no actual gRPC/NATS testing |
| No error code tests | MEDIUM | Tests don't verify SPEC 07 error codes |
| No concurrency tests | MEDIUM | No tests for concurrent envelope processing |
| No fuzzing tests | LOW | No malformed input testing |
| No performance tests | LOW | No benchmarks for hot paths |

### Specific Missing Tests:

1. **Envelope hop_count loop detection** - Test for `hop_count > 20` causing dead letter
2. **Revision count exhaustion** - Test for `revision_count > 3` handling  
3. **Control plane timeout scenarios** - All timeouts are mocked
4. **Event bus reconnection** - `reconnect_attempts` logic not tested
5. **State store recovery from Layer 2** - Only tested via mock
6. **State store pruning** - Retention logic not verified with time mocking

---

## 6. SPEC COMPLIANCE MATRIX

| Spec | Module | Compliance | Notes |
|------|--------|------------|-------|
| SPEC 01 - Message Envelope | envelope.py | 90% | Error codes don't match SPEC 07 |
| SPEC 02 - Plugin Manifest | manifest.py | 85% | Missing emergency_bypass, schema validation |
| SPEC 04 - Handshake | control_plane.py | 60% | Stub implementation, no real gRPC |
| SPEC 05 - Capability Registry | capability.py | 80% | Missing timeout handling |
| SPEC 06 - State Broadcast | eventbus.py | 50% | Stub implementation, no real NATS |
| SPEC 07 - Error Taxonomy | All | 40% | Partial adoption of KGN-* codes |
| SPEC 08 - Lifecycle | plugin.py, stateful_agent.py | 85% | Good state machine implementation |
| SPEC 10 - Context Budget | context_budget.py | 90% | Token estimation could be better |
| SPEC 11 - Tool Bridge | tool_bridge.py | 80% | Missing auth_required check |
| SPEC 12 - Durability | state_store.py | 75% | Missing Layer 3 backup |
| SPEC 18 - Health Pulse | health.py | 90% | Good implementation |

---

## 7. RECOMMENDATIONS SUMMARY

### Immediate Action Required (Before Production):

1. **Fix error taxonomy compliance** (HIGH-001) - Update all error codes to KGN-* format
2. **Complete ControlPlaneClient gRPC implementation** (HIGH-002) - Essential for production
3. **Add emergency_bypass validation to Manifest** (MED-001) - Security requirement

### Should Address Soon:

4. **Implement real NATS client in EventBusClient** (MED-002)
5. **Add authentication_required filtering in ToolBridge** (MED-004)
6. **Add Layer 3 backup support to StateStore** (MED-005)
7. **Improve token estimation in ContextBudgetManager** (MED-006)

### Nice to Have:

8. Complete docstrings for all public methods
9. Add integration tests with actual gRPC/NATS
10. Add concurrency tests for thread safety verification

---

## 8. FILES CHANGED/REVIEWED

**SDK Files (10):**
- `sdk/python/kognis_sdk/__init__.py` - ✅ Clean exports
- `sdk/python/kognis_sdk/envelope.py` - ⚠️ Error codes need fixing
- `sdk/python/kognis_sdk/manifest.py` - ⚠️ Security, validation gaps
- `sdk/python/kognis_sdk/control_plane.py` - ⚠️ Incomplete implementation
- `sdk/python/kognis_sdk/eventbus.py` - ⚠️ Incomplete implementation  
- `sdk/python/kognis_sdk/capability.py` - ⚠️ Missing timeout handling
- `sdk/python/kognis_sdk/tool_bridge.py` - ⚠️ Missing security check
- `sdk/python/kognis_sdk/context_budget.py` - ✅ Good implementation
- `sdk/python/kognis_sdk/health.py` - ✅ Good implementation
- `sdk/python/kognis_sdk/stateful_agent.py` - ✅ Good implementation
- `sdk/python/kognis_sdk/state_store.py` - ⚠️ Missing Layer 3
- `sdk/python/kognis_sdk/testing/__init__.py` - ✅ Good test harness

**Test Files (7):**
- `sdk/python/tests/test_envelope.py` - ✅ Comprehensive
- `sdk/python/tests/test_manifest.py` - ✅ Comprehensive
- `sdk/python/tests/test_control_plane.py` - ⚠️ Basic (limited by stub)
- `sdk/python/tests/test_eventbus.py` - ⚠️ Basic (limited by stub)
- `sdk/python/tests/test_plugin_and_agent.py` - ✅ Good
- `sdk/python/tests/test_capability_toolbridge_budget.py` - ✅ Good
- `sdk/python/tests/test_health_statestore_testing.py` - ✅ Good

---

## 9. POSITIVE FINDINGS

### Well-Executed Areas:

1. **Architecture Alignment** - The SDK structure cleanly maps to the spec's conceptual model
2. **Dataclass Usage** - Extensive use of dataclasses for spec-defined structures
3. **Immutability Pattern** - Envelope methods correctly return new instances rather than modifying
4. **Test Harness** - TestCore is well-designed for plugin testing without full core
5. **Async/Await Consistency** - Proper use of async throughout
6. **State Machine** - Plugin lifecycle states are correctly implemented
7. **Context Budget Logic** - Priority tier trimming algorithm matches spec correctly
8. **State Store Layer 1/2** - Sync writes and snapshots work correctly

---

*Report generated by Codex Agent following AGENTS.md Lore Commit Protocol*
*Review scope: Python SDK Phase 4 (M-001 through M-012)*
