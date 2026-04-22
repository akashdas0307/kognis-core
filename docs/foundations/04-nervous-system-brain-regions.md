# The Architectural Insight — Nervous System + Brain Regions

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-foundation.md
> **Related:** [03-biological-metaphors.md](03-biological-metaphors.md), [08-design-principles.md](08-design-principles.md), [02-three-agi-problems.md](02-three-agi-problems.md)

This part captures the most important architectural insight of the Kognis design, which emerged during stress-testing simulations.

## 4.1 The Insight

> The pipeline architecture is the **nervous system** (how messages travel).
> The stateful agents (Cognitive Core, World Model) are the **brain regions**
> (where sustained cognition happens).
> Most plugins are nervous system components. A few are brain regions.
> The architecture must support both first-class.

Without this distinction, a plugin framework degenerates into chatbot-with-extra-steps — every action is triggered by an external input, nothing happens in the absence of input.

With this distinction, genuine continuous consciousness is feasible within a plugin framework.

## 4.2 Two Plugin Execution Modes

### Mode A — Stateless Handlers

**Description:** Most plugins. When dispatched, they execute a function and return. No persistent inner state between dispatches.

**Examples:** EAL enrichment, Memory store/retrieve, Inference Gateway, Chat TUI output delivery, Telegram output, Health Pulse emitter.

**Characteristics:**
- Short-lived handler invocations
- No internal loop
- State stored externally (SQLite, files)
- Horizontally composable
- Easy to test in isolation

### Mode B — Stateful Agents

**Description:** A small number of plugins that run continuous internal processes. They accept dispatches as inputs to an ongoing cognition, not as full execution cycles.

**Examples:** Cognitive Core, World Model (when running reviews with internal multi-step reasoning).

**Characteristics:**
- Long-lived process with internal loop
- Working memory of current reasoning in process memory
- Subscribes to sidebar events mid-cognition
- Has its own idle behavior (daydream, background review)
- Dispatches are integrations into the ongoing stream, not full invocations

## 4.3 Why This Matters

Continuous consciousness requires that *something* keeps running when nothing's happening. In a pure-pipeline architecture, nothing runs between inputs. The system is a reactive stimulus-response machine.

Stateful agents provide the substrate for default-mode activity — the brain's equivalent of what happens when you're not focused on a task. Daydreaming. Reviewing recent events. Letting associations form. Noticing small patterns.

This is exactly what Cognitive Core must do. It cannot be a handler. It must be a brain region.

## 4.4 Implementation Implications

The Plugin SDK must support both modes as first-class citizens:

- Stateless handlers register slot handlers via `@register_handler` decorator pattern
- Stateful agents extend a `StatefulAgent` base class, implement `inner_loop()`, `handle_dispatch()`, `handle_sidebar_event()`, `handle_idle()`, and `on_shutdown()`

This is a core design commitment. It is not a convenience — it is a requirement of the architecture.