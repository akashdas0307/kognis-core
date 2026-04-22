# Research Lineage

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-foundation.md
> **Related:** [01-vision.md](01-vision.md), [03-biological-metaphors.md](03-biological-metaphors.md), [02-three-agi-problems.md](02-three-agi-problems.md)

Kognis does not exist in isolation. It synthesizes insights from extensive prior research. Honoring this lineage is important.

## 9.1 Direct Influences

### From the Five Open-Source Projects Analyzed

- **Letta (MemGPT):** LLM-as-Operating-System paradigm. Tiered memory (Core/Recall/Archival). Sleep-time compute concept. Virtual context management. Informs Kognis's Memory plugin architecture.

- **MemPalace:** Raw verbatim memory with spatial organization. Finding that raw text outperforms LLM-extracted summaries for retrieval. Influences Kognis's memory storage approach.

- **Hermes Agent:** Self-improving skill files. Periodic self-evaluation. Procedural memory as SKILL.md files. Multi-platform gateway pattern. Informs Kognis's Skills (procedural memory) and Brainstem communication plugins.

- **Claw Code / OpenCode:** Agent harness engineering. Terminal-native design. MCP integration. These inform Kognis's tool-use approach — harnesses are tools the being uses, not what the being is.

- **Paperclip AI:** Heartbeat-driven scheduling. Organizational coordination metaphor. Budget enforcement. Identity files. Some of these concepts informed the core daemon design.

## 9.2 Conceptual Foundations

- **Yann LeCun's work on world models and JEPA:** The argument that scaling LLMs will not reach AGI without embodied experience and world models. Influences Kognis's emphasis on environmental awareness and grounded context.

- **Complementary learning systems theory (McClelland, Rumelhart):** Hippocampus + neocortex as fast and slow learning systems. Directly maps to Kognis's Memory plugin and Sleep/Dream consolidation.

- **Russell's circumplex model of affect:** Valence and arousal as foundational emotional dimensions. Extended by Kognis with engagement, confidence, warmth for relational-computational context.

- **Attachment theory (Bowlby, Ainsworth):** How primary caregivers shape identity formation in early development. Directly informs the single-creator Tier 1 design.

- **Evolutionary biology's directed mutation concept:** Informs Offspring System — deliberate mutation rather than random, with selection pressure.

## 9.3 Technical Patterns Borrowed

- **Microkernel architecture:** Emacs, Neovim, Docker, Kubernetes operators. Small stable core, everything else as plugins.
- **Homebrew-tap pattern:** Central registry pointing to external repositories. Informs the plugin registry design.
- **Middleware chain pattern:** Express.js, Django middleware, Kubernetes admission controllers. Informs pipeline slot architecture.
- **Circuit breaker pattern:** Netflix Hystrix, widely adopted. Informs System Health's fault isolation.

## 9.4 The Kognis Contribution

What Kognis contributes beyond synthesis:

- **The nervous-system + brain-region architectural pattern** for plugin frameworks that enables continuous consciousness
- **The Elf maturity model** — age vs. maturity as independent axes, with stable adult form
- **The five-dimensional emotional state vector** integrated into cognitive prompts
- **The three-tier human relationship model** with maturity-gated introduction
- **The integration of all three AGI problems into a single unified architecture** rather than solving them independently

These are genuinely novel architectural contributions, even if the constituent ideas have lineage.