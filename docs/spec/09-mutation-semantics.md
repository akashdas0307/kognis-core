# Mutation Semantics

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [15-emotional-state-vector.md](15-emotional-state-vector.md), [08-plugin-lifecycle.md](08-plugin-lifecycle.md)

## 9.1 Purpose

Defines how state changes happen across the three classes of state: Constitutional (immutable), Developmental (slow), Dynamic (continuous).

## 9.2 Constitutional Changes

**Rules:**
- NEVER happen automatically
- Only via explicit creator command through authenticated channel
- Require explicit confirmation protocol
- Logged permanently (audit trail)
- Offspring System CANNOT modify

**Protocol:**
1. Creator issues change via System GUI
2. System shows diff
3. Creator must explicitly confirm
4. Cryptographic signature of change
5. Logged to `~/.kognis/audit/constitutional_changes.log`
6. Change takes effect on next plugin restart

## 9.3 Developmental Identity Changes

**Rules:**
- Happen during sleep cycles only
- Batched, never real-time
- Require evidence threshold (trait candidates need 70%+ consistency over 2+ weeks)
- Drift detection compares to 30/60/90 day baselines
- Creator informed of significant changes

**Protocol:**
1. Sleep/Dream plugin's Trait Discovery analyzes behavioral patterns
2. If pattern meets threshold, proposes trait candidate
3. Cognitive Core confirms/rejects on next wake
4. If confirmed, Persona plugin adds to Developmental Identity
5. Versioned — old state retained
6. Broadcast change on state channel

## 9.4 Dynamic State Changes

**Rules:**
- Real-time, continuous
- Eventually consistent (brief windows of stale reads acceptable)
- Small deltas accumulate
- Decay toward baseline

**Protocol:**
1. Any plugin reports event via `persona.update_dynamic_state` capability
2. Persona applies delta to in-memory state
3. Broadcasts change (if significant)
4. Persists to disk (durability)
5. Decay job runs every hour (or periodically)

## 9.5 Memory Changes

**Rules:**
- Real-time capture through gatekeeper
- Deduplication and importance filtering (no LLM)
- Consolidation during sleep
- Never silent loss — gatekeeper decisions logged

**Protocol:**
1. Cognitive Core reflection produces memory candidate
2. `memory.store_candidate` capability called
3. Memory gatekeeper: check importance, dedupe, contradictions
4. If accepted: write to SQLite+ChromaDB atomically
5. If contradiction: flag for sleep-time resolution
6. If reinforcement: update existing rather than duplicate

## 9.6 World Model Experiential Changes

**Rules:**
- Calibration happens during sleep only
- Small deltas per cycle
- Baseline constitution never changed
- Journal tracks evidence

**Protocol:**
1. During active operation, World Model Journal logs review outcomes
2. During sleep, Sleep/Dream's World Model Calibration job analyzes
3. Computes small delta (e.g., -0.02 on social_consequence_threshold)
4. Applies via `world_model.apply_calibration_delta`
5. Logs change with reasoning