"""Event bus client for NATS pub/sub communication.

Implements NATS-based event bus wrapper for inter-plugin
communication via the core daemon's embedded NATS server.
"""

from __future__ import annotations

import asyncio
import json
from dataclasses import dataclass, field
from typing import Any, Callable, Awaitable
from datetime import datetime, timezone

import nats
from nats.aio.client import Client as NATSClient
from nats.aio.msg import Msg


@dataclass
class EventBusConfig:
    """Configuration for NATS connection."""
    servers: list[str] = field(default_factory=lambda: ["nats://localhost:4222"])
    token: str = ""
    name: str = ""
    reconnect_attempts: int = 5
    reconnect_wait_seconds: float = 2.0
    max_pending_messages: int = 1000


@dataclass
class Subscription:
    """An active NATS subscription."""
    topic: str
    handler: Callable[[dict[str, Any]], Awaitable[None]]
    queue_group: str = ""
    sid: int = 0


class EventBusError(Exception):
    """Raised when event bus operations fail."""

    def __init__(self, code: str, message: str) -> None:
        self.code = code
        self.message = message
        super().__init__(f"[{code}] {message}")


class EventBusClient:
    """NATS event bus client for the Kognis Plugin SDK.

    Wraps NATS pub/sub for inter-plugin communication.
    Connects using token obtained during registration handshake.
    """

    def __init__(self, config: EventBusConfig | None = None) -> None:
        self.config = config or EventBusConfig()
        self._nc: NATSClient | None = None
        self._subscriptions: dict[str, Any] = {}
        self._message_count = 0

    async def connect(self, token: str | None = None, servers: list[str] | None = None) -> None:
        """Connect to NATS server using registration token."""
        if self._nc and self._nc.is_connected:
            return

        if token:
            self.config.token = token
        if servers:
            self.config.servers = servers

        self._nc = await nats.connect(
            servers=self.config.servers,
            token=self.config.token,
            name=self.config.name,
            max_reconnect_attempts=self.config.reconnect_attempts,
            reconnect_time_wait=self.config.reconnect_wait_seconds,
        )

    async def close(self) -> None:
        """Disconnect from NATS server."""
        if self._nc:
            await self._nc.drain()
            self._nc = None
        self._subscriptions.clear()

    @property
    def is_connected(self) -> bool:
        return self._nc is not None and self._nc.is_connected

    async def publish(self, topic: str, data: dict[str, Any]) -> None:
        """Publish a message to a topic.

        Topic naming convention from SPEC 06: state.<plugin_id>.<state_name>
        """
        if not self.is_connected or self._nc is None:
            raise EventBusError("not_connected", "Event bus not connected")

        payload = json.dumps(data).encode("utf-8")
        await self._nc.publish(topic, payload)
        self._message_count += 1

    async def subscribe(
        self,
        topic: str,
        handler: Callable[[dict[str, Any]], Awaitable[None]],
        queue_group: str = "",
    ) -> Subscription:
        """Subscribe to a topic with an async handler.

        Args:
            topic: NATS topic pattern (supports wildcards).
            handler: Async callback invoked for each message.
            queue_group: Optional queue group for load balancing.

        Returns:
            Subscription object for managing the subscription.
        """
        if not self.is_connected or self._nc is None:
            raise EventBusError("not_connected", "Event bus not connected")

        async def nats_handler(msg: Msg) -> None:
            try:
                data = json.loads(msg.data.decode("utf-8"))
                await handler(data)
            except Exception as e:
                # Log error in handler
                pass

        sub = await self._nc.subscribe(topic, queue=queue_group, cb=nats_handler)
        
        sdk_sub = Subscription(topic=topic, handler=handler, queue_group=queue_group, sid=0) # sid is internal to nats-py now
        self._subscriptions[topic] = sub
        return sdk_sub

    async def unsubscribe(self, topic: str) -> None:
        """Remove a subscription by topic."""
        if topic in self._subscriptions:
            sub = self._subscriptions.pop(topic)
            await sub.unsubscribe()

    def get_subscribed_topics(self) -> list[str]:
        """Return list of currently subscribed topics."""
        return list(self._subscriptions.keys())

    async def request(
        self,
        topic: str,
        data: dict[str, Any],
        timeout: float = 5.0,
    ) -> dict[str, Any]:
        """Send a request and wait for a response (request-reply pattern).

        Used for capability queries and other synchronous interactions.
        """
        if not self.is_connected or self._nc is None:
            raise EventBusError("not_connected", "Event bus not connected")

        payload = json.dumps(data).encode("utf-8")
        msg = await self._nc.request(topic, payload, timeout=timeout)
        return json.loads(msg.data.decode("utf-8"))

    @property
    def message_count(self) -> int:
        return self._message_count


def make_state_topic(plugin_id: str, state_name: str) -> str:
    """Build a state broadcast topic per SPEC 06 naming convention.

    Format: state.<plugin_id>.<state_name>
    """
    return f"state.{plugin_id}.{state_name}"


def make_event_topic(plugin_id: str, event_name: str) -> str:
    """Build a plugin event topic.

    Format: event.<plugin_id>.<event_name>
    """
    return f"event.{plugin_id}.{event_name}"


def parse_topic(topic: str) -> tuple[str, str, str]:
    """Parse a topic into (kind, plugin_id, name).

    Examples:
        state.cognitive_core.activity_state → ("state", "cognitive_core", "activity_state")
        event.memory.consolidation_complete → ("event", "memory", "consolidation_complete")
    """
    parts = topic.split(".", 2)
    if len(parts) < 3:
        raise EventBusError("invalid_topic", f"Topic must have at least 3 parts: {topic}")
    return parts[0], parts[1], parts[2]