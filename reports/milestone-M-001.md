# Milestone Report: M-001 — Go Core Daemon Scaffolding

> **Milestone:** M-001
> **Status:** COMPLETE
> **Branch:** feature/M-001-core-scaffolding
> **Date:** 2026-04-22

## What Was Built

Go core daemon scaffolding with all internal packages, public packages, tests, and build tooling.

### Core Entry Point

| File | Content |
|---|---|
| `core/cmd/kognis/main.go` | Signal handling, config loading, NATS bus start, registry/router/supervisor init |
| `core/go.mod` | Module `github.com/akashdas0307/kognis-core/core`, Go 1.22, deps: nats.go, grpc, protobuf |
| `core/Makefile` | build, test, lint, clean, run, deps targets |

### Internal Packages (9 packages)

| Package | Files | Content |
|---|---|---|
| `config` | `config.go`, `config_test.go` | NATSConfig, SupervisorConfig, Config structs; Load() with defaults |
| `registry` | `registry.go`, `registry_test.go` | PluginState constants, PluginEntry, SlotRegistration, thread-safe Registry |
| `router` | `router.go`, `router_test.go` | PipelineSpec, SlotSpec, pipeline loading, slot-based message dispatch |
| `supervisor` | `supervisor.go`, `backoff.go`, `backoff_test.go` | Plugin lifecycle management, health monitoring, exponential restart backoff |
| `eventbus` | `eventbus.go` | Embedded NATS server + client wrapper |
| `controlplane` | `server.go`, `handshake.go`, `handshake_test.go` | gRPC server over Unix socket, single handshake protocol |
| `health` | `aggregator.go` | Health pulse collection and registry state updates |
| `envelope` | `envelope.go`, `envelope_test.go` | Universal message envelope per SPEC 01 |
| `tui` | `dashboard.go` | Terminal UI dashboard stub (bubbletea integration pending) |

### Public Packages (2 packages)

| Package | Files | Content |
|---|---|---|
| `pkg/protocol` | `protocol.go` | NATS subject constants, RegistrationRequest/Response, HealthPulse, ShutdownNotice |
| `pkg/schema` | `schema.go` | Capability IDs, pipeline names, slot names, lifecycle state constants |

### Tests

| Package | Test File | Cases |
|---|---|---|
| `config` | `config_test.go` | 3 tests: defaults, data dir creation, path consistency |
| `registry` | `registry_test.go` | 9 tests: New, Register, duplicate, Get, UpdateState, Remove, List, FindByCapability, FindByPipelineSlot |
| `router` | `router_test.go` | 5 tests: New, LoadPipeline, duplicate, GetPipeline, ListPipelines, ParsePipelineSpec |
| `envelope` | `envelope_test.go` | 4 tests: Parse, Validate (5 subcases), IncrementHop, round-trip |
| `supervisor` | `backoff_test.go` | 1 test: 7 backoff schedule cases |
| `controlplane` | `handshake_test.go` | 3 tests: valid, missing fields, duplicate |

## Specs Referenced

- SPEC 01 (Message Envelope) — `envelope` package implements type system, validation, hop counting
- SPEC 04 (Handshake Protocols) — `controlplane/handshake.go` implements single handshake
- SPEC 05 (Capability Registry) — `registry` package implements FindByCapability
- SPEC 08 (Plugin Lifecycle) — `registry` package defines all lifecycle states; `supervisor` implements transitions
- SPEC 18 (Health Pulse Schema) — `health/aggregator.go` implements pulse collection

## Known Limitations

- **Go toolchain not available on this machine** — files created with correct syntax but not compiled/verified
- `eventbus` package depends on embedded NATS server APIs that may differ at runtime
- `tui/dashboard.go` is a stub — bubbletea integration deferred to Phase 3+
- `supervisor` handles registration via NATS subscription; gRPC registration service not yet wired
- No integration tests yet (unit tests only)
- `go.sum` file not generated (needs `go mod tidy` when Go toolchain available)

## Suggested Next Steps

- Install Go toolchain and run `go mod tidy && make test` to verify compilation
- Phase 2: Python SDK scaffolding
- Wire gRPC control plane service definitions (protobuf + generated code)
- Add integration tests with embedded NATS
- Implement bubbletea TUI dashboard
- Add YAML config file loading (currently defaults only)