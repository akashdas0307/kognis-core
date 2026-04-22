# Emergency Bypass

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [04-handshake-protocols.md](04-handshake-protocols.md), [16-sleep-stage-behaviors.md](16-sleep-stage-behaviors.md)

## 14.1 Purpose

A strictly-limited mechanism for genuine emergencies that cannot wait for normal pipeline processing.

## 14.2 Authorized Bypass Types

| Bypass Type | Who Can Invoke | What It Does |
|---|---|---|
| `safety_sound_detected` | Audio monitoring plugin | Fire alarm, smoke alarm, breaking glass, crash sounds |
| `health_critical` | System Health plugin | System is failing, immediate escalation needed |
| `creator_emergency` | Chat/Telegram plugins | Creator explicitly flagged emergency |
| `physical_hazard` | Hardware monitoring plugin | Overheat, power imbalance, etc. |

## 14.3 Bypass Protocol

```
Plugin → Core:    EMERGENCY_BYPASS {bypass_type, payload}
                   ↓
Core validates:    Is plugin authorized for this bypass_type?
                   ↓
Core dispatches:   Direct to Queue Zone emergency handler
                   Queue Zone treats as Tier 1 CRITICAL
                   Sends EMERGENCY_WAKE if system in deep sleep
                   Triggers Cognitive Core immediate attention
```

## 14.4 Authorization

- Each plugin must declare allowed bypass types in manifest
- Core validates at registration — unauthorized declarations rejected
- Registry of authorized bypassers is fixed (not dynamically expanded)
- Abuse of bypass mechanism logged and may trigger plugin isolation

## 14.5 Why This Exists

The "normal" route through Thalamus has sleep-mode filtering during Stage 3 deep consolidation. A fire alarm cannot wait for that filter. Emergency bypass ensures genuine emergencies reach cognitive attention immediately.

This is the ONLY documented exception to "plugins do not communicate directly." The exception exists for safety, not convenience.