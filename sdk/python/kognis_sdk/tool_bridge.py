"""Tool Bridge — LLM tool-call translation layer.

Implements SPEC 11: Tool Bridge. Translates between the framework's
capability system and LLM-facing tool-call protocols.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from kognis_sdk.capability import CapabilityRegistryClient


@dataclass
class ToolSchema:
    """An LLM-facing tool definition."""
    name: str
    description: str
    parameters: dict[str, Any] = field(default_factory=dict)


@dataclass
class ToolUseBlock:
    """An LLM tool_use response block."""
    id: str
    name: str
    params: dict[str, Any] = field(default_factory=dict)


@dataclass
class ToolResult:
    """Result to return to LLM for a tool_use."""
    tool_use_id: str
    content: Any


class ToolBridge:
    """Translates between capability registry and LLM tool-call schemas.

    Spec reference: docs/spec/11-tool-bridge.md

    Two distinct layers:
    - Layer 1: Plugin-to-plugin queries (internal, no LLM)
    - Layer 2: LLM tool calls (model decides to use based on prompt)

    Tool Bridge handles Layer 2 translation.
    """

    def __init__(self, capability_client: CapabilityRegistryClient, plugin_id: str) -> None:
        self.capability_client = capability_client
        self.plugin_id = plugin_id
        self._tool_cache: list[ToolSchema] = []

    async def assemble_tools(self) -> list[ToolSchema]:
        """Build tool-call schemas for LLM prompt.

        Spec reference: SPEC 11 Section 11.3

        Queries capability registry for tools exposed to this plugin,
        translates to OpenAI/Anthropic tool-call format.
        """
        raw_tools = await self.capability_client.list_for_llm(self.plugin_id)
        tools: list[ToolSchema] = []
        for raw in raw_tools:
            tool = ToolSchema(
                name=raw.get("name", ""),
                description=raw.get("description", ""),
                parameters=raw.get("parameters", {}),
            )
            tools.append(tool)
        self._tool_cache = tools
        return tools

    async def handle_tool_uses(self, tool_uses: list[ToolUseBlock]) -> list[ToolResult]:
        """Process LLM tool_use blocks and return results.

        Spec reference: SPEC 11 Section 11.4

        For each tool_use:
        1. Extract capability_id and params
        2. Execute double-handshake capability query
        3. Return result as tool_result
        """
        results: list[ToolResult] = []
        for block in tool_uses:
            response = await self.capability_client.query(
                target=block.name,
                params=block.params,
                await_response=True,
            )
            results.append(ToolResult(
                tool_use_id=block.id,
                content=response.result,
            ))
        return results

    def get_cached_tools(self) -> list[ToolSchema]:
        """Return the most recently assembled tools without re-querying."""
        return self._tool_cache

    async def refresh_tools(self) -> list[ToolSchema]:
        """Force re-fetch and cache tools from capability registry.

        Called when capability.changed events are received.
        """
        return await self.assemble_tools()
