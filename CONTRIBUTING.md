# Contributing to Kognis Framework

Thank you for your interest in the Kognis Framework. This document describes how to contribute.

## Code of Conduct

All contributors must follow our Code of Conduct. Be respectful, constructive, and inclusive.

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Read `docs/DEVELOPMENT_SOP.md` for the development workflow
4. Read `docs/GLOSSARY.md` for terminology
5. Read the relevant specs in `docs/spec/` before writing code

## Development Workflow

The Kognis Framework follows a spec-first discipline:

1. **Read the spec** — Every component has a specification in `docs/spec/`
2. **Write tests first** — Tests are written against the spec
3. **Implement** — Code is written to pass tests
4. **Verify** — All quality gates must pass before merge

See `docs/DEVELOPMENT_SOP.md` for the complete workflow.

## Branch Naming

| Type | Pattern | Example |
|---|---|---|
| Feature | `feature/M-XXX-slug` | `feature/M-001-manifest-parser` |
| Bug fix | `fix/<short-desc>` | `fix/router-race-condition` |
| Spec revision | `spec/<spec-name>` | `spec/revise-handshake-timeout` |
| Documentation | `docs/<scope>` | `docs/add-plugin-examples` |

## Commit Messages

Format: `<type>(<scope>): <subject>`

Types: `feat`, `fix`, `docs`, `spec`, `refactor`, `test`, `chore`, `perf`, `style`

Example: `feat(sdk): implement manifest parser with YAML validation`

## Pull Requests

1. Ensure all tests pass
2. Ensure lint is clean (Go: vet + golangci-lint; Python: ruff + mypy)
3. Update documentation if relevant
4. Reference the spec in your PR description
5. Keep PRs focused — one logical change per PR

## Technology Stack

The stack is a locked decision. Do not propose alternatives:

- Core daemon: **Go**
- Plugin SDK (primary): **Python 3.11+**
- Event bus: **NATS** (embedded)
- Control plane: **gRPC** over Unix socket
- Memory storage: **SQLite + FTS5** (metadata), **ChromaDB** (embeddings)
- Local LLM: **Ollama**
- TUI: **charmbracelet/bubbletea**
- Agent navigation: **tmux**

## License

- **Code:** MIT License
- **Research content** (`docs/foundations/`): Proprietary to project creator

By contributing code, you agree your contribution is licensed under the MIT License.

## Questions?

Open an issue with the `question` label.