"""Stateful agent base class for continuous-loop plugins.

Implements SPEC 02 handler_mode=stateful_agent and SPEC 08 lifecycle
with continuous cognition loop support.
"""

from __future__ import annotations

import asyncio
import logging
from typing import Any, Awaitable, Callable

from kognis_sdk.control_plane import ControlPlaneClient, PluginState
from kognis_sdk.envelope import Envelope, create_envelope
from kognis_sdk.eventbus import EventBusClient
from kognis_sdk.manifest import Manifest
from kognis_sdk.plugin import PluginConfig, PluginError

logger = logging.getLogger("kognis_sdk")


class StatefulAgent:
    """Base class for stateful Kognis agent plugins.

    Spec reference: docs/spec/02-plugin-manifest.md (handler_mode=stateful_agent)
    Spec reference: docs/spec/08-plugin-lifecycle.md

    Stateful agents run a continuous cognition loop — they think
    even when no one is interacting with them. They maintain persistent
    state and process it across iterations.

    Unlike Plugin (stateless), StatefulAgent:
    - Has a continuous main loop (cognition_cycle)
    - Maintains working memory between iterations
    - Supports sleep/wake transitions
    - Can compact working memory in long sessions (SPEC 10 Section 10.5)
    """

    def __init__(self, manifest: Manifest, config: PluginConfig | None = None) -> None:
        self.manifest = manifest
        self.config = config or PluginConfig()
        self.control_plane = ControlPlaneClient(socket_path=self.config.socket_path)
        self.event_bus = EventBusClient()
        self._running = False
        self._state = PluginState.UNREGISTERED
        self._working_memory: dict[str, Any] = {}
        self._cycle_count = 0
        self._slot_handlers: dict[str, Callable[[Envelope], Awaitable[Envelope]]] = {}

    @property
    def plugin_id(self) -> str:
        return self.manifest.plugin_id

    @property
    def state(self) -> PluginState:
        return self._state

    @property
    def is_running(self) -> bool:
        return self._running

    @property
    def working_memory(self) -> dict[str, Any]:
        return self._working_memory

    def register_slot_handler(
        self, slot: str, handler: Callable[[Envelope], Awaitable[Envelope]]
    ) -> None:
        """Register a handler for pipeline dispatches."""
        self._slot_handlers[slot] = handler

    async def cognition_cycle(self) -> None:
        """One iteration of the continuous cognition loop.

        Override this to implement the agent's thinking process.
        Called repeatedly while the agent is HEALTHY_ACTIVE.

        The cycle should:
        1. Process any pending dispatches
        2. Update working memory
        3. Optionally compact working memory if session is long
        4. Emit state changes via event bus
        """
        self._cycle_count += 1
        await asyncio.sleep(0.1)

    async def on_startup(self) -> None:
        """Called after registration handshake completes."""
        pass

    async def on_shutdown(self) -> None:
        """Called during graceful shutdown."""
        pass

    async def on_wake(self) -> None:
        """Called when transitioning from sleep to wake."""
        pass

    async def on_sleep(self) -> None:
        """Called when transitioning from sleep mode."""
        pass

    async def on_health_check(self) -> dict[str, Any]:
        """Return health metrics for heartbeat."""
        return {
            "status": "HEALTHY",
            "cycle_count": self._cycle_count,
            "working_memory_size": len(self._working_memory),
        }

    async def on_dispatch(self, envelope: Envelope) -> Envelope:
        """Handle a pipeline dispatch.

        Override for custom dispatch handling. Default implementation
        looks up registered slot handlers.
        """
        slot = envelope.routing.current_stage
        if slot and slot in self._slot_handlers:
            return await self._slot_handlers[slot](envelope)
        return envelope

    async def compact_working_memory(self) -> None:
        """Compact working memory for long sessions.

        Spec reference: SPEC 10 Section 10.5

        Take reasoning-so-far → summarize to half size → continue.
        Mirrors human memory fade during sustained thought.
        """
        keys = list(self._working_memory.keys())
        if len(keys) <= 2:
            return

        keys_to_remove = keys[len(keys) // 2:]
        for key in keys_to_remove:
            del self._working_memory[key]

        logger.debug(
            "Compacted working memory: %d → %d keys",
            len(keys), len(self._working_memory),
        )

    async def start(self) -> None:
        """Full startup sequence."""
        import os

        await self.control_plane.connect()

        ack = await self.control_plane.register(self.manifest, pid=os.getpid())

        await self.event_bus.connect(token=ack.event_bus_token)

        subscribed_topics = [sub.topic for sub in self.manifest.event_subscriptions]
        await self.control_plane.send_ready(subscribed_topics=subscribed_topics)
        self._state = PluginState.HEALTHY_ACTIVE
        self._running = True

        await self.on_startup()
        logger.info("StatefulAgent %s started", self.plugin_id)

    async def run(self) -> None:
        """Main continuous cognition loop.

        Runs cognition_cycle() repeatedly until stopped.
        Compacts working memory every 100 cycles.
        """
        await self.start()
        try:
            while self._running:
                await self.cognition_cycle()

                if self._cycle_count % 100 == 0:
                    await self.compact_working_memory()

        except asyncio.CancelledError:
            pass
        finally:
            await self.stop()

    async def stop(self) -> None:
        """Graceful shutdown."""
        self._running = False
        self._state = PluginState.SHUTTING_DOWN

        await self.on_shutdown()
        await self.control_plane.shutdown()
        await self.event_bus.close()

        self._state = PluginState.SHUT_DOWN
        logger.info("StatefulAgent %s stopped", self.plugin_id)

    async def emit_event(self, topic: str, data: dict[str, Any]) -> None:
        """Publish an event to the event bus."""
        if not self.event_bus.is_connected:
            raise PluginError("not_connected", "Event bus not connected")
        await self.event_bus.publish(topic, data)