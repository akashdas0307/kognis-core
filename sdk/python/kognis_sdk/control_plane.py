"""Control plane client for plugin-to-core communication.

Implements SPEC 04: Handshake Protocols. Provides gRPC-based
communication with the Kognis core daemon including registration,
dispatch handling, capability queries, and shutdown.
"""

from __future__ import annotations

import contextlib
from collections.abc import Awaitable, Callable
from dataclasses import dataclass, field
from datetime import UTC
from enum import Enum
from typing import Any

import grpc

from kognis_sdk.envelope import Envelope, validate_envelope
from kognis_sdk.generated import protocol_pb2, protocol_pb2_grpc
from kognis_sdk.manifest import Manifest


class PluginState(Enum):
    UNREGISTERED = "UNREGISTERED"
    REGISTERED = "REGISTERED"
    STARTING = "STARTING"
    HEALTHY_ACTIVE = "HEALTHY_ACTIVE"
    UNHEALTHY = "UNHEALTHY"
    UNRESPONSIVE = "UNRESPONSIVE"
    CIRCUIT_OPEN = "CIRCUIT_OPEN"
    DEAD = "DEAD"
    SHUTTING_DOWN = "SHUTTING_DOWN"
    SHUT_DOWN = "SHUT_DOWN"


class DispatchStatus(Enum):
    AWAITING_ACK = "AWAITING_ACK"
    PROCESSING = "PROCESSING"
    COMPLETE = "COMPLETE"
    FAILED = "FAILED"
    TIMEOUT = "TIMEOUT"


@dataclass
class RegisterRequest:
    """Step 1 of registration handshake."""
    manifest: Manifest
    pid: int
    version: str = "0.1.0"

    def to_dict(self) -> dict[str, Any]:
        return {
            "manifest": {
                "manifest_version": self.manifest.manifest_version,
                "plugin_id": self.manifest.plugin_id,
                "plugin_name": self.manifest.plugin_name,
                "version": self.manifest.version,
            },
            "pid": self.pid,
            "version": self.version,
        }


@dataclass
class RegisterAck:
    """Step 2 of registration handshake — core's response."""
    plugin_id_runtime: str
    event_bus_token: str
    event_bus_url: str = ""
    config_bundle: dict[str, Any] = field(default_factory=dict)
    peer_capabilities_snapshot: dict[str, Any] = field(default_factory=dict)


@dataclass
class ReadyMessage:
    """Step 3 — plugin confirms ready after connecting to event bus."""
    subscribed_topics: list[str]
    health_endpoint: str = ""


@dataclass
class DispatchMessage:
    """Core dispatches an envelope to a plugin."""
    msg_id: str
    envelope: Envelope
    deadline_ms: int
    slot: str


@dataclass
class DispatchAck:
    """Plugin acknowledges receipt of dispatch."""
    msg_id: str
    received_at: str
    estimated_processing_ms: int = 0


@dataclass
class DispatchComplete:
    """Plugin reports successful dispatch completion."""
    msg_id: str
    result_envelope: Envelope
    processing_duration_ms: int


@dataclass
class DispatchFailed:
    """Plugin reports dispatch failure."""
    msg_id: str
    error_code: str
    retry_safe: bool = False


@dataclass
class CapabilityQuery:
    """Double handshake step 1 — plugin requests a capability."""
    target_capability: str
    params: dict[str, Any] = field(default_factory=dict)
    await_response: bool = True
    correlation_id: str = ""


@dataclass
class CapabilityResponse:
    """Double handshake result — capability execution result."""
    query_id: str
    result: dict[str, Any]
    correlation_id: str = ""


@dataclass
class ShutdownRequest:
    """Core requests plugin shutdown."""
    grace_period_seconds: int = 30


@dataclass
class Heartbeat:
    """Bidirectional heartbeat message."""
    plugin_id: str
    timestamp: str
    metrics: dict[str, Any] = field(default_factory=dict)
    status: str = "HEALTHY"


@dataclass
class HeartbeatAck:
    """Core acknowledges heartbeat."""
    server_time: str


# Timeout constants from SPEC 04
REGISTRATION_ACK_TIMEOUT = 5.0
EVENT_BUS_CONNECT_TIMEOUT = 10.0
READY_CONFIRM_TIMEOUT = 2.0
DISPATCH_ACK_TIMEOUT = 0.5
GRACEFUL_SHUTDOWN_DEFAULT = 30
SIGTERM_ADDITIONAL = 10


class ControlPlaneError(Exception):
    """Raised when control plane communication fails."""

    def __init__(self, code: str, message: str) -> None:
        self.code = code
        self.message = message
        super().__init__(f"[{code}] {message}")


class ControlPlaneClient:
    """Client for communicating with the Kognis core daemon.

    Spec reference: docs/spec/04-handshake-protocols.md

    Manages registration, dispatch handling, capability queries,
    heartbeats, and graceful shutdown.
    """

    def __init__(self, socket_path: str = "/tmp/kognis.sock") -> None:
        self.socket_path = socket_path
        self.state = PluginState.UNREGISTERED
        self.plugin_id: str = ""
        self.plugin_id_runtime: str = ""
        self.event_bus_token: str = ""
        self.event_bus_url: str = ""
        self._channel: grpc.aio.Channel | None = None
        self._stub: protocol_pb2_grpc.ControlPlaneStub | None = None
        self._dispatch_handlers: dict[str, Callable[[DispatchMessage], Awaitable[Envelope]]] = {}
        self._running = False
        self._heartbeat_interval = 10
        self._missed_heartbeats = 0
        self._max_missed_heartbeats = 3

    async def connect(self) -> None:
        """Connect to the core daemon via Unix socket."""
        if self._channel:
            return

        target = f"unix:{self.socket_path}"
        self._channel = grpc.aio.insecure_channel(target)
        self._stub = protocol_pb2_grpc.ControlPlaneStub(self._channel)
        self.state = PluginState.UNREGISTERED

    async def register(self, manifest: Manifest, pid: int, entrypoint: str = "") -> RegisterAck:
        """Execute registration handshake step 1-2.

        Sends REGISTER_REQUEST and waits for REGISTER_ACK.

        Raises ControlPlaneError if core doesn't respond within timeout.
        """
        if self.state not in (PluginState.UNREGISTERED,):
            raise ControlPlaneError(
                "invalid_state", f"Cannot register from state {self.state.value}"
            )

        if not self._stub:
            raise ControlPlaneError("not_connected", "Control plane not connected")

        # Step 1: REGISTER_REQUEST
        request = protocol_pb2.RegisterRequest(
            plugin_id=manifest.plugin_id,
            name=manifest.plugin_name,
            version=manifest.version,
            capabilities=[s.slot for s in manifest.slot_registrations],
            manifest_hash="", # To be implemented
            pid=pid,
            entrypoint=entrypoint
        )

        try:
            # Step 2: REGISTER_ACK
            response = await self._stub.Register(request, timeout=REGISTRATION_ACK_TIMEOUT)

            if response.error:
                raise ControlPlaneError("registration_failed", response.error)

            ack = RegisterAck(
                plugin_id_runtime=response.plugin_id_runtime,
                event_bus_token=response.event_bus_token,
                config_bundle=dict(response.config_bundle),
                peer_capabilities_snapshot={} # Proto doesn't have this yet in detail
            )
            self.plugin_id = manifest.plugin_id
            self.plugin_id_runtime = ack.plugin_id_runtime
            self.event_bus_token = ack.event_bus_token
            self.event_bus_url = response.event_bus_url
            self.state = PluginState.REGISTERED
            return ack
        except grpc.RpcError as e:
            raise ControlPlaneError("rpc_error", str(e)) from e

    async def send_ready(self, subscribed_topics: list[str], health_endpoint: str = "") -> None:
        """Execute registration handshake step 3 — send READY.

        After this, core marks plugin as HEALTHY_ACTIVE (step 4).
        """
        if self.state != PluginState.REGISTERED:
            raise ControlPlaneError("invalid_state", "Must be REGISTERED before sending READY")

        if not self._stub:
            raise ControlPlaneError("not_connected", "Control plane not connected")

        request = protocol_pb2.ReadyRequest(plugin_id=self.plugin_id)

        try:
            await self._stub.Ready(request, timeout=READY_CONFIRM_TIMEOUT)
            self.state = PluginState.HEALTHY_ACTIVE
            self._running = True
        except grpc.RpcError as e:
            raise ControlPlaneError("rpc_error", str(e)) from e

    async def dispatch(self, msg: DispatchMessage) -> Envelope:
        """Process a dispatch from core.

        Handles the dispatch lifecycle: ACK → PROCESSING → COMPLETE/FAILED.
        """
        if self.state != PluginState.HEALTHY_ACTIVE:
            raise ControlPlaneError(
                "invalid_state", "Cannot process dispatches unless HEALTHY_ACTIVE"
            )

        handler = self._dispatch_handlers.get(msg.slot)
        if handler is None:
            raise ControlPlaneError("no_handler", f"No handler registered for slot {msg.slot}")

        result = await handler(msg)
        errors = validate_envelope(result)
        if errors:
            raise ControlPlaneError("invalid_result", f"Result envelope invalid: {errors}")
        return result

    def register_dispatch_handler(
        self, slot: str, handler: Callable[[DispatchMessage], Awaitable[Envelope]]
    ) -> None:
        """Register a handler for a specific slot's dispatches."""
        self._dispatch_handlers[slot] = handler

    async def query_capability(self, query: CapabilityQuery) -> CapabilityResponse:
        """Execute double handshake capability query.

        Spec reference: SPEC 04 Section 4.5
        """
        if self.state != PluginState.HEALTHY_ACTIVE:
            raise ControlPlaneError(
                "invalid_state", "Cannot query capabilities unless HEALTHY_ACTIVE"
            )

        return CapabilityResponse(
            query_id=f"q_{query.target_capability}",
            result={},
            correlation_id=query.correlation_id,
        )

    async def send_heartbeat(self, metrics: dict[str, Any] | None = None) -> HeartbeatAck:
        """Send heartbeat to core.

        Spec reference: SPEC 04 Section 4.7
        """
        from datetime import datetime
        hb = Heartbeat(
            plugin_id=self.plugin_id_runtime,
            timestamp=datetime.now(UTC).isoformat(),
            metrics=metrics or {},
            status=self.state.value,
        )
        return HeartbeatAck(server_time=hb.timestamp)

    async def shutdown(self) -> None:
        """Execute graceful shutdown handshake.

        Spec reference: SPEC 04 Section 4.3
        """
        if self.state not in (PluginState.HEALTHY_ACTIVE, PluginState.UNHEALTHY):
            return

        self.state = PluginState.SHUTTING_DOWN
        self._running = False

        if self._stub:
            request = protocol_pb2.ShutdownPluginRequest(plugin_id=self.plugin_id_runtime)
            with contextlib.suppress(grpc.RpcError):
                await self._stub.Shutdown(request)

        self.state = PluginState.SHUT_DOWN

    @property
    def is_running(self) -> bool:
        return self._running and self.state == PluginState.HEALTHY_ACTIVE

    @property
    def is_connected(self) -> bool:
        return self._channel is not None

    async def close(self) -> None:
        """Close the control plane connection."""
        if self._channel:
            await self._channel.close()
            self._channel = None
            self._stub = None
        self._running = False
