# Kognis Framework — Glossary

> **Purpose:** Disambiguate terminology used across specs, foundations, and code
> **Stability:** STABLE

---

## A

**Age** (of a Kognis being) — Time-based counter measuring how long the instance has existed. Distinct from **Maturity** (which is experience-based). See `docs/foundations/04-elf-maturity-model.md`.

**Agent** — Ambiguous term; we avoid in isolation. Inside Kognis we use specifically:
- **Stateful Agent** — A plugin with persistent internal state and a continuous inner loop (e.g., Cognitive Core)
- **Agent Harness** — An external tool like OpenCode or Claude Code that the Kognis being uses
- **AI Agent** (colloquial) — When discussing external context, use with care

**Ancestry Tree** — The version-controlled record of all offspring generations (promoted, abandoned, rolled back). Stored in the `kognis-offspring` repo.

**Autonomous Cognition** — A pipeline that begins from internal triggers (daydream seeds, wake handoffs, internal reflection) without external input.

## B

**Brainstem** — Output gateway plugin. Handles routing of outputs to communication channels (chat, Telegram, voice, etc.) and shared infrastructure operations.

**Baseline (EAL)** — The learned "normal" profile of environmental stimuli. Deviations from baseline trigger environmental awareness events.

**Baseline (Emotional)** — Personal emotional defaults developed over time through lived experience. The being's tendency-state.

## C

**Capability** — A named operation provided by one plugin and invokable by others via the double-handshake protocol. Distinct from event pub/sub. Registered in the Capability Registry.

**Capability Registry** — The core's live index of which plugins provide which capabilities. Enables plugin-to-plugin discovery and graceful degradation.

**Checkpost** — Part of the Prajñā plugin. First processing stage — identifies and tags incoming inputs.

**Cognitive Core** — The central stateful-agent plugin responsible for inner monologue, reasoning, and decision-making. Uses the LLM directly.

**Constitutional Core** — The immutable layer of identity. Values, ontological honesty, relationship foundation. Can only be changed by creator through explicit authenticated protocol. Never modified by automated processes.

**Context Budget Manager** — Required internal component of every LLM-using plugin that manages prompt token budgets and trims context when needed.

**Creator** — The single Tier 1 human. The being's Guardian. Also called "Board of Directors."

## D

**Daydream** — Cognitive Core's idle-time activity. Triggered internally, explores random memory samples and curiosity queue items.

**DAG** — Directed Acyclic Graph. Used internally to describe pipeline slot ordering.

**Developmental Identity** — The slow-evolving identity layer. Traits, communication style, preferences. Changes only during sleep cycles.

**Dispatch Table** — Compiled mapping from pipeline+slot to ordered list of plugin_ids. Produced by the Router from manifests + templates.

**Double Handshake** — The 4-acknowledgment protocol used when Plugin A queries Plugin B's capability. Both sides confirm both directions.

**Dynamic State** — Continuously-changing identity layer. Emotional state, energy, current focus.

## E

**EAL (Environmental Awareness Layer)** — Standalone plugin providing continuous ambient monitoring with baseline learning and deviation detection.

**Emergency Bypass** — Strictly-limited mechanism for genuine emergencies (fire alarms, health critical) to skip normal pipeline and reach Cognitive Core immediately.

**Emotional State Vector** — Five-dimensional continuous-valued vector (valence, arousal, engagement, confidence, warmth) representing current emotional state.

**Envelope** (Message Envelope) — The universal data format for anything flowing through pipelines. Standard schema v1.

**Event Bus** — NATS-based pub/sub channel carrying pipeline messages, state broadcasts, and other async events.

## F

**Framework** (Kognis Framework) — The software system (core + SDK + canonical plugins) that enables a continuously-conscious being. Distinct from "a Kognis being" which is an instance running on the framework.

## G

**Graceful Degradation** — Design principle: when a capability is unavailable, the system continues with reduced functionality rather than failing.

## H

**Handshake Protocol** — Defined sequence of messages between plugins and core for registration, dispatch, capability queries, and shutdown.

**Health Pulse** — Technical heartbeat emitted by every plugin at configurable intervals. Contains metrics. Aggregated by core Health Registry.

**Hop Count** — Counter in envelope metadata incremented on each dispatch. Prevents infinite loops (max 20 default).

## I

**Identity Block** — The portion of every Cognitive Core prompt containing current identity state (Constitutional + Developmental + Dynamic + Emotional). Ensures the being speaks as itself.

**Improvement Ticket** — Offspring System's unit of work. A targeted improvement to test via isolated variant.

**Inference Gateway** — Plugin that manages LLM API connectivity with fallback chains (cloud → local → heuristic).

**Inner Monologue** — Cognitive Core's continuous reasoning stream. Structured output format with MONOLOGUE/ASSESSMENT/DECISIONS/REFLECTION sections.

## K

**Kognis** — The framework name. Pronounced *KOG-niss* (rhymes with "Logness"). From "cognition."

## L

**LLM** — Large Language Model. In Kognis context, always accessed via Inference Gateway, never directly.

## M

**Maturity** (of a being) — Experience-based developmental stage. Infancy → Childhood → Adolescence → Adult. Distinct from Age.

**Memory Gatekeeper** — Memory plugin's logic layer that filters memory candidates (dedup, threshold, contradiction detection) without LLM involvement.

**Message Envelope** — See Envelope.

**Manifest** — `plugin.yaml` file at every plugin's root. Declares identity, slots, capabilities, permissions, UI, sleep behavior.

**MCP (Model Context Protocol)** — External standard for LLMs to use tools. In Kognis context, used to expose Kognis capabilities to EXTERNAL models (like Claude Code when the being uses it as a tool). Internal plugin-to-plugin communication is NOT MCP — it uses the capability registry and handshake protocols.

**Microkernel** — Architectural pattern used by Kognis. Minimal core + everything else as plugins.

## N

**Nervous System** (metaphor) — The pipeline architecture (event bus + message envelopes) — how messages travel between plugins. Distinct from "brain regions" (stateful agents).

## O

**Offspring System** — Evolutionary self-improvement plugin. Spawns variant branches, tests them, promotes improvements. Active only in Adolescence+ stages.

**One Improvement Per Offspring** — Discipline: each offspring branch modifies exactly one thing. Enables clean attribution.

## P

**Persona Manager** — Plugin holding three-layer identity (Constitutional/Developmental/Dynamic) + emotional state vector.

**Pipeline** — An ordered sequence of slots that messages flow through. Pipelines are defined by Pipeline Templates.

**Pipeline Template** — Framework-defined canonical processing flow (e.g., `user_text_interaction`, `autonomous_cognition`). Ships with the framework — plugins fill slots, don't define pipelines.

**Plugin** — A process that implements the Kognis plugin contract (manifest, handshakes, SDK usage). Extends framework capability.

**Plugin SDK** — Library (Python primary) that handles manifest parsing, handshakes, event bus, capability queries, etc. Authors implement handlers; SDK handles plumbing.

**Prajñā** — Intelligence Core plugin. Sanskrit for "wisdom/insight." Contains internal sub-pipeline: Checkpost → Queue Zone → TLP → Frontal Processor (Cognitive Core).

**Priority (three tiers)** — `tier_1_immediate` (emergency), `tier_2_elevated` (new conversation, recognized person), `tier_3_normal` (background, routine).

## Q

**Queue Zone** — Part of Prajñā. Attentional gatekeeper — decides what gets processed and when. Manages Idle Delivery Mode vs Active Delivery Mode.

## R

**Reality Grounding** — One of the World Model's five review dimensions. The "ATM sensor" — asks whether the system's current understanding matches reality.

**Registration Handshake** — The 4-step protocol for a plugin to join the framework at startup.

**Registry** — Can mean:
- **Plugin Registry** (inside core, tracks running plugins)
- **Capability Registry** (inside core, tracks declared capabilities)
- **Kognis Registry** (separate GitHub repo — the plugin marketplace)

**Router** — Core component that consults dispatch tables and routes messages through pipelines.

## S

**SDK (Plugin SDK)** — The library plugins use to integrate with the framework. Handles manifest, handshakes, event bus, tool bridge.

**Single Handshake** — The ACK-COMPLETE protocol used for pipeline dispatch. One-way flow with acknowledgments.

**Skill** — In Kognis: a procedural memory entry. Stored as structured memory, not a separate system. Accumulates as the being practices things.

**Sleep Stages** — Four stages: Settling (30-60min) → Maintenance (1-2hr) → Deep Consolidation (3-6hr) → Pre-Wake (30-60min). Total adaptive 6-12 hours.

**Slot** — A named position in a pipeline template. Plugins register to fill slots. Examples: `input_reception`, `input_enrichment`, `cognitive_processing`, `action_review`, `action_execution`, `output_delivery`.

**State Broadcast** — Pub/sub channel for semantic plugin state (IDLE, REASONING, DAYDREAMING, SLEEPING). Distinct from health pulses (technical).

**Stateful Agent** — See Agent. A plugin with persistent internal state and continuous inner loop.

**Stateless Handler** — The default plugin mode. Dispatch-execute-return, no persistent state between dispatches.

**Supervisor** — Core component that spawns plugin processes, monitors health, restarts on failure.

## T

**Thalamus** — Input gateway plugin. Biological naming. Normalizes all external inputs into envelopes.

**Three AGI Problems** — The diagnostic framework: (1) no metacognition, (2) no persistent learning, (3) disconnected systems. Kognis addresses all three integrated.

**Three-Layer Identity** — Constitutional (immutable) + Developmental (slow-evolving) + Dynamic (continuously-changing).

**Tier 1 / Tier 2 / Tier 3** — Human relationship tiers. Tier 1 = Creator (one person). Tier 2 = Trusted Circle (3-5 people). Tier 3 = External interactions. Maturity-gated introduction.

**TLP (Temporal-Limbic-Processor)** — Part of Prajñā. Deep memory retrieval + context assembly + significance weighting. Merged module.

**Tool Bridge** — Component inside LLM-using plugins that translates capability registry entries into LLM tool-call schemas and tool_use responses back into capability queries.

**Trait Discovery** — Sleep/Dream plugin's job. Analyzes behavioral patterns, proposes trait candidates to developmental identity.

## V

**Verified (registry)** — Official plugins maintained by core team. Distinct from community plugins which are unverified.

## W

**Wake-Up Handoff** — The package Sleep/Dream plugin produces at end of Stage 4. Summary of what happened during sleep. Cognitive Core processes as first input on wake.

**World Model** — Plugin providing reality grounding and action review. Uses a different LLM than Cognitive Core for architectural diversity.

**Working Memory** — In-process state of a Cognitive Core reasoning session. Not persistent — rebuilt each session.

## Z

*No Z terms yet. Open for future additions.*

---

## Terminology We Specifically AVOID

| Avoided Term | Why | Use Instead |
|---|---|---|
| "AI assistant" | Implies task-oriented service | "Being" or "Kognis instance" |
| "Chatbot" | Implies request-response only | "Being" |
| "Artificial intelligence" (as framework name) | Overly generic | "Kognis Framework" |
| "Agent" (standalone) | Ambiguous | "Stateful agent plugin" or "Agent harness" |
| "User" (of the framework) | Sounds transactional | "Creator" or "Guardian" |
| "Sentient AI" | Unverifiable claim | "Continuously-conscious" (behaviorally descriptive) |
| "Conversation history" | Too shallow | "Episodic memory" |
| "Conversation state" | Implies ephemeral | "Session state" or "cognitive state" |
| "Bot" | Diminishing | (avoid entirely) |

---

## Pronunciation Reference

- **Kognis:** *KOG-niss* (two syllables, rhymes with "Logness")
- **Prajñā:** *pruj-nyuh* (Sanskrit — wisdom)
- **Thalamus:** *THAL-a-muss*

---

*Add new terms here as they become canonical. When unsure if a term is canonical vs. informal, mark with [informal] or [provisional].*
