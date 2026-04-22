"""Plugin base class for stateless handler mode.

Implements SPEC 02 handler_mode=stateless and SPEC 08 lifecycle states.
Provides the base class that all Kognis plugins extend.
"""

from __future__ import annotations

import asyncio
import logging
import os
import sys
from abc import ABC
from collections.abc import Awaitable, Callable
from dataclasses import dataclass, field
from typing import Any

from kognis_sdk.control_plane import (
    ControlPlaneClient,
    DispatchMessage,
    PluginState,
)
from kognis_sdk.envelope import Envelope
from kognis_sdk.eventbus import EventBusClient
from kognis_sdk.health import HealthPulseEmitter
from kognis_sdk.manifest import Manifest

logger = logging.getLogger("kognis_sdk")


@dataclass
class PluginConfig:
    """Runtime configuration for a plugin instance."""
    socket_path: str = "/tmp/kognis.sock"
    nats_servers: list[str] = field(default_factory=lambda: ["nats://localhost:4222"])
    log_level: str = "INFO"
    custom: dict[str, Any] = field(default_factory=dict)


class PluginError(Exception):
    """Raised when plugin operations fail."""

    def __init__(self, code: str, message: str) -> None:
        self.code = code
        self.message = message
        super().__init__(f"[{code}] {message}")


class Plugin(ABC):  # noqa: B024 — intentional non-abstract base for plugin inheritance
    """Base class for stateless Kognis plugins.

    Spec reference: docs/spec/02-plugin-manifest.md (handler_mode=stateless)
    Spec reference: docs/spec/08-plugin-lifecycle.md

    Stateless plugins handle dispatches without maintaining
    persistent state between invocations. Each dispatch is
    processed independently.
    """

    def __init__(self, manifest: Manifest, config: PluginConfig | None = None) -> None:
        self.manifest = manifest
        self.config = config or PluginConfig()
        self.control_plane = ControlPlaneClient(socket_path=self.config.socket_path)
        self.event_bus = EventBusClient()
        self.health_emitter = HealthPulseEmitter(
            plugin_id=self.manifest.plugin_id,
            event_bus=self.event_bus,
            interval_seconds=getattr(self.manifest.lifecycle, "health_pulse_interval", 10),
        )
        self._slot_handlers: dict[str, Callable[[Envelope], Awaitable[Envelope]]] = {}
        self._running = False
        self._state = PluginState.UNREGISTERED

    @property
    def plugin_id(self) -> str:
        return self.manifest.plugin_id

    @property
    def state(self) -> PluginState:
        return self._state

    @property
    def is_running(self) -> bool:
        return self._running

    def register_slot_handler(
        self, slot: str, handler: Callable[[Envelope], Awaitable[Envelope]]
    ) -> None:
        """Register an async handler for a pipeline slot.

        The handler receives the incoming envelope and must return
        a result envelope. Enrichments should be added via
        envelope.with_enrichment().
        """
        self._slot_handlers[slot] = handler

    async def on_startup(self) -> None:  # noqa: B027 — hook method, intentionally not abstract
        """Called after registration handshake completes.

        Override to perform initialization logic (e.g., connect to
        external services, load models, subscribe to events).
        """
        pass

    async def on_shutdown(self) -> None:  # noqa: B027 — hook method, intentionally not abstract
        """Called during graceful shutdown.

        Override to perform cleanup logic (e.g., close connections,
        flush buffers, persist state).
        """
        pass

    async def on_health_check(self) -> dict[str, Any]:
        """Called during heartbeat. Return health metrics.

        Override to provide plugin-specific health metrics.
        """
        return {"status": "HEALTHY", "queue_depth": 0}

    async def start(self) -> None:
        """Full startup sequence: connect → register → ready → active.

        Implements the 4-step registration handshake from SPEC 04.
        """
        # Step 0: Connect to control plane
        await self.control_plane.connect()

        # Step 1: Register
        entrypoint = f"{sys.executable} {sys.argv[0]}"
        ack = await self.control_plane.register(
            self.manifest,
            pid=os.getpid(),
            entrypoint=entrypoint
        )

        # Step 2: Connect NATS (Event Bus)
        await self.event_bus.connect(token=ack.event_bus_token)

        # Register slot handlers
        for slot_reg in self.manifest.slot_registrations:
            if slot_reg.slot in self._slot_handlers:

                async def make_handler(
                    slot: str,
                ) -> Callable[[DispatchMessage], Awaitable[Envelope]]:
                    original = self._slot_handlers[slot]

                    async def dispatch_handler(msg: DispatchMessage) -> Envelope:
                        return await original(msg.envelope)

                    return dispatch_handler

                self.control_plane.register_dispatch_handler(
                    slot_reg.slot, await make_handler(slot_reg.slot)
                )

        subscribed_topics = []
        for sub in self.manifest.event_subscriptions:
            subscribed_topics.append(sub.topic)

        # Step 3: Send Ready
        await self.control_plane.send_ready(subscribed_topics=subscribed_topics)

        # Step 4: Start continuous Pulse loop
        await self.health_emitter.start()

        self._state = PluginState.HEALTHY_ACTIVE
        self._running = True

        await self.on_startup()
        logger.info("Plugin %s started successfully", self.plugin_id)

    async def run(self) -> None:
        """Main execution loop. Processes dispatches until stopped.

        For stateless plugins, this primarily handles heartbeats
        and waits for dispatches.
        """
        await self.start()
        try:
            while self._running:
                await asyncio.sleep(1)
        except asyncio.CancelledError:
            pass
        finally:
            await self.stop()

    async def stop(self) -> None:
        """Graceful shutdown sequence."""
        self._running = False
        self._state = PluginState.SHUTTING_DOWN

        await self.on_shutdown()
        await self.health_emitter.stop()
        await self.control_plane.shutdown()
        await self.event_bus.close()

        self._state = PluginState.SHUT_DOWN
        logger.info("Plugin %s stopped", self.plugin_id)

    async def emit_event(self, topic: str, data: dict[str, Any]) -> None:
        """Publish an event to the event bus."""
        if not self.event_bus.is_connected:
            raise PluginError("not_connected", "Event bus not connected")
        await self.event_bus.publish(topic, data)
