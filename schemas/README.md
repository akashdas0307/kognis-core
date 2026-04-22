# Kognis Framework — Shared Schemas

This directory contains the canonical YAML schema definitions for all Kognis data formats. These schemas are derived from the technical specifications in `docs/spec/`.

## Schema Files

| Schema | File | Source Spec |
|---|---|---|
| Plugin Manifest | `manifest-v1.yaml` | SPEC 02 |
| Message Envelope | `envelope-v1.yaml` | SPEC 01 |
| Pipeline Template | `pipeline-template-v1.yaml` | SPEC 03 |
| Health Pulse | `health-pulse-v1.yaml` | SPEC 18 |
| State Broadcast | `state-broadcast-v1.yaml` | SPEC 06 |
| Registry Entry | `registry-entry-v1.yaml` | kognis-registry |

## Usage

### For Plugin Authors

Plugin authors should validate their `plugin.yaml` against `manifest-v1.yaml` before submission. The SDK provides built-in validation via `kognis_sdk.manifest.validate()`.

### For Framework Developers

When implementing core components, use these schemas as the authoritative field definitions. If a schema conflicts with a spec, the spec takes precedence — file an issue.

## Versioning

Schema versions follow the `vN` convention in filenames. When a breaking change is needed, create a new file (e.g., `manifest-v2.yaml`) rather than modifying the existing one.

## Validation

```bash
# Validate a plugin manifest
python3 scripts/validate-manifest.py path/to/plugin.yaml

# Validate all schemas
python3 -m pytest tests/integration/test_schemas.py
```