# Tool Bridge

> **Stability:** STABLE
> **Version:** 0.1.0
> **Source:** Split from master-spec.md
> **Related:** [05-capability-registry.md](05-capability-registry.md), [10-context-budget-manager.md](10-context-budget-manager.md)

## 11.1 Purpose

Translates between the framework's internal capability system and LLM-facing tool-call protocols.

Two distinct layers that must not be conflated:
- **Layer 1:** Plugin-to-plugin capability queries (internal plumbing, no LLM involvement)
- **Layer 2:** LLM tool calls (model decides to use a tool based on prompt-exposed schema)

Tool Bridge is the translation layer that exists inside every LLM-using plugin.

## 11.2 Architecture

```
Inside a stateful agent plugin (e.g., Cognitive Core):

  ┌─────────────────────────────────────────┐
  │ Capability Registry Client              │
  │ (via core control plane)                 │
  └────────────────┬────────────────────────┘
                    │
                    ↓
  ┌─────────────────────────────────────────┐
  │ Tool Bridge                              │
  │   ↓ Prompt Assembly                      │
  │ Translate registry entries → OpenAI/     │
  │ Anthropic tool-call schema               │
  │   ↑ Tool Use Handling                    │
  │ Translate LLM tool_use blocks → capability│
  │ queries → execute → translate result     │
  │ back as tool_result                      │
  └────────────────┬────────────────────────┘
                    │
                    ↓
  ┌─────────────────────────────────────────┐
  │ LLM Interface                            │
  │ (talks to Inference Gateway plugin)      │
  └─────────────────────────────────────────┘
```

## 11.3 Prompt-Time Tool Assembly

Before each LLM call:

```python
# Pseudo-code
available_tools = []
for capability in capability_registry.list_for_llm("cognitive_core"):
    tool_schema = {
        "name": capability.id,
        "description": capability.llm_tool_description,
        "parameters": capability.params_schema
    }
    available_tools.append(tool_schema)

prompt_with_tools = assemble_prompt(context_blocks, available_tools)
response = inference_gateway.complete(prompt_with_tools)
```

## 11.4 Tool Use Handling

When LLM emits tool_use:

```python
for tool_use_block in response.tool_uses:
    capability_id = tool_use_block.name
    params = tool_use_block.params
    
    # Double-handshake capability query
    result = await capability_registry.query(
        target=capability_id,
        params=params,
        await_response=True
    )
    
    # Return result to LLM as tool_result
    tool_results.append({
        "tool_use_id": tool_use_block.id,
        "content": result
    })
```

## 11.5 Security Boundaries

- `llm_tool_expose_to` in manifest controls which plugins' LLMs see which capabilities
- Not all capabilities are LLM-exposed
- Capabilities marked `authentication_required: true` never exposed to LLM
- LLM cannot invoke capabilities outside its exposure list (enforced at registry query time)

## 11.6 New Tool Auto-Discovery

When a new plugin registers providing LLM-exposed capabilities:
- Capability Registry broadcasts change
- Tool Bridges in active plugins update their cache
- Next LLM call includes new tool
- LLM can start using it immediately (with appropriate description)

This is the mechanism that enables "plug in robotic arms, the being uses them" — described in the Foundation document.