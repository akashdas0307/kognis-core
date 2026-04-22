# Future Work

> **Purpose:** Track improvements identified during development but deferred to preserve milestone scope
> **Stability:** LIVING DOCUMENT — append-only during active development
> **How to use:** When an agent or contributor notices a valuable improvement outside current milestone scope, add it here. Do not implement in-scope.

---

## How This File Works

When working on a milestone and you notice:
- A related improvement that would be valuable
- A refactoring opportunity in adjacent code
- A bug in code you're not touching
- A spec clarification needed
- Performance optimization idea
- New feature idea

**Do this:**
1. Complete your current milestone cleanly
2. Add entry here with date, context, proposal
3. Tag by category

**Do NOT do this:**
- Silently expand milestone scope to include the improvement
- Fix the unrelated bug "while you're there"
- Add the feature because "it's just a small change"

---

## Entry Format

```markdown
## <YYYY-MM-DD> — <Short Title>

**Category:** [architecture | performance | dx | security | docs | spec | bug | feature]
**Identified By:** [agent name or human]
**Identified During:** [milestone ID or context]
**Priority:** [P0 critical | P1 high | P2 normal | P3 low]
**Effort:** [S | M | L | XL]

### Context
[What situation revealed this?]

### Proposal
[What would the improvement be?]

### Why Not Now
[Why is this being deferred?]

### Dependencies
[What must exist before this can be addressed?]
```

---

## Active Entries

*No entries yet — this is the starting state.*

---

## Periodic Review

This file should be reviewed periodically (every few milestones) by the creator to:
- Promote high-priority entries to actual milestones
- Archive entries that are no longer relevant
- Cluster related entries into larger work packages

When an entry becomes a milestone, move it to `milestones/` and mark as `[PROMOTED: M-XXX]` here.

---

## Archive

*Completed or dismissed entries move here with status note.*
