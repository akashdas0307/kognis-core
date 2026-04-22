# Offspring System

> **Stability:** EVOLVING
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [09-mutation-semantics.md](09-mutation-semantics.md), [08-plugin-lifecycle.md](08-plugin-lifecycle.md)

## 17.1 Components

1. **Improvement Identifier** — collects signals, prioritizes tickets
2. **Spawning Engine** — creates git branches, delegates code modification
3. **Testing Sandbox** — isolated execution environment
4. **Ancestry Tree Manager** — tracks evolutionary history
5. **Harness Integration** — delegates execution to OpenCode/Claude Code

## 17.2 Ancestry Tree Structure

Stored in `[name]-offspring` repo as `ancestry/tree.json`:

```json
{
  "ancestry_version": 1,
  "main_generation": 5,
  "branches": {
    "gen-5-daydream-novelty-threshold": {
      "parent_commit": "abc123",
      "created_at": "2026-04-21T07:00:00Z",
      "status": "PROMOTED",
      "target_improvement": "Daydream novelty threshold 0.75 → 0.65",
      "test_results": {
        "replay_improvement": "+22% novelty acceptance",
        "regression_check": "none detected"
      },
      "promoted_at": "2026-04-22T09:30:00Z"
    }
  }
}
```

## 17.3 One Improvement Per Branch

Critical discipline: each offspring branch modifies ONE thing. Enables clean attribution and rollback.

## 17.4 3-5 Generation Buffer

`main` branch runs 3-5 generations behind the experimental edge. A change must survive 3+ descendant generations before promoting to main.

## 17.5 Status Lifecycle

```
SPAWNED → TESTING → VALIDATED → PROMOTED
                  → ABANDONED
PROMOTED → ROLLED_BACK (if problems detected later)
```

## 17.6 Safety Boundaries

- Cannot modify Constitutional Core
- Cannot modify World Model baseline constitution
- Cannot modify manifest_version schema or protocol definitions
- Can modify thresholds, prompts, pipeline parameters, skill procedures
- Major changes (Cognitive Core reasoning loop) flagged for human review

## 17.7 Test Types

- **Replay Testing** — real inputs from past 24-48h replayed through variant
- **Synthetic Testing** — targeted edge cases
- **Limited Live Testing** — 10-20% of real inputs in parallel (canary pattern)