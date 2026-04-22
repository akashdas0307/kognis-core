# Capability Registry

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [02-plugin-manifest.md](02-plugin-manifest.md), [04-handshake-protocols.md](04-handshake-protocols.md), [11-tool-bridge.md](11-tool-bridge.md)

## 5.1 Purpose

The capability registry is the core's live index of everything every plugin can do. It enables:
- Plugin-to-plugin discovery (double handshake queries)
- LLM tool exposure (Tool Bridge integration)
- Graceful degradation (checking if capability available before needing it)

## 5.2 Registry Structure

```
Capability Registry (in-memory, maintained by core):
├── By capability_id
│   ├── memory.retrieve_episodes
│   │   ├── providing_plugins: [memory]
│   │   ├── status: available
│   │   ├── schema: {params, response}
│   │   ├── latency_class: fast
│   │   └── llm_exposed_to: [cognitive_core, world_model]
│   ├── eal.get_environment_summary
│   │   └── ...
│   └── ...
├── By plugin_id
│   ├── memory
│   │   ├── provides: [memory.retrieve_episodes, memory.store_episode, ...]
│   │   └── requires: [inference.complete]
│   └── ...
└── By llm_tool_exposure
    ├── cognitive_core: [list of capabilities to expose in prompt]
    ├── world_model: [...]
    └── ...
```

## 5.3 Registry API

Core exposes these operations to plugins via control plane:

```
query_capability_available(capability_id) → boolean
list_capabilities_for_llm(requesting_plugin_id) → array of tool schemas
find_providers(capability_id) → array of plugin_ids
get_capability_schema(capability_id) → schema object
subscribe_to_capability_changes(capability_ids, callback)
```

## 5.4 Registry Lifecycle

- **On plugin registration:** Core reads provides_capabilities, adds entries
- **On plugin shutdown:** Core marks capabilities as `unavailable`
- **On plugin crash:** Capabilities marked `unavailable` immediately; restored when plugin healthy again
- **On registry change:** Core broadcasts `capability.changed` event; subscribers can react

## 5.5 Capability Namespacing

Capability IDs follow the convention: `<plugin_namespace>.<capability_name>`

Examples:
- `memory.retrieve_episodes`
- `memory.store_episode`
- `eal.get_environment_summary`
- `persona.get_current_emotional_state`
- `world_model.review_proposed_action`
- `inference.complete`

This prevents conflicts and makes ownership clear.

## 5.6 Conflicts

If two plugins declare the same capability_id:
- Registration rejects the second
- Error code: `CAPABILITY_CONFLICT`
- Solution: plugins must namespace properly

Exception: redundancy/failover (future feature) — capabilities can declare themselves as alternatives.