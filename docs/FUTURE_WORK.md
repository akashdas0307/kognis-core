# Future Work — Kognis Framework

> **Purpose:** Track out-of-scope improvements discovered during development
> **Stability:** EVOLVING
> **Rule:** Do NOT implement items listed here within current milestones. Record and move on.

---

## Plugin SDK Enhancements

- **Go SDK** — Secondary plugin SDK for Go-native plugins. Lower latency for core-adjacent plugins.
- **Node.js SDK** — Tertiary SDK for web ecosystem plugins.
- **Rust SDK** — Potential future for ultra-low-latency plugins.

## Core Daemon

- **Hot plugin reload** — Reload a plugin without restarting the entire daemon.
- **Plugin dependency graph** — Auto-resolve plugin startup order from declared dependencies.
- **Dynamic pipeline reconfiguration** — Change pipeline templates at runtime without restart.
- **Plugin resource limits** — CPU and memory limits per plugin process (cgroups).
- **Web dashboard** — HTTP-based dashboard in addition to TUI.

## Event Bus

- **Message persistence** — Optional NATS JetStream for durable message storage.
- **Message replay** — Replay messages from a specific point for debugging.
- **Rate limiting** — Per-plugin rate limits on event bus publish.

## Memory System

- **ChromaDB integration** — Embedding storage and similarity search.
- **Memory compaction** — Automated consolidation of similar memories during sleep.
- **Memory export/import** — Backup and restore memory state.

## Security

- **Plugin sandboxing** — seccomp/AppArmor profiles for plugin isolation.
- **Capability revocation** — Dynamic revocation of granted capabilities.
- **Audit log** — Comprehensive audit trail of all capability invocations.

## Developer Experience

- **Plugin scaffolding CLI** — `kognis plugin create <name>` command.
- **Local development server** — `kognis dev` with hot reload and debug mode.
- **Plugin marketplace** — Search and install plugins from kognis-registry.
- **VS Code extension** — Syntax highlighting and validation for plugin.yaml.

## Testing

- **Chaos testing** — Random plugin failures and network partitions.
- **Load testing** — Simulate high message throughput.
- **Fuzzing** — Fuzz the message envelope parser.

## Documentation

- **Architecture decision records** — Formal ADRs for key decisions.
- **Plugin authoring guide** — Step-by-step tutorial for new plugin developers.
- **API reference** — Auto-generated from code docstrings.

---

*Items move from here to milestone scope only when explicitly approved by the creator.*