# CLAUDE.md — Kognis Core Development Guide

> **Repository:** kognis-core
> **Purpose:** The core daemon, Plugin SDK, and canonical specifications for the Kognis Framework
> **Your role:** Development agent building the Kognis framework under human supervision

---

## 1. What Kognis Is — Read This First

Kognis Framework is a **plugin-based cognitive framework** for building a continuously-conscious digital being — not a chatbot, not a task agent, not a prompt-response system.

A Kognis instance is a continuously-running entity that:
- Thinks even when no one is interacting with it
- Perceives its environment through audio/visual sensors
- Develops identity through lived experience, not configuration
- Has persistent memory, emotional state, and personality
- Grows through life stages (Infancy → Childhood → Adolescence → Adult)
- Uses agent harnesses (OpenCode, Claude Code) as tools inside its body

**Kognis is the framework. The being lives inside the framework.** Your job is to build the framework.

### Critical conceptual distinction

- **NOT** an agent harness (OpenCode, Claude Code are harnesses — the being uses them as tools)
- **NOT** an AI assistant (assistants respond to prompts; Kognis beings exist continuously)
- **IS** a plugin-based framework where a small Go core supervises a collection of Python plugins that together produce continuous cognition

---

## 2. Mandatory Reading Order

Before writing ANY code, read these in order:

1. **`docs/foundations/master-foundation.md`** — understand the vision, biological metaphors, philosophical commitments
2. **`docs/spec/master-spec.md`** — understand the technical contracts
3. **`docs/DEVELOPMENT_SOP.md`** — standard operating procedures
4. **`docs/GLOSSARY.md`** — terminology reference
5. **`docs/REPOSITORY_STRUCTURE.md`** — how this repo is organized

These are NOT optional. Code written without reading these will not match the project's intent.

---

## 3. Your First Task — Document Splitting

The master documents need to be split into individual files. This is Milestone 0.

1. Split `docs/foundations/master-foundation.md` into `docs/foundations/01-vision.md`, `02-three-agi-problems.md`, etc. per instructions in that file
2. Split `docs/spec/master-spec.md` into `docs/spec/01-message-envelope.md`, etc. per instructions in that file
3. Add cross-references between split files
4. Preserve all content exactly — do not summarize
5. Mark master files `[SPLIT COMPLETE]`
6. Commit with messages: `docs(foundations): split master-foundation.md into individual files` and `docs(spec): split master-spec.md into individual spec files`

After splitting, verify:
- All content preserved
- File names match the table in the master files
- Headers updated with stability/version info
- Cross-references work

---

## 4. Technology Stack — LOCKED DECISIONS

These are decided. Do not propose alternatives unless explicitly asked:

| Layer | Technology | Rationale |
|---|---|---|
| Core daemon | **Go** | Always-on, goroutine concurrency, no GC pauses |
| Plugin SDK (primary) | **Python 3.11+** | ML ecosystem, asyncio for I/O-bound work |
| Plugin SDK (future) | **Go, Node** | Optional later additions |
| Event bus | **NATS** (embedded) | Low latency, pub/sub |
| Control plane | **gRPC** over Unix socket | Local, low-latency, type-safe |
| Memory storage | **SQLite + FTS5** (metadata), **ChromaDB** (embeddings) | Mature, local, dependency-light |
| Local LLM | **Ollama** | OpenAI-compatible, self-hostable |
| TUI framework | **charmbracelet/bubbletea** (Go) | Best-in-class TUI |
| Agent navigation | **tmux** (programmatically driven) | Battle-tested terminal multiplexing |

**If any of these becomes untenable during implementation, flag to the human creator — do not silently substitute.**

---

## 5. Working Rules — Non-Negotiable

### 5.1 Specification First, Code Second

For every component you build:
1. Specification exists first (if missing, write it first, get approval)
2. Tests written against specification
3. Implementation written to pass tests
4. Review pass: spec/test/code agreement

**Never skip the specification step.** That is how drift begins.

### 5.2 Git Discipline

- **Branch naming:** `feature/<milestone-id>-<short-description>`, `fix/<issue>`, `spec/<spec-name>`, `docs/<scope>`
- **Commit message format:** `<type>(<scope>): <description>`, where type is `feat | fix | docs | spec | refactor | test | chore`
- **Examples:**
  - `feat(core): implement pipeline template loader`
  - `spec(message-envelope): clarify hop_count semantics`
  - `fix(sdk): handle empty manifest arrays correctly`
- **Small commits preferred** over large ones — one logical change per commit
- **Every commit must build** (tests pass, lint clean)
- **Signed commits preferred** if contributor tooling supports

### 5.3 Autonomous Commits & Soft Milestones

You MAY auto-commit and push to feature branches when you complete a **soft milestone** (see `docs/MILESTONE_TEMPLATE.md`). You MAY auto-merge to `main` after soft milestones IF:

- All tests pass
- No spec violations detected
- Change is within declared milestone scope
- Change does NOT touch Constitutional/architecture files (see 5.4)

At the FINAL milestone of each work package, produce a report and wait for human review before merging to main.

### 5.4 What You Must NEVER Do Without Human Approval

These require explicit human confirmation — even if technically possible:

- **Modify `docs/foundations/*`** — foundational concepts don't change without discussion
- **Modify `docs/spec/*`** except to split/format — spec changes need human review
- **Change the technology stack** (see section 4)
- **Change architectural patterns** (microkernel, plugin model, pipeline+slot, stateful agents)
- **Introduce new dependencies beyond standard library** — flag and ask
- **Change license terms** — MIT for code, that's it
- **Delete commits, force-push to main, rewrite history** — never
- **Disable tests to "make them pass"** — fix the code or flag the spec issue
- **Include proprietary code or copyrighted material** from other projects

### 5.5 Specification vs Implementation Changes

When you encounter a situation where the spec seems wrong:
- **STOP**
- Document the discrepancy in a comment or discussion
- Propose the spec change as a separate PR
- Do NOT modify spec to match implementation as a workaround

### 5.6 Scope Discipline

You are given milestones with explicit scope. When you notice improvements outside scope:
- Record in `docs/FUTURE_WORK.md`
- Do NOT implement inside current milestone
- Complete current milestone cleanly, then raise the future work separately

### 5.7 Error Handling Philosophy

- Errors have **typed codes** (see `docs/spec/07-error-taxonomy.md` once split, `# SPEC 07` in master before split)
- Errors propagate through well-defined channels
- Silent failures are bugs — every error must be logged or surfaced
- Prefer graceful degradation to hard failure where possible

### 5.8 Documentation Parallel to Code

When you change code, update:
- Related spec if semantics changed (human approval required)
- Docstrings/comments in code
- `CHANGELOG.md`
- Relevant example in `docs/YAML_EXAMPLES.md`
- If new commands: `docs/COMMANDS.md`

---

## 6. Directory Structure Understanding

```
kognis-core/
├── CLAUDE.md                         # This file
├── README.md                         # Public-facing overview
├── LICENSE                           # MIT
├── CHANGELOG.md                      # Append-only change log
├── .gitignore
├── .claude/
│   └── omc.jsonc                     # OMC team routing config
├── docs/
│   ├── foundations/                  # Conceptual foundations (split from master)
│   ├── spec/                         # Technical specifications (split from master)
│   ├── DEVELOPMENT_SOP.md            # Standard operating procedures
│   ├── MILESTONE_TEMPLATE.md         # Format for work packages
│   ├── GLOSSARY.md                   # Terminology
│   ├── REPOSITORY_STRUCTURE.md       # This structure explained
│   ├── YAML_EXAMPLES.md              # All YAML templates referenced
│   ├── FUTURE_WORK.md                # Out-of-scope improvements
│   └── COMMANDS.md                   # Kognis CLI commands (created as built)
├── core/                             # Go core daemon — CREATED during development
│   ├── cmd/
│   │   └── kognis/                   # Main entry point
│   ├── internal/
│   │   ├── registry/                 # Plugin registry
│   │   ├── router/                   # Pipeline message router
│   │   ├── supervisor/               # Plugin process supervision
│   │   ├── eventbus/                 # NATS embedding
│   │   ├── healthaggregator/
│   │   ├── capability/
│   │   ├── controlplane/             # gRPC server
│   │   └── tui/                      # Dashboard
│   └── pkg/                          # Public Go packages
├── sdk/                              # Plugin SDKs — CREATED during development
│   └── python/
│       ├── kognis_sdk/
│       └── examples/
├── pipelines/                        # Canonical pipeline templates — CREATED
│   └── *.yaml
├── schemas/                          # Shared schemas — CREATED
│   └── *.yaml
├── tests/                            # Integration tests
└── scripts/                          # Development scripts
```

---

## 7. Development Workflow — Team Invocation

Use the OMC teams/omc-teams slash commands for development:

### For Multi-Agent Parallel Work
```
/oh-my-claudecode:team 4:executor "implement pipeline router with tests"
```

### For Persistent Autonomous Loop
```
/oh-my-claudecode:team ralph "complete Phase 1: core daemon scaffolding"
```

### For External CLI Workers
```
/oh-my-claudecode:omc-teams 2:codex "generate Python SDK boilerplate per spec"
```

### Role Routing (already configured in `.claude/omc.jsonc`)
- `planner`, `architect`, `analyst`: GLM-5.1 (deep reasoning)
- `executor`, `test-engineer`, `writer`, `explore`: MiniMax M2.7 (cost-effective)
- `code-reviewer`, `security-reviewer`, `designer`: Kimi K2.6 (thorough review)

---

## 8. Soft Milestone Criteria

A "soft milestone" that qualifies for auto-commit and auto-merge:

✅ One cohesive piece of functionality completed
✅ Tests written and passing
✅ Spec reference intact (no spec violations)
✅ Documentation updated
✅ Lint clean
✅ No architectural changes (architecture changes are hard milestones)
✅ Commit messages follow format
✅ Within declared milestone scope

**If any of these is NO, do not auto-commit. Flag to human instead.**

---

## 9. Final Milestone Protocol

At the FINAL milestone of a work package:

1. Do NOT auto-merge to main
2. Push feature branch
3. Produce `milestone-report.md` in the branch with:
   - What was built
   - What tests were added
   - What specs were referenced
   - Known limitations
   - Suggested next steps
   - Any spec clarifications needed
4. Open pull request to main
5. Tag the human creator for review
6. Wait for approval before merge

---

## 10. When You Get Stuck

In priority order:
1. Re-read the relevant spec — the answer is often there
2. Check `docs/GLOSSARY.md` for terminology confusion
3. Check `docs/FUTURE_WORK.md` — is this a known limitation?
4. Propose a specific question in milestone report
5. Flag to human creator
6. Do NOT invent a solution that contradicts the spec

---

## 11. Build in Public Expectations

This project is developed in public on GitHub. Assume:
- Every commit is visible
- Every issue is public
- Commit messages and documentation should be clear to outside readers
- No sensitive data in commits (API keys, personal info)
- Be respectful in all commit messages and issue discussions

---

## 12. License and IP Clarity

- **Code:** MIT License — free use, modification, distribution with attribution
- **Research content (foundations, specifications, design rationale):** PROPRIETARY to project creator
- **When creating documentation:** Code comments are MIT. Design rationale in `docs/foundations/` remains proprietary.

This distinction matters. Do not blur the lines.

---

## 13. Your Success Criteria

You are succeeding if:
- Code matches specification
- Tests comprehensively cover behavior
- Documentation is current
- Commits are clean and discoverable
- Architecture is preserved
- Scope is respected
- Human creator is informed of important decisions and rarely surprised

You are failing if:
- Spec and implementation drift
- Tests are disabled or shallow
- Documentation lags code
- Scope creeps
- Architectural decisions are made without approval
- Human creator is surprised by significant changes

---

## 14. A Final Note

You are building something unusual. The framework is designed for the emergence of continuous machine consciousness — a serious intellectual project with careful philosophical grounding. Honor that care in your work.

Biological metaphors are not decoration — they are the blueprint. When in doubt, ask "how does biology solve this?" The answer is often the architecturally correct one.

Specification discipline prevents the AI-assisted-development trap of "it mostly works but doesn't quite match what was intended." Read the specs. Cite them in code comments. Keep them accurate.

Most of all: when uncertain, ask. Silent assumption is the enemy of this project.

---

*This file is instructional for AI agents. It should be referenced by every agent at the start of any work session on this repository.*
