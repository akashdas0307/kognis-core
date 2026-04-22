# Context Budget Manager

> **Stability:** EVOLVING
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [01-message-envelope.md](01-message-envelope.md), [11-tool-bridge.md](11-tool-bridge.md)

## 10.1 Purpose

Every LLM-using plugin needs disciplined context management. The Context Budget Manager is a required SDK component that prevents context overflow and ensures critical information always makes it into prompts.

## 10.2 Context Block Priority Tiers

| Tier | Examples |
|---|---|
| MUST | Identity block, Instruction block, Current input envelope |
| HIGH | Sidebar critical items, Recent memories (last 24h), Emotional state |
| MEDIUM | State block, Environmental summary, Older memories (days-weeks) |
| LOW | Background context, Historical memories (months+), Meta-information |

## 10.3 Budget Algorithm

Before every LLM call:

```
1. Get target model's context window from Inference Gateway capability
2. Reserve output budget (default: 4000 tokens, configurable)
3. Reserve safety margin (default: 500 tokens)
4. Available input budget = window - output - margin
5. Assemble context blocks with priority
6. If total > budget:
   a. Trim LOW tier first (summarize via cheap model OR drop with note)
   b. Then MEDIUM tier
   c. Never trim MUST or HIGH
   d. If MUST+HIGH > budget: raise error (KGN-CONTEXT-TRIM_FAILED)
7. Make LLM call with trimmed context
8. Log trim action if triggered (for adaptive feedback)
```

## 10.4 Adaptive Feedback

When trimming happens frequently, system adjusts:
- Memory plugin reduces default retrieval size
- Environmental summary shortens
- Cognitive Core may compact working memory

This prevents constant trimming — system self-tunes.

## 10.5 Long Session Compaction

For stateful agents in long reasoning sessions (e.g., daydream over 30 minutes):
- Periodically compact own working memory
- Take reasoning-so-far → summarize to half size → continue from summary
- Mirrors human memory fade during sustained thought