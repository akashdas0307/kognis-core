# Changelog

All notable changes to the Kognis Framework will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-04-22

### Added

- Python SDK core implementation (Phase 4, M-001 through M-012):
  - M-001: Manifest parser (`kognis_sdk/manifest.py`) — SPEC 02 compliance with from_yaml/from_dict, validation
  - M-002: Envelope handling (`kognis_sdk/envelope.py`) — SPEC 01 compliance with immutable pattern, derive, validation
  - M-003: Control plane client (`kognis_sdk/control_plane.py`) — SPEC 04 handshake protocols, registration, dispatch, heartbeat
  - M-004: Event bus client (`kognis_sdk/eventbus.py`) — NATS pub/sub wrapper, topic helpers
  - M-005: Plugin base class (`kognis_sdk/plugin.py`) — SPEC 02/08 stateless handler mode, lifecycle hooks
  - M-006: Stateful agent (`kognis_sdk/stateful_agent.py`) — SPEC 02/08 continuous cognition loop, memory compaction
  - M-007: Capability registry client (`kognis_sdk/capability.py`) — SPEC 05 double handshake queries, LLM tool listing
  - M-008: Tool bridge (`kognis_sdk/tool_bridge.py`) — SPEC 11 LLM tool-call translation layer
  - M-009: Context budget manager (`kognis_sdk/context_budget.py`) — SPEC 10 priority tiers, trimming, adaptive feedback
  - M-010: Health pulse emitter (`kognis_sdk/health.py`) — SPEC 18 periodic pulses; SPEC 06 state broadcaster
  - M-011: State store (`kognis_sdk/state_store.py`) — SPEC 12 three-layer durability with crash recovery
  - M-012: Testing harness (`kognis_sdk/testing/`) — TestCore fixture for plugin unit testing
- Full SDK public API exports via `kognis_sdk/__init__.py`
- 204 unit tests across 6 test files + 59 integration tests (263 total, all passing)

## [0.2.1] - 2026-04-22

### Added

- Python SDK scaffolding: `sdk/python/pyproject.toml`, `kognis_sdk/__init__.py`, `testing/__init__.py`
- `CONTRIBUTING.md` — contributor guidelines with workflow, branch naming, commit format
- `SECURITY.md` — vulnerability reporting policy and security architecture principles
- `.github/workflows/validate-docs.yaml` — CI for markdown linting, spec cross-references, YAML validation
- `.github/workflows/lint.yaml` — CI for Go vet/golangci-lint, Python ruff/mypy, YAML yamllint
- `.github/PULL_REQUEST_TEMPLATE.md` — standardized PR template
- `docs/YAML_EXAMPLES.md` — canonical YAML templates for manifest, envelope, pipeline, health pulse, state broadcast, registry entry
- `docs/FUTURE_WORK.md` — out-of-scope improvements tracking
- `docs/COMMANDS.md` — CLI command documentation (planned commands placeholder)

### Changed

- Fixed Go module path: `github.com/akashdas0307/kognis-core/core` → `github.com/kognis-framework/kognis-core/core`

## [0.2.0] - 2026-04-22

### Added

- Go core daemon scaffolding with 9 internal packages and 2 public packages
  - `core/cmd/kognis/main.go` — entry point with signal handling
  - `core/internal/config` — configuration loading with defaults
  - `core/internal/registry` — thread-safe plugin registry with lifecycle states
  - `core/internal/router` — pipeline template loading and slot-based dispatch
  - `core/internal/supervisor` — plugin lifecycle management with exponential backoff
  - `core/internal/eventbus` — embedded NATS server wrapper
  - `core/internal/controlplane` — gRPC server over Unix socket + handshake protocol
  - `core/internal/health` — health pulse aggregation
  - `core/internal/envelope` — universal message envelope (SPEC 01)
  - `core/internal/tui` — dashboard stub
  - `core/pkg/protocol` — NATS subject constants and message types
  - `core/pkg/schema` — capability IDs, pipeline names, state constants
  - `core/Makefile` — build, test, lint targets
  - Unit tests for registry, router, envelope, config, backoff, handshake

## [0.1.0] - 2026-04-22

### Added

- Split `docs/foundations/master-foundation.md` into 10 individual files:
  - `01-vision.md`, `02-three-agi-problems.md`, `03-biological-metaphors.md`,
  - `04-nervous-system-brain-regions.md`, `05-elf-maturity-model.md`,
  - `06-emotional-state-vector.md`, `07-relationship-model.md`,
  - `08-design-principles.md`, `09-research-lineage.md`, `10-what-kognis-is-not.md`

- Split `docs/spec/master-spec.md` into 18 individual files:
  - `01-message-envelope.md`, `02-plugin-manifest.md`, `03-pipeline-templates.md`,
  - `04-handshake-protocols.md`, `05-capability-registry.md`, `06-state-broadcast.md`,
  - `07-error-taxonomy.md`, `08-plugin-lifecycle.md`, `09-mutation-semantics.md`,
  - `10-context-budget-manager.md`, `11-tool-bridge.md`, `12-durability-backup.md`,
  - `13-startup-dependency-order.md`, `14-emergency-bypass.md`,
  - `15-emotional-state-vector.md`, `16-sleep-stage-behaviors.md`,
  - `17-offspring-system.md`, `18-health-pulse-schema.md`

### Changed

- Marked `docs/foundations/master-foundation.md` as `[SPLIT COMPLETE]`
- Marked `docs/spec/master-spec.md` as `[SPLIT COMPLETE]`
- Added cross-references between all split files
- Added stability levels, version, and source metadata to each split file