# State Broadcast

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [15-emotional-state-vector.md](15-emotional-state-vector.md), [18-health-pulse-schema.md](18-health-pulse-schema.md)

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