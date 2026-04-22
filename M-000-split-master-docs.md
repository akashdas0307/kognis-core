# Milestone M-000: Split Master Foundation and Spec Documents

## Milestone Metadata

- **Milestone ID:** M-000
- **Milestone Name:** Split Master Documents
- **Priority:** P0 (critical — blocks all subsequent work)
- **Estimated Effort:** S (hours)
- **Type:** soft (auto-merge eligible)
- **Dependencies:** None (this is the first milestone)
- **Target Branch:** `feature/M-000-split-master-docs`

---

## 1. Goal

Split the consolidated master documents `docs/foundations/master-foundation.md` and `docs/spec/master-spec.md` into individual files per their embedded instructions, preserving all content exactly and adding cross-references between files.

---

## 2. Context Files to Read First

MANDATORY before starting:

- `docs/foundations/master-foundation.md` — contains the content to split AND instructions for splitting
- `docs/spec/master-spec.md` — contains the content to split AND instructions for splitting
- `docs/REPOSITORY_STRUCTURE.md` — confirms target directory structure
- `docs/DEVELOPMENT_SOP.md` — for commit standards
- `CLAUDE.md` — for overall working rules

---

## 3. Deliverables

### 3.1 Foundations split into 10 files

Create these files under `docs/foundations/` by splitting `master-foundation.md`:

- [ ] `01-vision.md` — from PART 01
- [ ] `02-three-agi-problems.md` — from PART 02
- [ ] `03-biological-metaphors.md` — from PART 03
- [ ] `04-nervous-system-brain-regions.md` — from PART 04
- [ ] `05-elf-maturity-model.md` — from PART 05
- [ ] `06-emotional-state-vector.md` — from PART 06
- [ ] `07-relationship-model.md` — from PART 07
- [ ] `08-design-principles.md` — from PART 08
- [ ] `09-research-lineage.md` — from PART 09
- [ ] `10-what-kognis-is-not.md` — from PART 10

### 3.2 Spec split into 18 files

Create these files under `docs/spec/` by splitting `master-spec.md`:

- [ ] `01-message-envelope.md` — from SPEC 01
- [ ] `02-plugin-manifest.md` — from SPEC 02
- [ ] `03-pipeline-templates.md` — from SPEC 03
- [ ] `04-handshake-protocols.md` — from SPEC 04
- [ ] `05-capability-registry.md` — from SPEC 05
- [ ] `06-state-broadcast.md` — from SPEC 06
- [ ] `07-error-taxonomy.md` — from SPEC 07
- [ ] `08-plugin-lifecycle.md` — from SPEC 08
- [ ] `09-mutation-semantics.md` — from SPEC 09
- [ ] `10-context-budget-manager.md` — from SPEC 10
- [ ] `11-tool-bridge.md` — from SPEC 11
- [ ] `12-durability-backup.md` — from SPEC 12
- [ ] `13-startup-dependency-order.md` — from SPEC 13
- [ ] `14-emergency-bypass.md` — from SPEC 14
- [ ] `15-emotional-state-vector.md` — from SPEC 15
- [ ] `16-sleep-stage-behaviors.md` — from SPEC 16
- [ ] `17-offspring-system.md` — from SPEC 17
- [ ] `18-health-pulse-schema.md` — from SPEC 18

### 3.3 Master files marked as split

Both `master-foundation.md` and `master-spec.md` should remain in place but have `[SPLIT COMPLETE]` marker at the top (as indicated in their closing notes).

### 3.4 Cross-references added

Each split file should include:
- Header block: name, stability level, version, related files
- Cross-references to related files where natural (e.g., spec 04 references spec 05 when discussing capability registry handshakes)

### 3.5 Index files (optional but recommended)

Consider creating:
- `docs/foundations/README.md` — index of all foundations files in reading order
- `docs/spec/README.md` — index of all spec files with stability levels

### 3.6 CHANGELOG entry

Add entry to `CHANGELOG.md`:

```
## [Unreleased]
### Changed
- Split master-foundation.md into 10 individual foundation files
- Split master-spec.md into 18 individual specification files
- Added cross-references between related files
```

---

## 4. Constraints — Non-Negotiable

- **Preserve content EXACTLY** — no summarization, no rewording, no "cleaning up"
- **Use the file names from the split plan tables** in each master document
- **Follow numbering convention:** `NN-topic.md` with leading zero for sort order
- **Each file starts with:**
  ```markdown
  # <Title>

  > **Stability:** <level>
  > **Version:** 0.1.0
  > **Source:** Split from master-<type>.md
  > **Related:** [link to related files]
  ```
- **No restructuring** of content within sections — preserve hierarchy as-is

---

## 5. Success Criteria

- [ ] All 10 foundation files created with correct content
- [ ] All 18 spec files created with correct content
- [ ] Master files marked `[SPLIT COMPLETE]`
- [ ] Cross-references added where they improve readability
- [ ] CHANGELOG updated
- [ ] Commit messages follow format: `docs(foundations): split master-foundation.md` and `docs(spec): split master-spec.md`
- [ ] No content lost or altered from masters
- [ ] File names exactly match the split plan tables

---

## 6. Review Gates

- [ ] **Gate 1:** After reading master files, confirm understanding of the split plan
- [ ] **Gate 2:** After creating all files, run diff verification (total line count preserved minus any added headers)
- [ ] **Gate 3:** Auto-commit if all criteria met (this is a soft milestone)

---

## 7. What to Flag

- If a section is ambiguous about which file it belongs to
- If split plan tables in master documents conflict with each other
- If you notice content that should be in a different file than where it is
- If the cross-reference pattern is unclear

---

## 8. Out of Scope — Do NOT Do These

- ❌ Do not edit content in any way beyond adding headers
- ❌ Do not introduce new sections
- ❌ Do not consolidate sections across files
- ❌ Do not remove content even if it seems redundant
- ❌ Do not update master files beyond the `[SPLIT COMPLETE]` marker
- ❌ Do not start other milestones before this one is merged

---

## 9. Completion Report Format

At the end, produce `reports/milestone-M-000.md`:

```markdown
# Milestone M-000 Report: Split Master Documents

## Status: COMPLETE

## Summary
Split master-foundation.md into 10 files and master-spec.md into 18 files per embedded instructions. Content preserved exactly. Cross-references added.

## Deliverables
- [x] docs/foundations/*.md (10 files)
- [x] docs/spec/*.md (18 files)
- [x] Master files marked [SPLIT COMPLETE]
- [x] CHANGELOG updated

## Metrics
- Files created: 28
- Lines preserved: <N> (same as source)
- Cross-references added: <M>
- Commits: 2

## Ready for next milestone
Yes — M-001 (Plugin Manifest Parser) can now proceed.
```

---

*This milestone is the gateway to all future work. Complete it carefully.*
