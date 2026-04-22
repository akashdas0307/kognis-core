# Sleep Stage Behaviors

> **Stability:** EVOLVING
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [14-emergency-bypass.md](14-emergency-bypass.md), [09-mutation-semantics.md](09-mutation-semantics.md), [12-durability-backup.md](12-durability-backup.md)

## 16.1 Stage Definitions

Four stages with adaptive duration:

| Stage | Duration | Interruptibility |
|---|---|---|
| Stage 1 — Settling | 30-60 min | HIGH (any Tier 1/2 wakes) |
| Stage 2 — Maintenance | 1-2 hrs | MEDIUM (Tier 1 wakes, maintenance pauses) |
| Stage 3 — Deep Consolidation | 3-6 hrs | LOW (sleepwalking mode for non-critical) |
| Stage 4 — Pre-Wake | 30-60 min | HIGH |

Total sleep duration: 6-12 hours, adaptive.

## 16.2 Plugin Sleep Behavior Declaration

Each plugin declares in manifest:

```yaml
sleep_behavior:
  stage_1_settling: continue_normal
  stage_2_maintenance: reduced_activity
  stage_3_deep_consolidation: monitoring_only
  stage_4_pre_wake: continue_normal
  
  maintenance_jobs:
    - name: database_optimize
      estimated_duration_seconds: 300
    - name: log_rotation
      estimated_duration_seconds: 60
```

## 16.3 Stage Transition Broadcasts

Core broadcasts system stage:

```
state.system.stage = "sleep_stage_3"
```

Plugins react per declared behavior.

## 16.4 Emergency Wake

On Tier 1 CRITICAL input during Stage 3:
1. Sleep/Dream checkpoints current consolidation job (2 seconds)
2. Stage 4 runs COMPRESSED (2-3 min)
3. Cognitive Core wakes with reduced context handoff
4. Emergency processed
5. If time remains in planned sleep window: re-enter Stage 3 from checkpoint
6. If not: stay awake, note sleep debt

## 16.5 Sleep Debt Tracking

System tracks incomplete consolidation:
```yaml
sleep_debt:
  episodes_unprocessed: int
  contradictions_unresolved: int
  skills_unrefined: int
  last_clean_sleep: timestamp
```

If debt accumulates, Sleep/Dream requests extended rest from creator.

## 16.6 Adaptive Duration Algorithm

Before sleep:
```python
estimated_work_time = (
    episodes_count * avg_consolidation_time +
    contradictions_count * avg_resolution_time +
    skills_with_new_data * avg_refinement_time +
    offspring_pending_tests * estimated_test_time +
    base_maintenance_time
)

planned_duration = min(
    max(
        estimated_work_time + STAGE_1_TIME + STAGE_4_TIME,
        MIN_SLEEP_HOURS * 3600
    ),
    MAX_SLEEP_HOURS * 3600
)
```