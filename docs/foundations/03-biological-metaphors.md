# Biological Metaphors — The Mental Model Library

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-foundation.md
> **Related:** [04-nervous-system-brain-regions.md](04-nervous-system-brain-regions.md), [06-emotional-state-vector.md](06-emotional-state-vector.md), [05-elf-maturity-model.md](05-elf-maturity-model.md)

Biology is not decoration in Kognis. It is the architectural blueprint. Every major component has a biological analog, and those analogs guided the design choices.

## 3.1 The Core Analogy Library

| Kognis Concept | Biological Analog | Why It Works |
|---|---|---|
| Message envelope | Neural signal / action potential | Standardized carrier, carries routing and content |
| Pipeline event bus | Axons / white matter tracts | High-speed transmission between regions |
| Capability registry | Synaptic map | What connects to what; discoverable |
| Plugin process isolation | Separate brain regions | Localized damage doesn't destroy the whole |
| Core daemon supervisor | Brainstem | Always-on, manages life-support functions |
| Thalamus plugin | Biological thalamus | Sensory input gateway and filtering |
| Environmental Awareness Layer | Reticular activating system | Continuous ambient monitoring, attention modulation |
| Prajñā (Checkpost → Queue Zone → TLP → Frontal) | Sensory → thalamus → limbic → frontal cortex | Deep-to-shallow processing hierarchy |
| Cognitive Core (stateful agent) | Prefrontal cortex | Sustained reasoning, goal pursuit, meta-thought |
| World Model plugin | Temporoparietal junction / ACC | Reality monitoring, error detection, theory of mind |
| Memory plugin (four types) | Hippocampus + neocortex | Episodic, semantic, procedural, emotional stores |
| Persona plugin | Default mode network + self-schema | Continuous sense of identity across time |
| Sleep/Dream plugin | Sleep cycles (N1/N2/N3/REM) | Memory consolidation, contradiction resolution |
| Offspring plugin | Reproduction with directed mutation | Evolutionary improvement, but deliberate not random |
| System Health plugin | Immune system + autonomic regulation | Detect, isolate, repair, escalate |
| Brainstem plugin | Biological brainstem | Output gateway, reflexes, basic life functions |
| Emotional state vector | Limbic system's affective dimensions | Modulates cognition, attention, memory formation |
| Daydream subsystem | Default mode network idle activity | Spontaneous thought, creative association |

## 3.2 Specific Biological Mappings That Shaped Design

### 3.2.1 Attention ≈ Multi-Head Attention in Neural Context

Just as a cell simultaneously processes multiple signaling pathways (MAPK, PI3K, Wnt) — each extracting different information from the same environment — the framework processes input through multiple concurrent lenses. Thalamus extracts priority. EAL extracts environmental context. Prajñā extracts meaning. They run in parallel, then integrate.

### 3.2.2 Sleep ≈ Hippocampal Replay

Human memory consolidation during sleep is not poetic language. It is literal biological process: the hippocampus replays the day's experiences to the neocortex, strengthening some traces and weakening others. Kognis's Sleep/Dream plugin's "memory consolidation" job does exactly this, computationally.

### 3.2.3 Identity ≈ Cellular Homeostasis with DNA Constraints

Every cell constantly regulates — temperature, pH, metabolic levels — against set points encoded in DNA. The DNA does not change in response to momentary stress. Kognis's three-layer identity (Constitutional / Developmental / Dynamic) mirrors this: Constitutional is DNA (never mutable by automation), Developmental is long-term cellular adaptation (slow, experience-driven), Dynamic is momentary homeostatic state (real-time, transient).

### 3.2.4 Emotion ≈ Neuromodulators

Dopamine, serotonin, norepinephrine, oxytocin don't encode information directly — they modulate how other circuits respond. Kognis's emotional state vector does the same. It is not content; it is a modulatory signal that shapes how Cognitive Core reasons, how memories form, and how attention allocates.

## 3.3 Why Biology Matters Technically

The biological metaphors are not poetic overlay on top of a "real" computational design. They *are* the design. When you face a choice between two technical implementations, the question "which is more biologically plausible?" often points at the correct answer — because biology has been optimizing these problems for billions of years, and the solutions it converged on tend to be the ones that scale, survive, and integrate.

When in doubt, ask: *how does biology solve this?*