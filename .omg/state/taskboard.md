# Phase 7 Stabilization Taskboard

## [ ] Wave 1: E2E Test Stabilization
- [ ] Task 7.1: Stabilize NATS test fixtures in `tests/e2e/conftest.py`
- [ ] Task 7.2: Fix race conditions in `test_daemon.py` and `test_routing.py`
- [ ] Task 7.3: Verify 100% pass rate over 5 iterations

## [ ] Wave 2: Repository Cleanup & Quality Gates
- [ ] Task 7.4: Audit `.gitignore` and clean repository artifacts (`*.log`, binaries, caches)
- [ ] Task 7.5: Resolve all `golangci-lint` issues in `core/`
- [ ] Task 7.6: Resolve all `ruff` and `mypy` issues in `sdk/python/`
