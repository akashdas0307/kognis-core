import pytest
import asyncio
import json
import uuid
from kognis_sdk.manifest import SlotRegistration

@pytest.mark.asyncio
async def test_cross_plugin_capability_routing(mock_plugin_factory):
    """
    Simulate a full double-handshake capability routing flow per SPEC 04 Section 4.5:
    Plugin A (requester) -> Core -> Plugin B (provider) -> Core -> Plugin A
    """
    capability = "test_capability"
    
    # Plugin B: The Provider
    plugin_b = await mock_plugin_factory(
        plugin_id="plugin_b",
        slots=[SlotRegistration(pipeline="test_pipe", slot=capability, priority=50)]
    )
    
    # Plugin A: The Requester
    plugin_a = await mock_plugin_factory(plugin_id="plugin_a")
    
    from kognis_sdk.generated import protocol_pb2
    
    # Wait for Plugin B to be fully registered and HEALTHY_ACTIVE so its capabilities are known
    stub = plugin_b.control_plane._stub
    plugin_b_ready = False
    for _ in range(50):
        resp = await stub.ListPlugins(protocol_pb2.ListPluginsRequest())
        info = next((p for p in resp.plugins if p.id == "plugin_b"), None)
        if info and info.state == "HEALTHY_ACTIVE":
            plugin_b_ready = True
            break
        await asyncio.sleep(0.1)
        
    assert plugin_b_ready, "Plugin B did not register successfully in time"
    
    # Correlation and Query IDs
    correlation_id = str(uuid.uuid4())
    query_id = f"q_{uuid.uuid4().hex[:8]}"
    
    # We'll use a Future to wait for the response delivered to Plugin A
    response_received = asyncio.Future()
    
    # Plugin A subscribes to its response delivery topic
    async def on_response(data):
        if data.get("query_id") == query_id:
            if not response_received.done():
                response_received.set_result(data)
            
    await plugin_a.event_bus.subscribe(
        f"kognis.capability.response_delivered.plugin_a",
        handler=on_response
    )
    
    # Plugin B subscribes to its dispatch topic
    dispatch_received = asyncio.Future()
    async def on_dispatch(data):
        if data.get("query_id") == query_id:
            if not dispatch_received.done():
                dispatch_received.set_result(data)
            
            # Step 3: Plugin B sends ACK
            await plugin_b.event_bus.publish("kognis.capability.ack", {
                "query_id": query_id,
                "plugin_id": "plugin_b"
            })
            
            # Step 5: Plugin B sends RESPONSE
            await plugin_b.event_bus.publish("kognis.capability.response", {
                "query_id": query_id,
                "plugin_id": "plugin_b",
                "result": {"answer": 42}
            })
            
    await plugin_b.event_bus.subscribe(
        f"kognis.capability.dispatch.plugin_b",
        handler=on_dispatch
    )
    
    # Step 1: Plugin A sends QUERY
    await plugin_a.event_bus.publish("kognis.capability.query", {
        "query_id": query_id,
        "target_capability": capability,
        "requester_plugin_id": "plugin_a",
        "params": {"question": "life?"},
        "await_response": True,
        "correlation_id": correlation_id
    })
    
    # Wait for the flow to complete
    # Wait for dispatch to B (Step 2)
    dispatch = await asyncio.wait_for(dispatch_received, timeout=2.0)
    assert dispatch["params"] == {"question": "life?"}
    
    # Wait for response to A (Step 6)
    response = await asyncio.wait_for(response_received, timeout=2.0)
    assert response["result"] == {"answer": 42}
    assert response["correlation_id"] == correlation_id
    
    # Step 7: Plugin A sends RECEIPT_ACK
    await plugin_a.event_bus.publish("kognis.capability.receipt_ack", {
        "query_id": query_id,
        "plugin_id": "plugin_a"
    })
    
    # Give Core a moment to process RECEIPT_ACK (clean up inflight)
    await asyncio.sleep(0.1)
