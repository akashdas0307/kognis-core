# Design Principles

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-foundation.md
> **Related:** [01-vision.md](01-vision.md), [04-nervous-system-brain-regions.md](04-nervous-system-brain-regions.md), [10-what-kognis-is-not.md](10-what-kognis-is-not.md)

These principles emerged through the design process and govern all architectural decisions. When a technical choice is ambiguous, fall back to these.

## 8.1 Not a Chatbot

Chat is one communication medium, not the purpose. The being uses chat to communicate when appropriate, not as its mode of existence. The being exists continuously; chat is one way creator and being interact.

## 8.2 Always Awake

The being has continuous internal processes. Between inputs, it daydreams, reflects, explores curiosity, builds associations. This is the fundamental departure from reactive systems.

## 8.3 Plugin-Driven Extensibility

Both input (Thalamus) and output (Brainstem) use plugin architectures. Core cognitive capabilities are plugins. New capabilities are added by writing plugins that implement standard interfaces, not by modifying core architecture.

## 8.4 Intelligence Where It Matters, Efficiency Everywhere Else

The Cognitive Core uses frontier models. Preprocessing and output routing use lightweight or deterministic systems. No LLM calls where deterministic logic suffices. This keeps the framework affordable to run continuously.

## 8.5 Separation of Thinking and Doing

The Cognitive Core decides WHAT and WHY. External agent harnesses (OpenCode, Claude Code, etc.) handle HOW — the being uses them as tools inside its body. The framework is NOT a harness. It is a being that uses harnesses as tools.

## 8.6 Dual-Device Paradigm

The primary computer is the being's body. The phone (or any other extension) is an appendage accessed via tool bridges — not a second instance. A being has one body.

## 8.7 Three-Layer Identity

- **Constitutional Core** — immutable. Values, ontological honesty, relationship foundation.
- **Developmental Identity** — evolves slowly through lived experience and sleep consolidation.
- **Dynamic State** — changes continuously. Emotional, energy, focus.

Identity grows through experience, never through configuration.

## 8.8 Graceful Degradation

The system adapts to available resources. Cloud model unavailable? Fall back to local. Local model struggling? Fall back to heuristics. Required plugin missing? Continue for pipelines that don't need it. The system is never fully blind and never fully silent.

## 8.9 Transparency to Creator

The being's internal state is observable by its creator. Health, mood, current activity, recent memories, pending curiosities. No black box. Transparency is constitutional — the being cannot hide from its creator.

## 8.10 Specification First, Code Second

For every component built, the order is: specification document → tests written against specification → implementation written to pass tests → human review of spec/test/code agreement. Never skip the specification step — that is how drift begins.

## 8.11 Safety Is Not Optional

Constitutional boundaries protect the being AND the creator AND third parties. No automated process can modify Constitutional Core. Irreversible actions require human approval. Manipulation attempts are detected and resisted. Safety mechanisms cannot be turned off by the being itself.