import os
import asyncio
import pytest

def test_daemon_is_running_and_socket_exists():
    """Verify that the daemon fixture starts the daemon and creates the socket."""
    socket_path = "/tmp/kognis.sock"
    assert os.path.exists(socket_path), f"Socket {socket_path} should exist"

@pytest.mark.asyncio
async def test_mock_plugin_can_start(mock_plugin_factory):
    """Verify that a mock plugin can connect to the daemon."""
    plugin = await mock_plugin_factory(plugin_id="test_plugin_1")
    assert plugin.is_running
    assert plugin.plugin_id == "test_plugin_1"

@pytest.mark.asyncio
async def test_plugin_pulse_updates_daemon_state(mock_plugin_factory):
    """Verify that a plugin's health pulse updates the daemon's state for that plugin."""
    from kognis_sdk.generated import protocol_pb2
    
    plugin_id = "test_pulse_plugin"
    # Create plugin with a short pulse interval
    plugin = await mock_plugin_factory(plugin_id=plugin_id)
    
    # Poll for up to 5 seconds to avoid race conditions
    # SDK HealthPulseEmitter emits immediately on start(), then every interval.
    stub = plugin.control_plane._stub
    plugin_info = None
    
    for _ in range(50):
        response = await stub.ListPlugins(protocol_pb2.ListPluginsRequest())
        plugin_info = next((p for p in response.plugins if p.id == plugin_id), None)
        if plugin_info is not None and plugin_info.state == "HEALTHY_ACTIVE":
            break
        await asyncio.sleep(0.1)
    
    assert plugin_info is not None, f"Plugin {plugin_id} not found in registry"
    assert plugin_info.state == "HEALTHY_ACTIVE"
    
    # Verify HealthCheck as well
    health_resp = await stub.HealthCheck(protocol_pb2.HealthCheckRequest(plugin_id=plugin_id))
    assert health_resp.status == "HEALTHY"
    assert health_resp.state == "HEALTHY_ACTIVE"
