"""Tests for control plane client — M-003.

Validates SPEC 04: Handshake Protocols.
"""

import asyncio
import pytest

from kognis_sdk.control_plane import (
    ControlPlaneClient,
    ControlPlaneError,
    PluginState,
    RegisterRequest,
    RegisterAck,
    ReadyMessage,
    DispatchMessage,
    DispatchAck,
    DispatchComplete,
    DispatchFailed,
    CapabilityQuery,
    CapabilityResponse,
    Heartbeat,
    HeartbeatAck,
    ShutdownRequest,
)
from kognis_sdk.envelope import Envelope, create_envelope
from kognis_sdk.manifest import Manifest, SlotRegistration


def make_manifest() -> Manifest:
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
        handler_mode="stateless",
        slot_registrations=[SlotRegistration(pipeline="p", slot="s", priority=50)],
    )


class TestPluginState:
    def test_states_exist(self):
        assert PluginState.UNREGISTERED.value == "UNREGISTERED"
        assert PluginState.HEALTHY_ACTIVE.value == "HEALTHY_ACTIVE"
        assert PluginState.SHUT_DOWN.value == "SHUT_DOWN"
        assert PluginState.DEAD.value == "DEAD"


class TestRegisterRequest:
    def test_to_dict(self):
        m = make_manifest()
        req = RegisterRequest(manifest=m, pid=1234)
        d = req.to_dict()
        assert d["pid"] == 1234
        assert d["manifest"]["plugin_id"] == "com.test.plugin"


class TestRegisterAck:
    def test_fields(self):
        ack = RegisterAck(plugin_id_runtime="p_1", event_bus_token="tok", config_bundle={"k": "v"})
        assert ack.plugin_id_runtime == "p_1"
        assert ack.event_bus_token == "tok"
        assert ack.config_bundle == {"k": "v"}


class TestControlPlaneClient:
    @pytest.mark.asyncio
    async def test_connect(self):
        cp = ControlPlaneClient()
        await cp.connect()
        assert cp.is_connected
        assert cp.state == PluginState.UNREGISTERED

    @pytest.mark.asyncio
    async def test_register(self):
        cp = ControlPlaneClient()
        await cp.connect()
        m = make_manifest()
        ack = await cp.register(m, pid=100)
        assert cp.state == PluginState.REGISTERED
        assert ack.plugin_id_runtime.startswith("com.test.plugin")
        assert ack.event_bus_token != ""

    @pytest.mark.asyncio
    async def test_register_invalid_state(self):
        cp = ControlPlaneClient()
        await cp.connect()
        m = make_manifest()
        await cp.register(m, pid=100)
        with pytest.raises(ControlPlaneError, match="invalid_state"):
            await cp.register(m, pid=100)

    @pytest.mark.asyncio
    async def test_send_ready(self):
        cp = ControlPlaneClient()
        await cp.connect()
        m = make_manifest()
        await cp.register(m, pid=100)
        await cp.send_ready(subscribed_topics=["topic.a"])
        assert cp.state == PluginState.HEALTHY_ACTIVE
        assert cp.is_running

    @pytest.mark.asyncio
    async def test_send_ready_invalid_state(self):
        cp = ControlPlaneClient()
        with pytest.raises(ControlPlaneError, match="invalid_state"):
            await cp.send_ready(subscribed_topics=[])

    @pytest.mark.asyncio
    async def test_dispatch_handler(self):
        cp = ControlPlaneClient()
        await cp.connect()
        m = make_manifest()
        await cp.register(m, pid=100)
        await cp.send_ready(subscribed_topics=[])

        env = create_envelope(origin_plugin="x", message_type="t", payload={}, pipeline="p")

        async def handler(msg: DispatchMessage) -> Envelope:
            return msg.envelope.with_enrichment("test", {"processed": True})

        cp.register_dispatch_handler("s", handler)

        msg = DispatchMessage(msg_id="m1", envelope=env, deadline_ms=5000, slot="s")
        result = await cp.dispatch(msg)
        assert result.enrichments.get("test") == {"processed": True}

    @pytest.mark.asyncio
    async def test_dispatch_no_handler(self):
        cp = ControlPlaneClient()
        await cp.connect()
        m = make_manifest()
        await cp.register(m, pid=100)
        await cp.send_ready(subscribed_topics=[])

        env = create_envelope(origin_plugin="x", message_type="t", payload={}, pipeline="p")
        msg = DispatchMessage(msg_id="m1", envelope=env, deadline_ms=5000, slot="unknown")
        with pytest.raises(ControlPlaneError, match="no_handler"):
            await cp.dispatch(msg)

    @pytest.mark.asyncio
    async def test_shutdown(self):
        cp = ControlPlaneClient()
        await cp.connect()
        m = make_manifest()
        await cp.register(m, pid=100)
        await cp.send_ready(subscribed_topics=[])
        assert cp.is_running

        await cp.shutdown()
        assert cp.state == PluginState.SHUT_DOWN
        assert not cp.is_running

    @pytest.mark.asyncio
    async def test_heartbeat(self):
        cp = ControlPlaneClient()
        await cp.connect()
        m = make_manifest()
        await cp.register(m, pid=100)
        await cp.send_ready(subscribed_topics=[])

        ack = await cp.send_heartbeat(metrics={"queue_depth": 5})
        assert ack.server_time != ""

    @pytest.mark.asyncio
    async def test_close(self):
        cp = ControlPlaneClient()
        await cp.connect()
        assert cp.is_connected
        await cp.close()
        assert not cp.is_connected