# Error Taxonomy

> **Stability:** EVOLVING
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [08-plugin-lifecycle.md](08-plugin-lifecycle.md), [04-handshake-protocols.md](04-handshake-protocols.md)

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