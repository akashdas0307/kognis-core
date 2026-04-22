"""Capability registry client for double-handshake capability queries.

Implements SPEC 05: Capability Registry. Provides client-side
interaction with the core's capability registry.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from kognis_sdk.control_plane import CapabilityQuery, CapabilityResponse, ControlPlaneClient


@dataclass
class RegistryEntry:
    """A capability entry in the registry."""

    capability_id: str
    providing_plugins: list[str] = field(default_factory=list)
    status: str = "available"
    params_schema: dict[str, Any] | None = None
    response_schema: dict[str, Any] | None = None
    latency_class: str = "medium"
    llm_exposed_to: list[str] = field(default_factory=list)


class CapabilityRegistryClient:
    """Client for interacting with the core's capability registry.

    Spec reference: docs/spec/05-capability-registry.md

    Enables plugin-to-plugin capability discovery and invocation
    via the double handshake protocol (SPEC 04 Section 4.5).
    """

    def __init__(self, control_plane: ControlPlaneClient) -> None:
        self.control_plane = control_plane
        self._cache: dict[str, RegistryEntry] = {}

    async def is_available(self, capability_id: str) -> bool:
        """Check if a capability is currently available.

        Spec reference: SPEC 05 Section 5.3 — query_capability_available
        """
        entry = self._cache.get(capability_id)
        if entry is None:
            return False
        return entry.status == "available"

    async def find_providers(self, capability_id: str) -> list[str]:
        """Find plugins providing a capability.

        Spec reference: SPEC 05 Section 5.3 — find_providers
        """
        entry = self._cache.get(capability_id)
        if entry is None:
            return []
        return [p for p in entry.providing_plugins]

    async def get_schema(self, capability_id: str) -> dict[str, Any] | None:
        """Get a capability's parameter and response schemas.

        Spec reference: SPEC 05 Section 5.3 — get_capability_schema
        """
        entry = self._cache.get(capability_id)
        if entry is None:
            return None
        return {
            "params_schema": entry.params_schema,
            "response_schema": entry.response_schema,
        }

    async def list_for_llm(self, requesting_plugin_id: str) -> list[dict[str, Any]]:
        """List capabilities exposed to a plugin's LLM.

        Spec reference: SPEC 05 Section 5.3 — list_capabilities_for_llm

        Returns tool-call schemas suitable for LLM prompt assembly.
        """
        tools: list[dict[str, Any]] = []
        for entry in self._cache.values():
            if requesting_plugin_id in entry.llm_exposed_to and entry.status == "available":
                tool_schema = {
                    "name": entry.capability_id,
                    "description": entry.capability_id,
                    "parameters": entry.params_schema or {"type": "object", "properties": {}},
                }
                tools.append(tool_schema)
        return tools

    async def query(
        self,
        target: str,
        params: dict[str, Any] | None = None,
        await_response: bool = True,
        correlation_id: str = "",
    ) -> CapabilityResponse:
        """Execute a double handshake capability query.

        Spec reference: SPEC 04 Section 4.5

        Args:
            target: capability_id to invoke.
            params: Parameters for the capability.
            await_response: If true, wait for result.
            correlation_id: For correlating request/response.

        Returns:
            CapabilityResponse with query result.
        """
        query = CapabilityQuery(
            target_capability=target,
            params=params or {},
            await_response=await_response,
            correlation_id=correlation_id,
        )
        return await self.control_plane.query_capability(query)

    def update_cache(self, entries: list[RegistryEntry]) -> None:
        """Update the local capability cache.

        Called when capability.changed events are received.
        """
        for entry in entries:
            self._cache[entry.capability_id] = entry

    def remove_from_cache(self, capability_id: str) -> None:
        """Remove a capability from cache when provider goes down."""
        if capability_id in self._cache:
            self._cache[capability_id].status = "unavailable"

    def clear_cache(self) -> None:
        """Clear the entire cache."""
        self._cache.clear()
