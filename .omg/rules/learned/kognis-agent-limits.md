---
name: kognis-agent-limits
description: Restrict parallel agent execution to prevent context fragmentation and resource contention.
---
# Agent Parallelism Rule

- **Limit**: Maximum of 3 parallel sub-agents (e.g., `omg-executor`, `generalist`) may be active in a single turn.
- **Isolation**: Each parallel agent must operate in a distinct "lane" (separate directory or independent file set).
- **Orchestration**: The primary agent must act as the `omg-director`, reconciling all outputs sequentially before final validation.
