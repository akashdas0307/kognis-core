"""Tests for plugin base class and stateful agent — M-005, M-006.

Validates SPEC 02 handler_mode and SPEC 08 lifecycle.
"""

import pytest

from kognis_sdk.control_plane import PluginState
from kognis_sdk.envelope import Envelope
from kognis_sdk.manifest import Manifest, SlotRegistration
from kognis_sdk.plugin import Plugin, PluginError
from kognis_sdk.stateful_agent import StatefulAgent


def make_manifest(handler_mode: str = "stateless") -> Manifest:
    return Manifest(
        manifest_version=1,
        plugin_id="com.test.plugin",
        plugin_name="Test",
        version="1.0",
        author="A",
        license="MIT",
        description="test",
        language="python",
        runtime=type("R", (), {"entrypoint": "x.py"})(),
        handler_mode=handler_mode,
        slot_registrations=[SlotRegistration(pipeline="p", slot="s", priority=50)],
    )


class TestPlugin:
    def test_creation(self):
        m = make_manifest()
        p = Plugin(manifest=m)
        assert p.plugin_id == "com.test.plugin"
        assert p.state == PluginState.UNREGISTERED
        assert not p.is_running

    def test_register_slot_handler(self):
        m = make_manifest()
        p = Plugin(manifest=m)

        async def handler(env: Envelope) -> Envelope:
            return env

        p.register_slot_handler("s", handler)
        assert "s" in p._slot_handlers

    @pytest.mark.asyncio
    async def test_start(self):
        m = make_manifest()
        p = Plugin(manifest=m)

        async def handler(env: Envelope) -> Envelope:
            return env

        p.register_slot_handler("s", handler)
        await p.start()
        assert p.state == PluginState.HEALTHY_ACTIVE
        assert p.is_running
        await p.stop()

    @pytest.mark.asyncio
    async def test_stop(self):
        m = make_manifest()
        p = Plugin(manifest=m)
        await p.start()
        await p.stop()
        assert p.state == PluginState.SHUT_DOWN
        assert not p.is_running

    @pytest.mark.asyncio
    async def test_emit_event_not_connected(self):
        m = make_manifest()
        p = Plugin(manifest=m)
        with pytest.raises(PluginError, match="not_connected"):
            await p.emit_event("topic", {})

    @pytest.mark.asyncio
    async def test_on_startup_hook(self):
        m = make_manifest()
        started = []

        class MyPlugin(Plugin):
            async def on_startup(self):
                started.append(True)

        p = MyPlugin(manifest=m)
        await p.start()
        assert started == [True]
        await p.stop()

    @pytest.mark.asyncio
    async def test_on_shutdown_hook(self):
        m = make_manifest()
        shutdown = []

        class MyPlugin(Plugin):
            async def on_shutdown(self):
                shutdown.append(True)

        p = MyPlugin(manifest=m)
        await p.start()
        await p.stop()
        assert shutdown == [True]

    @pytest.mark.asyncio
    async def test_on_health_check(self):
        m = make_manifest()
        p = Plugin(manifest=m)
        health = await p.on_health_check()
        assert "status" in health


class TestStatefulAgent:
    def test_creation(self):
        m = make_manifest(handler_mode="stateful_agent")
        agent = StatefulAgent(manifest=m)
        assert agent.plugin_id == "com.test.plugin"
        assert not agent.is_running

    def test_working_memory(self):
        m = make_manifest(handler_mode="stateful_agent")
        agent = StatefulAgent(manifest=m)
        assert agent.working_memory == {}

    def test_register_slot_handler(self):
        m = make_manifest(handler_mode="stateful_agent")
        agent = StatefulAgent(manifest=m)

        async def handler(env: Envelope) -> Envelope:
            return env

        agent.register_slot_handler("s", handler)
        assert "s" in agent._slot_handlers

    @pytest.mark.asyncio
    async def test_cognition_cycle(self):
        m = make_manifest(handler_mode="stateful_agent")
        agent = StatefulAgent(manifest=m)
        await agent.cognition_cycle()
        assert agent._cycle_count == 1

    @pytest.mark.asyncio
    async def test_compact_working_memory(self):
        m = make_manifest(handler_mode="stateful_agent")
        agent = StatefulAgent(manifest=m)
        for i in range(10):
            agent.working_memory[f"key_{i}"] = f"value_{i}"
        await agent.compact_working_memory()
        assert len(agent.working_memory) < 10

    @pytest.mark.asyncio
    async def test_compact_small_memory_noop(self):
        m = make_manifest(handler_mode="stateful_agent")
        agent = StatefulAgent(manifest=m)
        agent.working_memory["a"] = 1
        agent.working_memory["b"] = 2
        await agent.compact_working_memory()
        assert len(agent.working_memory) == 2

    @pytest.mark.asyncio
    async def test_start_and_stop(self):
        m = make_manifest(handler_mode="stateful_agent")
        agent = StatefulAgent(manifest=m)
        await agent.start()
        assert agent.state == PluginState.HEALTHY_ACTIVE
        assert agent.is_running
        await agent.stop()
        assert agent.state == PluginState.SHUT_DOWN
        assert not agent.is_running

    @pytest.mark.asyncio
    async def test_on_wake_on_sleep_hooks(self):
        m = make_manifest(handler_mode="stateful_agent")
        woke = []
        slept = []

        class MyAgent(StatefulAgent):
            async def on_wake(self):
                woke.append(True)

            async def on_sleep(self):
                slept.append(True)

        agent = MyAgent(manifest=m)
        await agent.on_wake()
        await agent.on_sleep()
        assert woke == [True]
        assert slept == [True]

    @pytest.mark.asyncio
    async def test_health_check(self):
        m = make_manifest(handler_mode="stateful_agent")
        agent = StatefulAgent(manifest=m)
        health = await agent.on_health_check()
        assert "cycle_count" in health
