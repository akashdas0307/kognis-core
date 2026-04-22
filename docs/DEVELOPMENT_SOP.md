# Development Standard Operating Procedures (SOP)

> **Purpose:** Define the disciplined workflow for developing the Kognis Framework
> **Audience:** AI development agents and the human creator
> **Stability:** STABLE

---

## 1. The Development Workflow in One Page

```
┌─────────────────────────────────────────────────────────────┐
│                    KOGNIS DEVELOPMENT WORKFLOW                 │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. READ       → Relevant spec(s) and foundations            │
│       ↓                                                      │
│  2. PLAN       → Milestone document (see template)           │
│       ↓                                                      │
│  3. TEST FIRST → Write tests against spec                    │
│       ↓                                                      │
│  4. IMPLEMENT  → Code to pass tests                          │
│       ↓                                                      │
│  5. VERIFY     → Tests pass, lint clean, docs updated        │
│       ↓                                                      │
│  6. COMMIT     → Small, atomic, well-messaged commits        │
│       ↓                                                      │
│  7. MILESTONE  → Soft: auto-merge ok | Hard: human review    │
│       ↓                                                      │
│  8. DOCUMENT   → Update CHANGELOG, relevant docs             │
│       ↓                                                      │
│  9. HANDOFF    → Next milestone or report                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Phases of a Work Package

A "work package" is one complete milestone from `docs/MILESTONE_TEMPLATE.md`. Each work package has three phases:

### Phase A — Comprehension (5-10% of effort)
- Read all referenced specs
- Read all referenced foundations
- Identify ambiguities or missing information
- If blockers exist, ask the creator BEFORE proceeding

### Phase B — Execution (70-80% of effort)
- Write tests first for each component
- Implement to pass tests
- Commit frequently (every logical chunk)
- Run full test suite before each commit
- Update documentation in parallel

### Phase C — Verification & Handoff (15-20% of effort)
- Run complete test suite
- Verify lint and formatting
- Review diff for scope creep
- Update CHANGELOG.md
- Produce milestone report (if final milestone)
- Open PR or auto-merge per rules

---

## 3. Test-First Discipline

For ANY new component, this order is mandatory:

```
1. Read spec that describes component
2. Write unit tests covering all spec requirements
3. Run tests → they fail (no implementation yet) — GOOD
4. Implement minimum code to pass tests
5. Run tests → they pass
6. Refactor for clarity while tests still pass
7. Add edge case tests
8. Implement edge case handling
9. Integration tests
10. Commit
```

**Why:** Writing tests after code leads to tests that match the (possibly wrong) implementation. Writing tests from spec ensures tests match the intent.

**Exception:** Infrastructure code (build scripts, CI configs) where TDD adds little value. Use judgment.

---

## 4. Commit Standards

### 4.1 Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:** `feat`, `fix`, `docs`, `spec`, `refactor`, `test`, `chore`, `perf`, `style`

**Scope:** Component area (e.g., `core`, `sdk`, `router`, `registry`, `foundations`, `spec-01`)

**Subject:** Imperative, present tense, lowercase, no period. Max 72 chars.

**Body:** Optional. Wrap at 80. Explain WHY, not WHAT.

**Footer:** Optional. References issues, milestones, breaking changes.

### 4.2 Examples

```
feat(router): implement dispatch table compiler

Builds per-pipeline dispatch tables from plugin slot registrations
sorted by priority. Recomputes on plugin registration/deregistration.

Spec reference: docs/spec/03-pipeline-templates.md
Milestone: M2-core-routing
```

```
fix(sdk/python): handle empty capability registry during Tool Bridge assembly

Previously raised AttributeError when no capabilities registered.
Now returns empty tools list gracefully.

Fixes: observed during Milestone 4 integration testing
```

```
spec(handshake): clarify timeout semantics for double handshake

The original spec was ambiguous about whether ACK timeout starts
from QUERY_DISPATCH or from initial QUERY. Confirmed: from QUERY_DISPATCH.

Spec change approved by creator: 2026-04-22
```

### 4.3 Atomic Commits

One commit = one logical change. Do NOT bundle:
- Feature + test in separate commits if test drives feature
- Documentation updates into feature commit (usually)
- Multiple bug fixes into one commit

Exception: When a change spans multiple files but is conceptually one thing (e.g., renaming a function used in 5 places), one commit is correct.

### 4.4 Git History Hygiene

- NEVER force-push to main
- NEVER rewrite commits that are on main
- Feature branches: rebasing onto main is OK before merge
- Prefer merge commits for milestones (preserves parallel history)
- Prefer squash merges for small PRs (cleaner main history)

---

## 5. Branching Strategy

```
main                        ← stable, always deployable
  ↑
develop (optional)          ← integration branch if multiple concurrent milestones
  ↑
feature/M3-memory-plugin    ← milestone work
fix/router-race-condition   ← bug fixes
spec/revise-handshake        ← spec revisions
docs/add-plugin-examples    ← docs work
```

### Rules

- **main** receives only: merges from feature branches passing all gates
- **feature branches** named: `feature/<milestone-id>-<short-desc>` (lowercase-hyphenated)
- **Delete branches** after merge
- **Long-lived branches are forbidden** — milestone branches should close within days, not weeks

---

## 6. Soft vs Hard Milestone Distinction

### Soft Milestones (Auto-Merge Allowed)

Qualifications (ALL must be true):
- ✅ Completes a well-defined piece of work
- ✅ All tests pass
- ✅ No changes to: `docs/foundations/`, `docs/spec/` (except formatting)
- ✅ No new dependencies
- ✅ No tech stack changes
- ✅ No architectural pattern changes
- ✅ Commit messages clean
- ✅ Documentation updated

Auto-merge protocol:
1. Ensure feature branch up-to-date with main
2. Run full CI
3. Merge to main with merge commit
4. Delete feature branch
5. Update CHANGELOG.md on main

### Hard Milestones (Human Review Required)

Anything that is:
- End of a multi-milestone work package
- Touches architectural code (core daemon, SDK interface, event bus)
- Introduces new dependencies
- Changes protocol versions
- Modifies specs or foundations
- Is the LAST milestone of a phase

Protocol:
1. Push feature branch
2. Open PR with milestone-report.md
3. Tag creator
4. DO NOT merge until approved

---

## 7. Milestone Reporting Format

Each milestone produces a report in `reports/milestone-<id>.md`:

```markdown
# Milestone Report: <Milestone ID> — <Name>

## Status: [COMPLETE | PARTIAL | BLOCKED]

## Summary
[One paragraph of what was accomplished]

## Deliverables
- [x] File: path/to/file.ext — description
- [x] Test: path/to/test.ext — description
- [x] Doc updated: path/to/doc.md
- [ ] Not done: thing that was skipped and why

## Specs Referenced
- docs/spec/XX-name.md (sections used)
- docs/foundations/XX-name.md (concepts applied)

## Tests Added
- N unit tests in path/to/tests
- M integration tests

## Performance Notes
- Latency measured: Xms p50, Yms p99
- Memory usage: ~Z MB steady state

## Known Limitations
- Thing that doesn't handle edge case X
- Optimization deferred to future

## Spec Clarifications Needed
- Question about spec NN section M.P
- Proposed resolution: ...

## Suggested Next Steps
- Milestone X would follow naturally
- Optimization Y would benefit

## Metrics
- Lines added: N
- Lines removed: M
- Commits: K
- Tests: T
- Coverage: C%
```

---

## 8. Testing Discipline

### 8.1 Test Categories

| Category | Purpose | Speed | When Run |
|---|---|---|---|
| Unit | One function/class in isolation | <10ms each | On every save, pre-commit |
| Integration | Multiple components together | <1s each | Pre-push |
| Conformance | Plugin matches SDK contract | Varies | Pre-merge |
| End-to-end | Full pipeline flow | <30s each | Pre-merge |
| Property | Invariants hold across inputs | Varies | Nightly |

### 8.2 Coverage Expectations

- Unit tests: aim for 80%+ line coverage on core/SDK code
- All public API surfaces: 100% of paths tested
- Error paths: tested, not just happy paths
- Integration: every pipeline slot has integration test

### 8.3 Test Quality

Tests should:
- Name reveal intent: `test_router_rejects_envelope_when_hop_count_exceeded()`, not `test_router_2()`
- Be independent (no shared state between tests)
- Be deterministic (no flaky tests — fix or delete)
- Be fast (slow tests get skipped, then bugs slip through)

---

## 9. Documentation Requirements

Every code change has a documentation counterpart:

| Change Type | Doc Requirement |
|---|---|
| New public function | Docstring with args, returns, raises, example |
| New plugin capability | Entry in capability catalog doc + example YAML |
| New message type | Entry in message envelope spec |
| New error code | Entry in error taxonomy spec |
| Spec behavior change | Spec file updated with rationale |
| Breaking change | CHANGELOG entry + migration note |
| New CLI command | Entry in docs/COMMANDS.md |

---

## 10. Quality Gates

Before a commit can merge to main:

```
Gate 1: Tests pass
  └─ unit: pass
  └─ integration: pass
  └─ conformance: pass (if applicable)

Gate 2: Lint clean
  └─ Go: go vet, golangci-lint
  └─ Python: ruff, mypy
  └─ YAML: yamllint
  └─ Markdown: markdownlint

Gate 3: Spec alignment
  └─ No silent spec deviations
  └─ Comments cite specs where appropriate

Gate 4: Documentation
  └─ CHANGELOG updated
  └─ Public API documented
  └─ Example YAML updated if relevant

Gate 5: Scope
  └─ Change within milestone scope
  └─ No silent feature additions
```

---

## 11. When Things Go Wrong

### 11.1 Test Failure
1. Do NOT disable the test
2. Do NOT modify test to match broken code
3. Investigate root cause
4. Fix code OR flag if spec appears wrong

### 11.2 Spec Ambiguity
1. Stop implementing
2. Document the ambiguity
3. Propose specific clarification
4. Ask creator to resolve
5. Resume only after resolution

### 11.3 Dependency Hell
1. Do NOT just upgrade everything
2. Identify specific conflict
3. Prefer compatible versions
4. If none exist: flag to creator

### 11.4 Performance Problems
1. Measure first (don't guess)
2. Establish baseline
3. Identify hot path
4. Flag if solution requires architecture change

### 11.5 "This Feels Wrong" Moments
If during implementation you feel the spec is wrong:
1. Complete implementation per spec (don't deviate)
2. Document the concern in milestone report
3. Propose specific spec change
4. Creator decides whether to revise spec

---

## 12. Collaborative Conduct

When multiple agents work in parallel:

- Use different feature branches
- Avoid modifying the same file simultaneously
- Use `git worktree` for parallel isolation (OMC supports this)
- Communicate through commit messages and PR discussions
- Merge to main in coordinated waves, not chaotically

---

## 13. The Boring Discipline That Saves Months

These practices feel slow but prevent exponential future pain:

1. **Read specs fully before coding** — 10 extra minutes prevents days of rework
2. **Write the test first** — clarifies what success means
3. **Small commits** — easier to revert, understand, and bisect
4. **Update docs immediately** — nothing rots faster than deferred docs
5. **Resist scope creep** — the "while I'm here" addition breaks timelines
6. **Ask when uncertain** — guessing costs more than asking

These are not bureaucracy. They are how serious projects get built reliably.

---

*When in doubt, err on the side of discipline.*
