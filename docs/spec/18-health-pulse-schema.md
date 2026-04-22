# Health Pulse Schema

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [06-state-broadcast.md](06-state-broadcast.md), [07-error-taxonomy.md](07-error-taxonomy.md), [08-plugin-lifecycle.md](08-plugin-lifecycle.md)

## 18.1 Pulse Format

```yaml
health_pulse:
  plugin_id: string
  timestamp: ISO8601
  status: string     # HEALTHY | DEGRADED | ERROR | CRITICAL | UNRESPONSIVE
  
  metrics:
    # Plugin-specific, defined in manifest
    queue_depth: integer
    processing_latency_p50_ms: number
    processing_latency_p99_ms: number
    error_count_last_60s: integer
    memory_usage_mb: number
    # ...
  
  current_activity: string  # Brief description
  last_dispatch_at: ISO8601 | null
  
  alerts:
    - severity: string  # warning | error | critical
      code: string
      message: string
```

## 18.2 Aggregation

Core aggregates pulses into Health Registry:
- Last N pulses per plugin (default 100)
- Current status
- Metric time-series (short window)
- Derived metrics (uptime %, error rate)

## 18.3 Visibility

Dashboard shows real-time health pulses.
System Health Layer 1 uses registry for anomaly detection.
System Health Layer 3 uses pulse history for root-cause analysis.