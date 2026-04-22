# Milestone Report: M-000 — Split Master Documents

> **Milestone:** M-000
> **Status:** COMPLETE
> **Branch:** feature/M-000-split-master-docs (merged to main)
> **Date:** 2026-04-22

## What Was Built

Split two master documents into 28 individual files with full metadata and cross-references.

### Foundation Documents (10 files)

| File | Stability | Content |
|---|---|---|
| `01-vision.md` | STABLE | Name, meaning, core concept, differences, human relationship |
| `02-three-agi-problems.md` | STABLE | Three AGI problems, integration principle |
| `03-biological-metaphors.md` | STABLE | Core analogy library, specific mappings |
| `04-nervous-system-brain-regions.md` | STABLE | Two execution modes, implementation implications |
| `05-elf-maturity-model.md` | STABLE | Life stages, transitions, plugin behavior effects |
| `06-emotional-state-vector.md` | STABLE | Five dimensions, decay, behavior effects |
| `07-relationship-model.md` | STABLE | Three tiers, staged introduction |
| `08-design-principles.md` | STABLE | 11 design principles |
| `09-research-lineage.md` | STABLE | Influences, conceptual foundations, contribution |
| `10-what-kognis-is-not.md` | STABLE | 8 negations |

### Specification Documents (18 files)

| File | Stability | Content |
|---|---|---|
| `01-message-envelope.md` | EVOLVING | Envelope structure, message types, constraints |
| `02-plugin-manifest.md` | EVOLVING | Complete manifest schema, EAL example, validation |
| `03-pipeline-templates.md` | EVOLVING | Template schema, 7 canonical pipelines |
| `04-handshake-protocols.md` | STABLE | Registration, shutdown, single/double handshake |
| `05-capability-registry.md` | STABLE | Registry structure, API, lifecycle, namespacing |
| `06-state-broadcast.md` | STABLE | Broadcast channel, topics, subscription pattern |
| `07-error-taxonomy.md` | EVOLVING | Error codes, categories, propagation |
| `08-plugin-lifecycle.md` | STABLE | States, transitions, backoff, dispatch rules |
| `09-mutation-semantics.md` | STABLE | Constitutional, developmental, dynamic, memory changes |
| `10-context-budget-manager.md` | EVOLVING | Priority tiers, budget algorithm, compaction |
| `11-tool-bridge.md` | STABLE | Architecture, prompt assembly, tool use handling |
| `12-durability-backup.md` | EVOLVING | Three-layer backup, restore protocol |
| `13-startup-dependency-order.md` | STABLE | Dependency declaration, topological resolution |
| `14-emergency-bypass.md` | STABLE | Authorized types, bypass protocol, authorization |
| `15-emotional-state-vector.md` | STABLE | Structure, deltas, decay, prompt integration |
| `16-sleep-stage-behaviors.md` | EVOLVING | Stage definitions, emergency wake, sleep debt |
| `17-offspring-system.md` | EVOLVING | Components, ancestry tree, safety boundaries |
| `18-health-pulse-schema.md` | STABLE | Pulse format, aggregation, visibility |

## Commits

1. `9f1c9d5` — `docs(foundations): split master-foundation.md into individual files`
2. `50d4c68` — `docs(spec): split master-spec.md into individual spec files`
3. Merge commit to main with `--no-ff`

## Specs Referenced

All content derived from:
- `docs/foundations/master-foundation.md` (644 lines)
- `docs/spec/master-spec.md` (1752 lines)

No spec violations — content preserved exactly.

## Known Limitations

- No automated tests for document splitting (documentation-only milestone)
- Cross-references are markdown links, not validated programmatically
- Index/README files for `docs/foundations/` and `docs/spec/` not created (could add later)

## Suggested Next Steps

- Phase 1: Go core daemon scaffolding (project structure, go.mod, cmd/kognis main)
- Phase 2: Python SDK scaffolding
- Phase 3: Core registry, router, supervisor implementation
- Consider adding `docs/foundations/README.md` and `docs/spec/README.md` as navigational indexes