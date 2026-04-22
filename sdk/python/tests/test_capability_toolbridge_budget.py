"""Tests for capability registry, tool bridge, and context budget — M-007, M-008, M-009."""

import pytest

from kognis_sdk.capability import CapabilityRegistryClient, RegistryEntry
from kognis_sdk.context_budget import (
    BudgetConfig,
    ContextBlock,
    ContextBudgetError,
    ContextBudgetManager,
    PriorityTier,
)
from kognis_sdk.control_plane import ControlPlaneClient
from kognis_sdk.tool_bridge import ToolBridge, ToolSchema, ToolUseBlock


class TestRegistryEntry:
    def test_defaults(self):
        entry = RegistryEntry(capability_id="cap1")
        assert entry.status == "available"
        assert entry.latency_class == "medium"
        assert entry.providing_plugins == []


class TestCapabilityRegistryClient:
    def setup_method(self):
        cp = ControlPlaneClient()
        self.client = CapabilityRegistryClient(control_plane=cp)

    @pytest.mark.asyncio
    async def test_is_available_empty(self):
        result = await self.client.is_available("nonexistent")
        assert result is False

    @pytest.mark.asyncio
    async def test_is_available_with_cache(self):
        self.client.update_cache([
            RegistryEntry(capability_id="cap1", providing_plugins=["p1"]),
        ])
        result = await self.client.is_available("cap1")
        assert result is True

    @pytest.mark.asyncio
    async def test_find_providers(self):
        self.client.update_cache([
            RegistryEntry(capability_id="cap1", providing_plugins=["p1", "p2"]),
        ])
        providers = await self.client.find_providers("cap1")
        assert "p1" in providers

    @pytest.mark.asyncio
    async def test_find_providers_empty(self):
        providers = await self.client.find_providers("nonexistent")
        assert providers == []

    @pytest.mark.asyncio
    async def test_get_schema(self):
        self.client.update_cache([
            RegistryEntry(capability_id="cap1", params_schema={"type": "object"}, response_schema={"type": "object"}),
        ])
        schema = await self.client.get_schema("cap1")
        assert schema is not None
        assert "params_schema" in schema

    @pytest.mark.asyncio
    async def test_get_schema_nonexistent(self):
        schema = await self.client.get_schema("nonexistent")
        assert schema is None

    @pytest.mark.asyncio
    async def test_list_for_llm(self):
        self.client.update_cache([
            RegistryEntry(
                capability_id="cap1",
                llm_exposed_to=["cognitive_core"],
                params_schema={"type": "object", "properties": {"q": {"type": "string"}}},
            ),
            RegistryEntry(
                capability_id="cap2",
                llm_exposed_to=["other_plugin"],
            ),
        ])
        tools = await self.client.list_for_llm("cognitive_core")
        assert len(tools) == 1
        assert tools[0]["name"] == "cap1"

    @pytest.mark.asyncio
    async def test_list_for_llm_unavailable(self):
        self.client.update_cache([
            RegistryEntry(capability_id="cap1", llm_exposed_to=["p"], status="unavailable"),
        ])
        tools = await self.client.list_for_llm("p")
        assert tools == []

    @pytest.mark.asyncio
    async def test_remove_from_cache(self):
        self.client.update_cache([RegistryEntry(capability_id="cap1")])
        self.client.remove_from_cache("cap1")
        result = await self.client.is_available("cap1")
        assert result is False

    def test_clear_cache(self):
        self.client.update_cache([RegistryEntry(capability_id="cap1")])
        self.client.clear_cache()
        assert self.client._cache == {}


class TestToolBridge:
    def setup_method(self):
        cp = ControlPlaneClient()
        self.cap_client = CapabilityRegistryClient(control_plane=cp)
        self.bridge = ToolBridge(capability_client=self.cap_client, plugin_id="test_plugin")

    @pytest.mark.asyncio
    async def test_assemble_tools(self):
        self.cap_client.update_cache([
            RegistryEntry(
                capability_id="memory.retrieve",
                llm_exposed_to=["test_plugin"],
                params_schema={"type": "object"},
            ),
        ])
        tools = await self.bridge.assemble_tools()
        assert len(tools) == 1
        assert tools[0].name == "memory.retrieve"

    @pytest.mark.asyncio
    async def test_assemble_tools_empty(self):
        tools = await self.bridge.assemble_tools()
        assert tools == []

    @pytest.mark.asyncio
    async def test_get_cached_tools(self):
        self.bridge._tool_cache = [ToolSchema(name="t1", description="d")]
        cached = self.bridge.get_cached_tools()
        assert len(cached) == 1

    @pytest.mark.asyncio
    async def test_refresh_tools(self):
        self.cap_client.update_cache([
            RegistryEntry(capability_id="cap1", llm_exposed_to=["test_plugin"]),
        ])
        tools = await self.bridge.refresh_tools()
        assert len(tools) == 1

    @pytest.mark.asyncio
    async def test_handle_tool_uses(self):
        # Control plane must be in HEALTHY_ACTIVE state for capability queries
        await self.cap_client.control_plane.connect()
        from kognis_sdk.manifest import Manifest, SlotRegistration
        m = Manifest(
            manifest_version=1, plugin_id="test", plugin_name="t", version="1",
            author="a", license="MIT", description="d", language="python",
            runtime=type("R", (), {"entrypoint": "x.py"})(),
            handler_mode="stateless",
            slot_registrations=[SlotRegistration(pipeline="p", slot="s", priority=50)],
        )
        await self.cap_client.control_plane.register(m, pid=1)
        await self.cap_client.control_plane.send_ready(subscribed_topics=[])

        self.cap_client.update_cache([
            RegistryEntry(capability_id="cap1", status="available"),
        ])
        blocks = [ToolUseBlock(id="tu1", name="cap1", params={"q": "test"})]
        results = await self.bridge.handle_tool_uses(blocks)
        assert len(results) == 1
        assert results[0].tool_use_id == "tu1"


class TestContextBlock:
    def test_estimate_tokens_explicit(self):
        block = ContextBlock(name="test", content="hello", priority=PriorityTier.MUST, token_count=10)
        assert block.estimate_tokens() == 10

    def test_estimate_tokens_from_content(self):
        block = ContextBlock(name="test", content="a" * 40, priority=PriorityTier.MUST)
        estimated = block.estimate_tokens(chars_per_token=4.0)
        assert estimated == 10


class TestContextBudgetManager:
    def test_calculate_budget(self):
        mgr = ContextBudgetManager(BudgetConfig(model_context_window=128000, output_budget=4000, safety_margin=500))
        budget = mgr.calculate_available_budget()
        assert budget == 123500

    def test_assemble_within_budget(self):
        mgr = ContextBudgetManager(BudgetConfig(model_context_window=1000, output_budget=100, safety_margin=50))
        blocks = [
            ContextBlock(name="identity", content="a" * 100, priority=PriorityTier.MUST, token_count=50),
            ContextBlock(name="input", content="b" * 100, priority=PriorityTier.HIGH, token_count=50),
            ContextBlock(name="memory", content="c" * 100, priority=PriorityTier.LOW, token_count=50),
        ]
        result = mgr.assemble(blocks)
        assert len(result) == 3

    def test_assemble_trim_low(self):
        mgr = ContextBudgetManager(BudgetConfig(model_context_window=300, output_budget=50, safety_margin=50))
        blocks = [
            ContextBlock(name="must1", content="a", priority=PriorityTier.MUST, token_count=100),
            ContextBlock(name="high1", content="b", priority=PriorityTier.HIGH, token_count=50),
            ContextBlock(name="low1", content="c", priority=PriorityTier.LOW, token_count=200),
            ContextBlock(name="low2", content="d", priority=PriorityTier.LOW, token_count=200),
        ]
        result = mgr.assemble(blocks)
        must_in = any(b.name == "must1" for b in result)
        high_in = any(b.name == "high1" for b in result)
        assert must_in
        assert high_in
        assert len(result) < 4

    def test_assemble_trim_fails(self):
        mgr = ContextBudgetManager(BudgetConfig(model_context_window=100, output_budget=10, safety_margin=10))
        blocks = [
            ContextBlock(name="must1", content="a", priority=PriorityTier.MUST, token_count=200),
        ]
        with pytest.raises(ContextBudgetError, match="KGN-CONTEXT-TRIM_FAILED"):
            mgr.assemble(blocks)

    def test_trim_log(self):
        mgr = ContextBudgetManager(BudgetConfig(model_context_window=200, output_budget=10, safety_margin=10))
        blocks = [
            ContextBlock(name="must1", content="a", priority=PriorityTier.MUST, token_count=50),
            ContextBlock(name="low1", content="c", priority=PriorityTier.LOW, token_count=200),
        ]
        mgr.assemble(blocks)
        assert mgr.trim_count > 0

    def test_frequent_trimming(self):
        mgr = ContextBudgetManager(BudgetConfig(model_context_window=200, output_budget=10, safety_margin=10))
        for _ in range(10):
            blocks = [
                ContextBlock(name="must1", content="a", priority=PriorityTier.MUST, token_count=50),
                ContextBlock(name="low1", content="c", priority=PriorityTier.LOW, token_count=200),
            ]
            mgr.assemble(blocks)
        assert mgr.frequent_trimming(threshold=5)

    def test_priority_ordering(self):
        mgr = ContextBudgetManager(BudgetConfig(model_context_window=500, output_budget=50, safety_margin=50))
        blocks = [
            ContextBlock(name="low", content="a", priority=PriorityTier.LOW, token_count=50),
            ContextBlock(name="high", content="b", priority=PriorityTier.HIGH, token_count=50),
            ContextBlock(name="must", content="c", priority=PriorityTier.MUST, token_count=50),
            ContextBlock(name="medium", content="d", priority=PriorityTier.MEDIUM, token_count=50),
        ]
        result = mgr.assemble(blocks)
        names = [b.name for b in result]
        must_idx = names.index("must")
        high_idx = names.index("high")
        assert must_idx < high_idx
