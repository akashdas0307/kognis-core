---
name: kognis-ci-precheck
description: Mandatory local validation matching GitHub Actions workflows before commit.
---
# CI Pre-check Rule

Before committing any "Soft Milestone", the following local checks MUST pass:
- **Go Core**: `golangci-lint run` (or `go vet`) and `go test ./...` within the `core/` directory.
- **Python SDK**: `ruff check .`, `mypy .`, and `pytest` within the `sdk/python/` directory.
- **Documentation**: Verify cross-references in `docs/` and run any available `validate-docs.yaml` equivalents.
- **Auto-Correction**: If local linting fails, the agent must fix the issues automatically before re-attempting the commit.
