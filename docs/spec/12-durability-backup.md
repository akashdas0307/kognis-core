# Durability & Backup

> **Stability:** EVOLVING
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [08-plugin-lifecycle.md](08-plugin-lifecycle.md), [09-mutation-semantics.md](09-mutation-semantics.md)

## 12.1 Purpose

State loss is unacceptable. Every stateful plugin must persist state durably and support restart without loss.

## 12.2 Durability Requirements

Each stateful plugin must:
- Write every state change to disk synchronously (fsync) before acknowledging
- Support crash recovery from disk state on restart
- Maintain backup chain for rollback

## 12.3 Three-Layer Backup Chain

| Layer | Frequency | Storage | Purpose |
|---|---|---|---|
| **Layer 1 — Synchronous writes** | Every state change | `~/.kognis/<plugin>/state/` | Crash recovery |
| **Layer 2 — Periodic snapshots** | Every 30 minutes | `~/.kognis/backup/<plugin>_<timestamp>.tar.gz` | Recent corruption recovery |
| **Layer 3 — Daily external** | Once per day | Configurable external (NAS, cloud) | Disaster recovery |

## 12.4 Critical Plugins

Memory, Persona, World Model, Sleep/Dream must follow full 3-layer backup.
Stateless plugins may have lighter requirements (no ongoing state).

## 12.5 Restore Protocol

On plugin startup:
1. Read from Layer 1 (primary state)
2. If corrupt/missing, restore from most recent Layer 2
3. If Layer 2 also gone, restore from Layer 3
4. If no valid backup, raise CRITICAL alert — do not silently start with empty state
5. Creator notified of any restore beyond Layer 1

## 12.6 Backup Management

- Old Layer 2 snapshots pruned after 7 days (configurable)
- Old Layer 3 backups pruned after 30 days (configurable)
- Critical events (major milestones, identity changes) create permanent backups