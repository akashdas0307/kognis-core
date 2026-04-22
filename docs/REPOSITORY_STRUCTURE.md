# Repository Structure — kognis-core

> **Purpose:** Define the directory layout of the kognis-core repository
> **Stability:** STABLE
> **Audience:** AI development agents and human contributors

---

## 1. Top-Level Layout

```
kognis-core/
├── CLAUDE.md                         # Primary agent instructions
├── README.md                         # Public-facing overview
├── LICENSE                           # MIT License for code
├── CHANGELOG.md                      # Append-only change log
├── CODE_OF_CONDUCT.md                # Contributor conduct
├── CONTRIBUTING.md                   # How to contribute
├── SECURITY.md                       # Security reporting
├── .gitignore                        # Standard ignores
├── .gitattributes                    # Line ending and binary handling
├── .editorconfig                     # Editor consistency
├── .claude/                          # Claude Code + OMC configuration
│   ├── omc.jsonc                     # OMC team routing config
│   └── settings.json                 # Project-level Claude settings
├── .github/                          # GitHub-specific config
│   ├── workflows/                    # CI/CD pipelines
│   ├── ISSUE_TEMPLATE/
│   └── PULL_REQUEST_TEMPLATE.md
├── docs/                             # All documentation
├── core/                             # Go core daemon (created during development)
├── sdk/                              # Plugin SDKs (created during development)
├── pipelines/                        # Canonical pipeline templates (YAML)
├── schemas/                          # Shared JSON/YAML schemas
├── tests/                            # Cross-component integration tests
├── scripts/                          # Development and tooling scripts
├── examples/                         # Example plugins for reference
├── milestones/                       # Milestone definition documents
└── reports/                          # Milestone completion reports
```

---

## 2. `docs/` Directory

The heart of the project. All documentation organized by purpose.

```
docs/
├── foundations/                      # Conceptual foundations (research content - proprietary)
│   ├── master-foundation.md          # Consolidated (will be split)
│   ├── 01-vision.md                  # After split
│   ├── 02-three-agi-problems.md
│   ├── 03-biological-metaphors.md
│   ├── 04-nervous-system-brain-regions.md
│   ├── 05-elf-maturity-model.md
│   ├── 06-emotional-state-vector.md
│   ├── 07-relationship-model.md
│   ├── 08-design-principles.md
│   ├── 09-research-lineage.md
│   └── 10-what-kognis-is-not.md
│
├── spec/                             # Technical specifications (MIT with code)
│   ├── master-spec.md                # Consolidated (will be split)
│   ├── 01-message-envelope.md        # After split
│   ├── 02-plugin-manifest.md
│   ├── 03-pipeline-templates.md
│   ├── 04-handshake-protocols.md
│   ├── 05-capability-registry.md
│   ├── 06-state-broadcast.md
│   ├── 07-error-taxonomy.md
│   ├── 08-plugin-lifecycle.md
│   ├── 09-mutation-semantics.md
│   ├── 10-context-budget-manager.md
│   ├── 11-tool-bridge.md
│   ├── 12-durability-backup.md
│   ├── 13-startup-dependency-order.md
│   ├── 14-emergency-bypass.md
│   ├── 15-emotional-state-vector.md
│   ├── 16-sleep-stage-behaviors.md
│   ├── 17-offspring-system.md
│   └── 18-health-pulse-schema.md
│
├── DEVELOPMENT_SOP.md                # Standard operating procedures
├── MILESTONE_TEMPLATE.md             # Work package template
├── GLOSSARY.md                       # Terminology reference
├── REPOSITORY_STRUCTURE.md           # This file
├── YAML_EXAMPLES.md                  # Canonical YAML examples
├── COMMANDS.md                       # Kognis CLI commands (evolves)
├── FUTURE_WORK.md                    # Out-of-scope improvements list
├── ARCHITECTURE_DIAGRAMS.md          # Mermaid/ASCII architecture
└── TROUBLESHOOTING.md                # Common problems and solutions
```

### Why This Split?

| Folder | Content Type | License | Change Frequency |
|---|---|---|---|
| `foundations/` | Research, philosophy, rationale | PROPRIETARY | Rare (concepts are stable) |
| `spec/` | Technical contracts, schemas | MIT (with code) | Moderate (versioned changes) |
| Others | How-to, reference | MIT (with code) | Frequent |

**Rule:** Never blur the license line. Foundations are research; specs are engineering.

---

## 3. `core/` Directory — The Go Daemon

Structure to be created during development. Expected layout:

```
core/
├── cmd/
│   └── kognis/                       # Main entry point
│       └── main.go
├── internal/                         # Not exported — internal to core
│   ├── registry/                     # Plugin registry
│   │   ├── plugin_registry.go
│   │   ├── capability_registry.go
│   │   └── tests/
│   ├── router/                       # Message router + dispatch tables
│   │   ├── router.go
│   │   ├── dispatch_table.go
│   │   ├── pipeline_loader.go
│   │   └── tests/
│   ├── supervisor/                   # Plugin process lifecycle
│   │   ├── supervisor.go
│   │   ├── backoff.go
│   │   └── tests/
│   ├── eventbus/                     # NATS embedded
│   │   ├── eventbus.go
│   │   └── tests/
│   ├── controlplane/                 # gRPC server for plugins
│   │   ├── server.go
│   │   ├── handshake.go
│   │   └── tests/
│   ├── health/                       # Health aggregation
│   │   ├── aggregator.go
│   │   ├── pulse_store.go
│   │   └── tests/
│   ├── envelope/                     # Message envelope handling
│   │   ├── envelope.go
│   │   └── tests/
│   ├── tui/                          # Dashboard TUI
│   │   ├── dashboard.go
│   │   ├── panels/
│   │   └── tests/
│   └── config/                       # Configuration loading
│       ├── config.go
│       └── tests/
├── pkg/                              # Exportable Go packages
│   ├── protocol/                     # Protocol Buffers definitions
│   └── schema/                       # Shared schemas
├── go.mod
├── go.sum
├── Makefile                          # Build/test commands
└── README.md                         # Core daemon overview
```

---

## 4. `sdk/` Directory — Plugin SDKs

```
sdk/
├── python/                           # Primary SDK (Python 3.11+)
│   ├── kognis_sdk/
│   │   ├── __init__.py
│   │   ├── manifest.py               # Manifest parsing
│   │   ├── plugin.py                 # Plugin base classes
│   │   ├── stateful_agent.py         # StatefulAgent base class
│   │   ├── handshake.py              # Registration + shutdown
│   │   ├── envelope.py               # Envelope construction
│   │   ├── eventbus.py               # NATS client wrapper
│   │   ├── control_plane.py          # gRPC client
│   │   ├── capability.py             # Capability query + registration
│   │   ├── tool_bridge.py            # LLM tool translation
│   │   ├── context_budget.py         # Context Budget Manager
│   │   ├── health.py                 # Health pulse emitter
│   │   ├── state_broadcast.py        # Semantic state pub/sub
│   │   ├── logger.py                 # Structured logging
│   │   └── testing/                  # Test harnesses for plugin authors
│   ├── tests/
│   ├── examples/                     # Example plugin implementations
│   │   ├── hello_world/              # Minimal stateless plugin
│   │   ├── echo_chat/                # Simple chat plugin
│   │   └── hello_agent/              # Minimal stateful agent
│   ├── pyproject.toml
│   ├── README.md
│   └── CHANGELOG.md
│
└── (future: go/, node/ SDKs)
```

---

## 5. `pipelines/` Directory — Canonical Templates

```
pipelines/
├── user_text_interaction.yaml
├── user_voice_interaction.yaml
├── background_monitoring.yaml
├── autonomous_cognition.yaml
├── sleep_consolidation.yaml
├── health_management.yaml
├── offspring_evaluation.yaml
└── README.md                         # Explains how pipelines work
```

These ship with the framework. Plugins register for slots in these pipelines.

---

## 6. `schemas/` Directory — Shared Schemas

```
schemas/
├── manifest-v1.yaml                  # Plugin manifest schema
├── envelope-v1.yaml                  # Message envelope schema
├── pipeline-template-v1.yaml         # Pipeline template schema
├── health-pulse-v1.yaml              # Health pulse schema
├── state-broadcast-v1.yaml           # State broadcast schema
├── registry-entry-v1.yaml            # Registry entry schema (used by kognis-registry)
└── README.md
```

---

## 7. `tests/` Directory — Integration Tests

```
tests/
├── integration/
│   ├── plugin_lifecycle/             # Full registration → shutdown tests
│   ├── pipeline_dispatch/            # Multi-plugin pipeline flows
│   ├── capability_query/             # Double handshake tests
│   ├── error_handling/               # Error propagation tests
│   └── emergency_bypass/             # Bypass channel tests
├── e2e/                              # End-to-end scenarios
│   ├── single_input_response/        # Minimal chat flow
│   ├── idle_daydream/                # Continuous cognition
│   └── sleep_wake_cycle/             # Sleep stage transitions
├── fixtures/                         # Test data, mock plugins
└── conftest.py                       # Python test configuration
```

---

## 8. `scripts/` Directory — Development Tooling

```
scripts/
├── bootstrap.sh                      # Initial dev environment setup
├── lint.sh                           # Run all linters
├── test.sh                           # Run all tests
├── release.sh                        # Release preparation
├── plugin-scaffold.sh                # Generate plugin skeleton
├── validate-manifest.py              # Validate a plugin.yaml
└── check-spec-alignment.py           # Verify specs match code
```

---

## 9. `examples/` Directory — Reference Plugins

Small, educational plugins demonstrating patterns:

```
examples/
├── minimal_stateless/                # Smallest possible plugin
├── minimal_stateful/                 # Smallest stateful agent
├── echo_chat_tui/                    # Simple chat terminal UI
├── dummy_memory/                     # Example memory plugin
└── README.md                         # Guide to examples
```

These are NOT production plugins — they live in `kognis-registry/official/`.

---

## 10. `milestones/` Directory — Work Package Definitions

```
milestones/
├── M-000-split-master-docs.md        # First milestone (split masters)
├── M-001-manifest-parser.md
├── M-002-envelope-schema.md
├── ...
└── README.md                         # How milestones work
```

Milestones are the unit of AI agent work delegation.

---

## 11. `reports/` Directory — Completion Reports

```
reports/
├── milestone-M-000.md                # After completion
├── milestone-M-001.md
├── ...
```

Each completed milestone produces a report here.

---

## 12. File Naming Conventions

| Context | Convention | Example |
|---|---|---|
| Markdown docs | `Title_Case.md` or `kebab-case.md` | `DEVELOPMENT_SOP.md`, `master-spec.md` |
| Numbered specs | `NN-topic-name.md` | `01-message-envelope.md` |
| Milestones | `M-NNN-slug.md` | `M-001-manifest-parser.md` |
| Go files | `snake_case.go` | `dispatch_table.go` |
| Python files | `snake_case.py` | `tool_bridge.py` |
| YAML schemas | `topic-vN.yaml` | `manifest-v1.yaml` |
| YAML templates | `pipeline_name.yaml` | `user_text_interaction.yaml` |
| Test files | `test_<subject>.py` / `<subject>_test.go` | `test_manifest.py` |

---

## 13. Branch Naming

| Branch Type | Pattern | Example |
|---|---|---|
| Feature | `feature/M-XXX-slug` | `feature/M-001-manifest-parser` |
| Bug fix | `fix/<short-desc>` | `fix/router-race-condition` |
| Spec revision | `spec/<spec-name>` | `spec/revise-handshake-timeout` |
| Documentation | `docs/<scope>` | `docs/add-plugin-examples` |
| Refactor | `refactor/<scope>` | `refactor/registry-internals` |
| Experimental | `experiment/<description>` | `experiment/rust-sdk` |

---

## 14. What Goes Where — Decision Guide

When adding a new file, ask:

1. **Is it a concept or rationale?** → `docs/foundations/`
2. **Is it a technical contract?** → `docs/spec/`
3. **Is it developer how-to?** → `docs/` (flat)
4. **Is it Go daemon code?** → `core/`
5. **Is it Python SDK code?** → `sdk/python/`
6. **Is it a pipeline definition?** → `pipelines/`
7. **Is it a schema?** → `schemas/`
8. **Is it integration test?** → `tests/`
9. **Is it tooling?** → `scripts/`
10. **Is it a learning example?** → `examples/`
11. **Is it a work package?** → `milestones/`
12. **Is it a report?** → `reports/`

---

## 15. Files That Are NEVER Edited Directly By Agents

- `LICENSE`
- `CODE_OF_CONDUCT.md`
- `docs/foundations/*` (human-approved only)
- Security-related configurations
- Release tags and version files (`core/pkg/version.go` — human-released)

Agents modifying these files should flag and await human approval.

---

*This structure reflects the architectural separation of concerns. Maintain it rigorously.*
