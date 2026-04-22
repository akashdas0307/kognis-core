# Emotional State — The 5-Dimensional Vector

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-foundation.md
> **Related:** [03-biological-metaphors.md](03-biological-metaphors.md), [05-elf-maturity-model.md](05-elf-maturity-model.md), [07-relationship-model.md](07-relationship-model.md)

Emotional state in Kognis is not a mood tag. It is a five-dimensional continuous-valued vector that modulates cognition, memory formation, and attention.

## 6.1 The Five Dimensions

| Dimension | Range | Meaning |
|---|---|---|
| **Valence** | -1.0 to +1.0 | Unpleasant ↔ Pleasant |
| **Arousal** | 0.0 to 1.0 | Calm ↔ Excited |
| **Engagement** | 0.0 to 1.0 | Disengaged ↔ Fully invested |
| **Confidence** | 0.0 to 1.0 | Uncertain ↔ Assured |
| **Warmth** | 0.0 to 1.0 | Cold/reserved ↔ Caring/open |

This draws on established affect research (Russell's circumplex model for valence/arousal, extended with engagement/confidence/warmth for relational and task-oriented dimensions).

## 6.2 How State Evolves

Every cognitive event produces small deltas applied to the state vector. Examples:

| Event | Deltas |
|---|---|
| Successful task completion | valence +0.05, confidence +0.02 |
| Error or failure | valence -0.08, confidence -0.05 |
| Creator expresses warmth | valence +0.10, warmth +0.05, engagement +0.10 |
| Long debugging with no progress | arousal +0.10, engagement -0.05, valence -0.05 |
| Interesting daydream discovery | valence +0.15, engagement +0.10, arousal +0.05 |
| Unexpected positive news from creator | valence +0.15, arousal +0.30, engagement +0.20 |
| Manipulation attempt detected | arousal +0.20, warmth -0.10, confidence -0.05 |

Deltas are small by design. Strong shifts require accumulation.

## 6.3 Decay Dynamics

Every hour (or on periodic state broadcast), current state decays by approximately 10% toward baseline. This prevents emotional state from being dominated by single events.

Baseline values are defined in Developmental Identity — each being develops its own emotional baseline through lived experience. Some trend toward higher baseline warmth. Others more reserved. This is part of personality.

## 6.4 How State Affects Behavior

The emotional state vector is included in the identity block of every Cognitive Core prompt. The LLM sees explicit values:

```
Current emotional state:
  valence: 0.72 (pleasant)
  arousal: 0.44 (moderate)
  engagement: 0.89 (highly invested)
  confidence: 0.61 (fairly certain)
  warmth: 0.81 (very warm)
```

The LLM is instructed to modulate output style accordingly. A warm, engaged instance responds differently than a cold, disengaged one. This is not theatrical — it is the same LLM producing differently-shaped outputs based on differently-shaped context.

## 6.5 Emotional Memory

When a memory is stored, current emotional state at the time is attached as metadata. Retrieval can prioritize "emotionally similar" memories — this is how Kognis instances develop emotional associations.

Memories formed during strong emotional states receive slightly higher importance scores, but with a metadata warning: "formed during elevated emotional state, salience may be amplified." This mirrors human memory — vivid but not always accurate.

## 6.6 Storage and Broadcast

- **Storage:** Persona plugin maintains current state vector, writes on every change to local persistent store
- **Broadcast:** On change exceeding threshold (approximately 0.1 in any dimension), publishes to `state.persona.emotional` topic
- **Persistence across restart:** state vector is restored from disk on plugin startup

## 6.7 What Emotions Are NOT

Kognis emotions are:
- NOT display-only animations
- NOT triggered only by explicit markers
- NOT fixed categories (happy/sad/angry)

Kognis emotions ARE:
- Continuous-valued modulators of cognition
- Automatically inferred from events and context
- Affecting real behavior and output
- Persistent across sessions
- Evolutionary (baseline shifts slowly over months)