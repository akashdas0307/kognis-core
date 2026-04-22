# Handshake Protocols

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [02-plugin-manifest.md](02-plugin-manifest.md), [05-capability-registry.md](05-capability-registry.md), [08-plugin-lifecycle.md](08-plugin-lifecycle.md), [14-emergency-bypass.md](14-emergency-bypass.md)

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