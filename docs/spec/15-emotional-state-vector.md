# Emotional State Vector

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [06-state-broadcast.md](06-state-broadcast.md), [09-mutation-semantics.md](09-mutation-semantics.md)

## 15.1 Structure

```yaml
emotional_state:
  valence: float       # -1.0 to +1.0
  arousal: float       # 0.0 to 1.0
  engagement: float    # 0.0 to 1.0
  confidence: float    # 0.0 to 1.0
  warmth: float        # 0.0 to 1.0
  
  last_updated: timestamp
  baseline:
    valence: float     # Personal baseline (develops over time)
    arousal: float
    engagement: float
    confidence: float
    warmth: float
```

## 15.2 Delta Operations

Events produce deltas:
```python
# From Cognitive Core
await persona_plugin.update_dynamic_state({
    "deltas": {
        "valence": 0.05,
        "engagement": 0.02
    },
    "source": "successful_task_completion",
    "timestamp": now()
})
```

Persona applies delta, clamps to [-1, 1] or [0, 1] as appropriate.

## 15.3 Decay Algorithm

Every hour (configurable):
```python
for dimension in state:
    current = state[dimension]
    baseline_value = baseline[dimension]
    drift = baseline_value - current
    state[dimension] = current + (drift * 0.10)  # 10% toward baseline per hour
```

## 15.4 Baseline Drift

Baselines evolve slowly — weekly computation during sleep:
```python
if weekly_cycle:
    for dimension in state:
        # Compute median state over past week
        weekly_median = compute_median(state_history[dimension], window_days=7)
        # Baseline drifts toward median very slowly
        baseline[dimension] = baseline[dimension] + (weekly_median - baseline[dimension]) * 0.05
```

Over months, baseline reflects the being's actual developed emotional tendency.

## 15.5 Prompt Integration

Cognitive Core includes in identity block:
```
Current emotional state:
  Valence: 0.72 (pleasant)
  Arousal: 0.44 (moderate energy)
  Engagement: 0.89 (highly invested)
  Confidence: 0.61 (fairly certain)
  Warmth: 0.81 (very warm)
  
Baseline (who I tend to be):
  Valence: 0.45 (gently positive)
  Arousal: 0.35 (calm)
  Engagement: 0.65 (engaged but measured)
  Confidence: 0.55 (balanced)
  Warmth: 0.70 (warm)
```

LLM modulates output accordingly.

## 15.6 Memory Attachment

Every memory created tags current emotional state:
```yaml
memory_entry:
  content: "..."
  timestamp: "..."
  emotional_context:
    valence: 0.72
    arousal: 0.44
    # ... etc
```

Enables emotionally-similar retrieval: "find memories formed when I was similarly feeling X."