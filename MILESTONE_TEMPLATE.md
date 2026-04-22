# Milestone Template

> **Purpose:** Standard format for defining work packages delegated to AI development agents
> **Usage:** Copy this template, fill in specifics, save as `milestones/M-<id>-<slug>.md` before invoking team command

---

## Milestone Metadata

- **Milestone ID:** M-XXX (sequential number)
- **Milestone Name:** Short descriptive name
- **Priority:** [P0 critical | P1 high | P2 normal | P3 low]
- **Estimated Effort:** [S: hours | M: days | L: week | XL: multi-week]
- **Type:** [soft (auto-merge eligible) | hard (human review required)]
- **Dependencies:** Other milestones that must complete first
- **Target Branch:** `feature/M-XXX-slug`

---

## 1. Goal

*One clear sentence describing what "done" means. If you can't write this in one sentence, the milestone is too big — split it.*

Example: "Implement the plugin manifest parser that validates plugin.yaml against schema v1 and produces a typed Manifest struct."

---

## 2. Context Files to Read First

MANDATORY reading before starting work:

- `docs/foundations/<relevant-file>.md` — [why it's relevant]
- `docs/spec/<relevant-spec>.md` — [sections to focus on]
- Related code files (if extending existing work): [list]
- Previous milestone reports (if building on them): [list]

---

## 3. Deliverables

Specific artifacts to produce:

- [ ] **File:** `path/to/implementation.ext` implementing [what]
- [ ] **File:** `path/to/tests.ext` testing [what]
- [ ] **Doc update:** `path/to/doc.md` reflecting [changes]
- [ ] **Example:** Update `docs/YAML_EXAMPLES.md` with [new examples]
- [ ] **Changelog entry:** Add to CHANGELOG.md

---

## 4. Constraints — Non-Negotiable

- **Language:** Go / Python / other
- **Must integrate with:** [existing components]
- **Must NOT change:** [files/areas outside scope]
- **Must follow patterns from:** [existing code or spec section]
- **Dependencies allowed:** [specific list or "standard library only"]
- **Performance target:** [if any — e.g., "registration handshake < 500ms p99"]

---

## 5. Success Criteria — How We Verify

A check against each of these:

- [ ] All specified unit tests pass
- [ ] Integration test demonstrates [behavior]
- [ ] Lint clean (Go vet + golangci-lint OR ruff + mypy)
- [ ] Documentation reflects changes accurately
- [ ] No architectural changes
- [ ] Scope respected (no feature creep)
- [ ] Performance target met (if applicable)
- [ ] Commit messages follow format
- [ ] [Milestone-specific check]

---

## 6. Review Gates

Points where you MUST pause and either:
- Auto-commit if this is a soft milestone AND all criteria met
- Request human review if hard milestone OR any criterion fails

- [ ] **Gate 1:** After spec reading, confirm understanding (implicit if no questions)
- [ ] **Gate 2:** After test writing, run failing tests (confirm they fail correctly)
- [ ] **Gate 3:** After implementation, run passing tests
- [ ] **Gate 4:** Before merge (human review if hard milestone)

---

## 7. What to Flag — Must Ask Before Proceeding

Do NOT proceed silently if you encounter any of these. Flag them in a comment or pause for human input:

- Ambiguity in the spec that materially affects design
- Conflict between spec and existing code (which is right?)
- Temptation to deviate from spec (flag the deviation reason, don't just do it)
- Missing dependency you believe is needed
- Performance concerns that suggest architectural change
- Scope creep temptation (should this be a separate milestone?)

---

## 8. Out of Scope

Explicit list of things NOT to do in this milestone. Prevents creep:

- ❌ Do not also implement [related thing]
- ❌ Do not refactor [adjacent code]
- ❌ Do not add [bonus feature]
- ❌ Do not optimize [performance aspect beyond target]

Record these in `docs/FUTURE_WORK.md` if they're valid improvements for later.

---

## 9. Completion Report Template

At milestone end, produce `reports/milestone-M-XXX.md`:

```markdown
# Milestone M-XXX Report

## Status: [COMPLETE | PARTIAL | BLOCKED]

## Summary
[One paragraph]

## Deliverables Checklist
- [x] / [ ] per deliverable from section 3

## Test Results
- Unit: X passed, Y failed
- Integration: X passed, Y failed
- Coverage: Z%

## Code Metrics
- Files changed: N
- Lines added: +X -Y
- Commits: K

## Spec References
[Which specs were implemented]

## Known Limitations
[What doesn't work, what was deferred]

## Spec Clarifications Needed
[Questions for creator]

## Next Milestone Ready?
[Yes/no, what enables it]
```

---

## Example Filled-In Milestone

```markdown
# Milestone M-001: Plugin Manifest Parser

## Metadata
- Milestone ID: M-001
- Priority: P0
- Estimated Effort: M (days)
- Type: soft
- Dependencies: None (first milestone)
- Target Branch: feature/M-001-manifest-parser

## Goal
Implement a Python library that parses plugin.yaml files, validates them against the manifest v1 schema, and produces typed Manifest dataclass objects suitable for plugin registration.

## Context Files to Read First
- docs/spec/02-plugin-manifest.md (complete spec)
- docs/foundations/01-vision.md (understand what plugins are for)
- docs/YAML_EXAMPLES.md (example manifests)

## Deliverables
- [ ] sdk/python/kognis_sdk/manifest.py with Manifest dataclass and parser
- [ ] sdk/python/tests/test_manifest_parser.py (min 25 test cases)
- [ ] Updated docs/YAML_EXAMPLES.md with validated examples
- [ ] CHANGELOG entry

## Constraints
- Python 3.11+
- Dependencies: pyyaml, dataclasses only
- Must validate against manifest_version: 1
- Must produce specific error codes per docs/spec/07-error-taxonomy.md

## Success Criteria
- [ ] 25+ unit tests covering valid/invalid manifests
- [ ] All spec fields represented in Manifest dataclass
- [ ] All error cases (KGN-MANIFEST-*) raised appropriately
- [ ] Parse time < 10ms for typical manifest
- [ ] mypy strict mode passes
- [ ] Documentation includes usage example

## Review Gates
- [ ] After reading spec: confirm all fields understood
- [ ] Before merge: verify all error codes match taxonomy

## What to Flag
- Ambiguity about optional vs required fields
- Edge cases in permission declarations
- Backward-compat concerns

## Out of Scope
- ❌ Do not implement plugin loading yet
- ❌ Do not implement capability registration yet
- ❌ Do not build CLI around this
- ❌ Do not add JSON support (YAML only for now)
```

---

## Tips for Writing Good Milestones

1. **One goal, not many.** If your goal contains "and" twice, split.
2. **Be specific about success.** "It works" is not a success criterion.
3. **List out-of-scope explicitly.** Prevents scope creep.
4. **Reference specs precisely.** Include section numbers.
5. **Estimate realistically.** Add buffer for spec clarifications.
6. **Right-size soft vs hard.** When in doubt, mark as hard.
