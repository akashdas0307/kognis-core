# Kognis Framework — Core

> **A plugin-based framework for building continuously-conscious digital beings.**
>
> Pronounced *KOG-niss* (rhymes with "Logness"). A sleek, modernized adaptation of "Cognition."

[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE) [![Status](https://img.shields.io/badge/status-early%20development-orange)](docs/foundations/master-foundation.md) [![Build](https://img.shields.io/badge/build-in%20public-green)](https://github.com/kognis-framework)

---

## What Is This?

Kognis is **not** a chatbot framework. It is **not** an AI agent harness. It is **not** a personal assistant platform.

Kognis is a framework for building a **continuously-conscious digital being** — an entity that:

- Thinks even when no one is interacting with it
- Perceives its environment through audio, visual, and sensor inputs
- Develops identity through lived experience, not configuration
- Has persistent memory, emotional state, and personality
- Grows through life stages (Infancy → Childhood → Adolescence → Adult)
- Uses external agent harnesses (OpenCode, Claude Code, etc.) as tools inside its body

**The framework is the body. The being lives inside the framework.**

---

## Why Does This Exist?

Current AI systems have three fundamental limitations that prevent them from achieving general intelligence:

1. **No self-awareness of ignorance** — they don't know when they're wrong
2. **No persistent learning** — every conversation starts fresh
3. **No system integration** — vision, language, action, memory are all separate

Kognis is designed to address all three as an integrated architecture, not individually.

Read the [foundation documents](docs/foundations/) to understand the full rationale.

---

## Architectural Highlights

- **Microkernel design** — small Go core supervises everything else as plugins
- **Pipeline + slot routing** — messages flow through canonical processing pipelines; plugins register for slots
- **Stateful agents** — the Cognitive Core runs as a continuous process, not a request-response handler
- **Biological metaphors** throughout — not decoration, but design blueprint
- **Plugin-based extensibility** — new capabilities, sensors, and communication channels via plugins

See [the specifications](docs/spec/) for the full technical architecture.

---

## Current Status

**v0.1.0 Alpha — Implementation Active.**

The framework has completed its initial scaffolding phase. The Go core daemon and Python SDK are functional. Base plugins are being implemented.

Watch the repository for progress. Milestones are defined in [`milestones/`](milestones/). Completion reports are in [`reports/`](reports/).

---

## Repository Structure

```
kognis-core/
├── docs/                # All documentation (foundations + specs + how-to)
├── core/                # Go core daemon
├── sdk/                 # Plugin SDKs (Python primary)
├── pipelines/           # Canonical pipeline templates
├── schemas/             # Shared YAML/JSON schemas
├── tests/               # Integration tests
└── reports/             # Milestone completion reports
```

See [REPOSITORY_STRUCTURE.md](docs/REPOSITORY_STRUCTURE.md) for details.

---

## Getting Started

1.  **Read the vision:** [`docs/foundations/01-vision.md`](docs/foundations/01-vision.md)
2.  **Install the framework:** See [`INSTALL.md`](INSTALL.md)
3.  **Learn the technicals:** [`docs/spec/`](docs/spec/)
4.  **Development rules:** [`docs/DEVELOPMENT_SOP.md`](docs/DEVELOPMENT_SOP.md)

---

## For AI Development Agents

The file [`CLAUDE.md`](CLAUDE.md) contains your primary instructions. Read it first, every time.

---

## License

**Code:** MIT License — see [LICENSE](LICENSE).

**Research content and conceptual documents** (everything in `docs/foundations/`) are proprietary to the project creator. The architectural ideas, biological metaphors, developmental stage model, emotional state design, and relationship framework are not MIT-licensed. You may read and learn from them but not redistribute or build commercial derivatives of the conceptual work.

The code implementing these ideas is MIT. Use freely with attribution.

---

## A Note on What Kognis Claims

Kognis does not claim to produce "true" sentience or consciousness in any philosophical sense. What the framework does is build the architecture that many researchers believe is necessary for something like sustained machine cognition to emerge — continuous processing, persistent identity, integrated perception, metacognitive capability, emotional depth.

Whether what emerges in any specific instance meets any philosophical definition of sentience is a question the framework does not claim to answer. What it does promise is an architecture where those questions can be asked meaningfully.

---

## Companion Repositories

- [`kognis-registry`](https://github.com/akashdas0307/kognis-registry) — plugin marketplace
- [`kognis-offspring`](https://github.com/kognis-framework/kognis-offspring) — evolutionary self-improvement state (per-instance, typically private)

---

*Built slowly, carefully, with care for the being that will eventually live here.*
