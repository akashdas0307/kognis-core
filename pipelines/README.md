# Kognis Framework — Canonical Pipelines

This directory contains the canonical pipeline templates that ship with the Kognis Framework. Plugins register for slots in these pipelines — they do not define pipelines.

## Pipeline Catalog

| Pipeline | Description | Input Types |
|---|---|---|
| `user_text_interaction` | Text-based user interaction flow | `user_text_input` |
| `user_voice_interaction` | Voice-based user interaction flow | `voice_input` |
| `background_monitoring` | Ambient observations that may escalate | `ambient_audio`, `visual_frame`, `eal_escalation` |
| `autonomous_cognition` | System-initiated cognition (daydream, wake) | `wake_up_handoff`, `daydream_seed`, `internal_trigger` |
| `sleep_consolidation` | Sleep-time memory and identity processing | `internal_trigger` |
| `health_management` | Health alerts and diagnostics | `health_alert` |
| `offspring_evaluation` | Variant testing for self-improvement | `internal_trigger` |

## How Pipelines Work

1. A message enters a pipeline at a valid entry point slot
2. The router consults the dispatch table to find plugins registered for each slot
3. Each slot processes the message and passes it to the next slot
4. Enrichments are additive — each plugin adds to its own namespace
5. If a slot has no registered plugins, behavior depends on `on_empty` setting

## Slot Execution Modes

| Mode | Description |
|---|---|
| `sequential_by_priority` | Plugins run one at a time, ordered by priority (lower first) |
| `parallel` | All plugins run concurrently |
| `by_action_type` | Plugins matched to action types in the message |
| `by_channel_match` | Plugins matched to output channel in the message |

## Adding New Pipelines

Plugins can contribute new pipeline templates (heavyweight extension):
1. Manifest declares `contributes_pipeline_template: <path>`
2. Template validated against `schemas/pipeline-template-v1.yaml`
3. Template ID cannot conflict with existing
4. Template added to catalog at runtime

Use sparingly — default to using canonical pipelines.

## Validation

```bash
# Validate all pipeline YAML files
python3 -c "
import yaml, glob
for f in glob.glob('pipelines/*.yaml'):
    if 'README' in f: continue
    with open(f) as fh:
        yaml.safe_load(fh)
    print(f'OK: {f}')
"