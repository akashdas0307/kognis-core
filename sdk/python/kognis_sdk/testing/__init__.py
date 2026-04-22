"""Testing harness for Kognis plugin development.

Implements the TestCore fixture for unit and integration testing
of plugins without requiring the full core daemon.
"""

from __future__ import annotations

import asyncio
from dataclasses import dataclass, field
from typing import Any, Callable, Awaitable
from unittest.mock import AsyncMock

from kognis_sdk.control_plane import (
    ControlPlaneClient,
    DispatchMessage,
    PluginState,
)
from kognis_sdk.envelope import Envelope, create_envelope
from kognis_sdk.eventbus import EventBusClient
from kognis_sdk.manifest import Manifest, SlotRegistration
from kognis_sdk.plugin import Plugin, PluginConfig
from kognis_sdk.stateful_agent import StatefulAgent


@dataclass
class TestResult:
    """Result from a test dispatch."""
    envelope: Envelope
    processing_time_ms: int = 0
    errors: list[str] = field(default_factory=list)


class TestCore:  # noqa: N801 — named for SDK convention, not a pytest test class
    """Test fixture for Kognis plugin development.

    Provides a lightweight test harness that simulates the core
    daemon's behavior without requiring NATS, gRPC, or the actual
    core running.

    Usage:
        test_core = TestCore()

        # Test a stateless plugin
        test_core.register_plugin(my_plugin)
        result = await test_core.dispatch(slot="input_reception", envelope=env)
        assert result is not None

        # Test a stateful agent
        test_core.register_agent(my_agent)
        await test_core.run_cycle(my_agent)
    """

    def __init__(self) -> None:
        self._plugins: dict[str, Plugin] = {}
        self._agents: dict[str, StatefulAgent] = {}
        self._dispatched: list[dict[str, Any]] = []
        self._events_published: list[dict[str, Any]] = []

    def make_test_manifest(
        self,
        plugin_id: str = "com.test.plugin",
        plugin_name: str = "Test Plugin",
        slots: list[tuple[str, str]] | None = None,
        handler_mode: str = "stateless",
    ) -> Manifest:
        """Create a minimal test manifest."""
        slot_regs = []
        for pipeline, slot in (slots or [("user_text_interaction", "input_reception")]):
            slot_regs.append(SlotRegistration(pipeline=pipeline, slot=slot, priority=50))

        return Manifest(
            manifest_version=1,
            plugin_id=plugin_id,
            plugin_name=plugin_name,
            version="0.1.0",
            author="Test",
            license="MIT",
            description="Test plugin",
            language="python",
            runtime=type("RuntimeSpec", (), {"entrypoint": "test.py"})(),
            handler_mode=handler_mode,
            slot_registrations=slot_regs,
        )

    def make_test_envelope(
        self,
        message_type: str = "user_text_input",
        payload: dict[str, Any] | None = None,
        pipeline: str = "user_text_interaction",
        entry_slot: str = "input_reception",
    ) -> Envelope:
        """Create a test envelope for dispatch testing."""
        return create_envelope(
            origin_plugin="test_core",
            message_type=message_type,
            payload=payload or {"text": "test input"},
            pipeline=pipeline,
            entry_slot=entry_slot,
        )

    def register_plugin(self, plugin: Plugin) -> None:
        """Register a plugin for testing."""
        self._plugins[plugin.plugin_id] = plugin

    def register_agent(self, agent: StatefulAgent) -> None:
        """Register a stateful agent for testing."""
        self._agents[agent.plugin_id] = agent

    async def dispatch(
        self,
        slot: str,
        envelope: Envelope,
        plugin_id: str | None = None,
    ) -> TestResult:
        """Dispatch an envelope to a registered plugin's slot handler.

        Simulates the core daemon's dispatch mechanism.
        """
        if plugin_id:
            plugin = self._plugins.get(plugin_id)
            if plugin is None:
                return TestResult(envelope=envelope, errors=["Plugin not found"])
        else:
            plugin = next(iter(self._plugins.values()), None)
            if plugin is None:
                return TestResult(envelope=envelope, errors=["No plugins registered"])

        handler = plugin._slot_handlers.get(slot)
        if handler is None:
            return TestResult(envelope=envelope, errors=[f"No handler for slot: {slot}"])

        import time
        start = time.monotonic()
        try:
            result = await handler(envelope)
            elapsed_ms = int((time.monotonic() - start) * 1000)
            self._dispatched.append({
                "slot": slot,
                "plugin_id": plugin.plugin_id,
                "msg_type": envelope.message_type,
                "success": True,
            })
            return TestResult(envelope=result, processing_time_ms=elapsed_ms)
        except Exception as e:
            self._dispatched.append({
                "slot": slot,
                "plugin_id": plugin.plugin_id,
                "msg_type": envelope.message_type,
                "success": False,
                "error": str(e),
            })
            return TestResult(envelope=envelope, errors=[str(e)])

    async def run_cycle(self, agent: StatefulAgent) -> None:
        """Run one cognition cycle on a stateful agent."""
        await agent.cognition_cycle()

    @property
    def dispatch_count(self) -> int:
        """Number of dispatches processed."""
        return len(self._dispatched)

    @property
    def dispatched(self) -> list[dict[str, Any]]:
        """History of all dispatches."""
        return list(self._dispatched)

    def reset(self) -> None:
        """Reset test state."""
        self._dispatched.clear()
        self._events_published.clear()