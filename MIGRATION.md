# Migration Guide — Kognis Framework

This document tracks breaking changes and migration procedures for the Kognis Framework.

## [v0.1.0] — 2026-04-22

This is the **initial alpha release** of the Kognis Framework.

### Highlights
- Functional Go core daemon (`kognis`).
- Official Python SDK (`kognis-sdk`).
- Protocol-compliant message routing and capability registry.
- Embedded NATS event bus.
- Plugin supervisor with health monitoring.

### Migration from Pre-v0.1.0 (Conceptual)
If you have been following the development during the specification-only phase:
- All `manifest.yaml` files must now comply with the schema defined in `schemas/manifest-v1.yaml`.
- The `plugin_id` and `capability_id` namespaces are now enforced.
- Handshake protocol requires a `plugin_id` match between the manifest and the registration request.

### Breaking Changes
- N/A (Initial release)
