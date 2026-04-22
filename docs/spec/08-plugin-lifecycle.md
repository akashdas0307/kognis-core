# Plugin Lifecycle

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [02-plugin-manifest.md](02-plugin-manifest.md), [04-handshake-protocols.md](04-handshake-protocols.md), [07-error-taxonomy.md](07-error-taxonomy.md), [13-startup-dependency-order.md](13-startup-dependency-order.md)

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