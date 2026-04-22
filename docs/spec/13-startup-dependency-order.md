# Startup Dependency Order

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [08-plugin-lifecycle.md](08-plugin-lifecycle.md), [02-plugin-manifest.md](02-plugin-manifest.md)

## 13.1 Purpose

Plugins depend on each other. The core daemon must start them in correct order.

## 13.2 Dependency Declaration

In manifest:

```yaml
startup_dependencies:
  hard: [inference-gateway]  # Must be HEALTHY before this plugin starts
  soft: [memory, persona]     # Prefer HEALTHY but can start in parallel
```

## 13.3 Critical Startup Order

Based on the design, approximate order:

1. **Core daemon** (Go) — the orchestrator itself
2. **Inference Gateway** — LLM infrastructure
3. **Memory** — needed by most cognitive operations
4. **Persona Manager** — identity provider
5. **World Model** — review layer
6. **Thalamus** — input gateway
7. **EAL** — environmental awareness
8. **Prajna** — cognitive pipeline (Cognitive Core inside)
9. **Brainstem** — output gateway
10. **Communication plugins** (Chat TUI, Telegram, Voice)
11. **Sleep/Dream** — can start later
12. **System Health** — can start later
13. **Offspring** — starts last, only during Adolescence+

## 13.4 Topological Resolution

Core does:
1. Read all manifests, extract dependencies
2. Topologically sort
3. Detect cycles — if any, refuse to start, alert creator
4. Start in order
5. Wait for each HEALTHY before starting next wave
6. Timeout per plugin: 60 seconds default

## 13.5 Partial Startup

If a plugin fails to start:
- If hard dependency of others: others wait
- If soft dependency: others start anyway
- System continues with degraded capability
- Creator notified