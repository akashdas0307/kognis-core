# Changelog

All notable changes to the Kognis Framework will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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