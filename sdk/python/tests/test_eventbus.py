"""Tests for event bus client — M-004.

Validates NATS event bus wrapper.
"""

import pytest

from kognis_sdk.eventbus import (
    EventBusClient,
    EventBusConfig,
    EventBusError,
    make_event_topic,
    make_state_topic,
    parse_topic,
)


class TestEventBusConfig:
    def test_defaults(self):
        cfg = EventBusConfig()
        assert cfg.servers == ["nats://localhost:4222"]
        assert cfg.token == ""
        assert cfg.reconnect_attempts == 5


class TestEventBusClient:
    @pytest.mark.asyncio
    async def test_connect(self):
        eb = EventBusClient()
        await eb.connect(token="test_token")
        assert eb.is_connected

    @pytest.mark.asyncio
    async def test_close(self):
        eb = EventBusClient()
        await eb.connect(token="tok")
        await eb.close()
        assert not eb.is_connected

    @pytest.mark.asyncio
    async def test_publish_not_connected(self):
        eb = EventBusClient()
        with pytest.raises(EventBusError, match="not_connected"):
            await eb.publish("topic", {"data": 1})

    @pytest.mark.asyncio
    async def test_publish(self):
        eb = EventBusClient()
        await eb.connect(token="tok")
        await eb.publish("state.test.mode", {"value": "active"})
        assert eb.message_count == 1

    @pytest.mark.asyncio
    async def test_subscribe_not_connected(self):
        eb = EventBusClient()
        with pytest.raises(EventBusError, match="not_connected"):
            await eb.subscribe("topic", handler=lambda d: None)

    @pytest.mark.asyncio
    async def test_subscribe(self):
        eb = EventBusClient()
        await eb.connect(token="tok")

        async def handler(data):
            pass

        sub = await eb.subscribe("state.test.mode", handler=handler)
        assert sub.topic == "state.test.mode"
        assert "state.test.mode" in eb.get_subscribed_topics()

    @pytest.mark.asyncio
    async def test_unsubscribe(self):
        eb = EventBusClient()
        await eb.connect(token="tok")

        async def handler(data):
            pass

        await eb.subscribe("topic.a", handler=handler)
        assert "topic.a" in eb.get_subscribed_topics()
        await eb.unsubscribe("topic.a")
        assert "topic.a" not in eb.get_subscribed_topics()

    @pytest.mark.asyncio
    async def test_request_not_connected(self):
        eb = EventBusClient()
        with pytest.raises(EventBusError, match="not_connected"):
            await eb.request("topic", {"q": 1})

    @pytest.mark.asyncio
    async def test_request(self):
        eb = EventBusClient()
        await eb.connect(token="tok")
        result = await eb.request("topic", {"q": 1}, timeout=2.0)
        assert "result" in result


class TestTopicHelpers:
    def test_make_state_topic(self):
        assert make_state_topic("cognitive_core", "activity_state") == "state.cognitive_core.activity_state"

    def test_make_event_topic(self):
        assert make_event_topic("memory", "consolidation_complete") == "event.memory.consolidation_complete"

    def test_parse_state_topic(self):
        kind, plugin, name = parse_topic("state.cognitive_core.activity_state")
        assert kind == "state"
        assert plugin == "cognitive_core"
        assert name == "activity_state"

    def test_parse_event_topic(self):
        kind, plugin, name = parse_topic("event.memory.consolidation_complete")
        assert kind == "event"
        assert plugin == "memory"

    def test_parse_invalid_topic(self):
        with pytest.raises(EventBusError, match="invalid_topic"):
            parse_topic("invalid")
