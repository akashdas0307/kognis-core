"""Health pulse emitter and state broadcast for plugin reporting.

Implements SPEC 18: Health Pulse Schema and SPEC 06: State Broadcast.
"""

from __future__ import annotations

import asyncio
import contextlib
import logging
from dataclasses import dataclass, field
from datetime import UTC, datetime
from typing import Any

from kognis_sdk.eventbus import EventBusClient, make_state_topic

logger = logging.getLogger("kognis_sdk")


# Health statuses from SPEC 18
HEALTHY = "HEALTHY"
DEGRADED = "DEGRADED"
ERROR = "ERROR"
CRITICAL = "CRITICAL"
UNRESPONSIVE = "UNRESPONSIVE"

VALID_STATUSES = (HEALTHY, DEGRADED, ERROR, CRITICAL, UNRESPONSIVE)


@dataclass
class HealthPulse:
    """Health pulse message per SPEC 18.

    Emitted periodically by plugins to report technical health.
    """
    plugin_id: str
    timestamp: str
    status: str
    metrics: dict[str, Any] = field(default_factory=dict)
    current_activity: str = ""
    last_dispatch_at: str | None = None
    alerts: list[dict[str, str]] = field(default_factory=list)

    def to_dict(self) -> dict[str, Any]:
        return {
            "plugin_id": self.plugin_id,
            "timestamp": self.timestamp,
            "status": self.status,
            "metrics": self.metrics,
            "current_activity": self.current_activity,
            "last_dispatch_at": self.last_dispatch_at,
            "alerts": self.alerts,
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> HealthPulse:
        hp = data.get("health_pulse", data)
        if hp is None:
            hp = data
        return cls(
            plugin_id=hp.get("plugin_id", ""),
            timestamp=hp.get("timestamp", ""),
            status=hp.get("status", HEALTHY),
            metrics=hp.get("metrics", {}),
            current_activity=hp.get("current_activity", ""),
            last_dispatch_at=hp.get("last_dispatch_at"),
            alerts=hp.get("alerts", []),
        )

    def add_alert(self, severity: str, code: str, message: str) -> None:
        """Add a health alert."""
        self.alerts.append({"severity": severity, "code": code, "message": message})


class HealthPulseEmitter:
    """Periodic health pulse emitter.

    Spec reference: docs/spec/18-health-pulse-schema.md

    Emits health pulses at configurable intervals. Core daemon
    aggregates these into the Health Registry.
    """

    def __init__(
        self,
        plugin_id: str,
        event_bus: EventBusClient,
        interval_seconds: int = 10,
    ) -> None:
        self.plugin_id = plugin_id
        self.event_bus = event_bus
        self.interval_seconds = interval_seconds
        self._status = HEALTHY
        self._metrics: dict[str, Any] = {}
        self._current_activity = ""
        self._last_dispatch_at: str | None = None
        self._alerts: list[dict[str, str]] = []
        self._task: asyncio.Task | None = None
        self._running = False

    @property
    def status(self) -> str:
        return self._status

    def set_status(self, status: str) -> None:
        """Update the plugin's health status."""
        if status not in VALID_STATUSES:
            raise ValueError(f"Invalid health status: {status}")
        self._status = status

    def set_metrics(self, metrics: dict[str, Any]) -> None:
        """Update health metrics."""
        self._metrics.update(metrics)

    def set_activity(self, activity: str) -> None:
        """Update current activity description."""
        self._current_activity = activity

    def record_dispatch(self) -> None:
        """Record that a dispatch was just received."""
        self._last_dispatch_at = datetime.now(UTC).isoformat()

    def add_alert(self, severity: str, code: str, message: str) -> None:
        """Add a health alert to the next pulse."""
        self._alerts.append({"severity": severity, "code": code, "message": message})

    def build_pulse(self) -> HealthPulse:
        """Build a health pulse from current state."""
        now = datetime.now(UTC).isoformat()
        pulse = HealthPulse(
            plugin_id=self.plugin_id,
            timestamp=now,
            status=self._status,
            metrics=dict(self._metrics),
            current_activity=self._current_activity,
            last_dispatch_at=self._last_dispatch_at,
            alerts=list(self._alerts),
        )
        self._alerts.clear()
        return pulse

    async def emit(self) -> HealthPulse:
        """Emit a single health pulse to the event bus."""
        pulse = self.build_pulse()
        topic = f"kognis.health.{self.plugin_id}"
        await self.event_bus.publish(topic, pulse.to_dict())
        return pulse

    async def start(self) -> None:
        """Start periodic health pulse emission."""
        self._running = True
        self._task = asyncio.create_task(self._emit_loop())

    async def _emit_loop(self) -> None:
        """Background loop for periodic emission."""
        while self._running:
            try:
                await self.emit()
            except Exception:
                logger.exception("Health pulse emission failed for %s", self.plugin_id)
            await asyncio.sleep(self.interval_seconds)

    async def stop(self) -> None:
        """Stop periodic emission."""
        self._running = False
        if self._task and not self._task.done():
            self._task.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await self._task
        self._task = None


@dataclass
class StateChange:
    """A state change event per SPEC 06."""
    plugin_id: str
    state_name: str
    old_value: str
    new_value: str
    timestamp: str
    source: str = ""

    def to_dict(self) -> dict[str, Any]:
        return {
            "plugin_id": self.plugin_id,
            "state_name": self.state_name,
            "old_value": self.old_value,
            "new_value": self.new_value,
            "timestamp": self.timestamp,
            "source": self.source,
        }


class StateBroadcaster:
    """Emits state change events per SPEC 06.

    Distinct from health pulses — state broadcasts carry semantic
    state information (what the plugin is doing), not technical
    health (how the plugin is running).
    """

    def __init__(self, plugin_id: str, event_bus: EventBusClient) -> None:
        self.plugin_id = plugin_id
        self.event_bus = event_bus
        self._current_states: dict[str, str] = {}

    async def broadcast_change(
        self, state_name: str, new_value: str, source: str = ""
    ) -> StateChange:
        """Broadcast a state change if value actually changed.

        Spec reference: SPEC 06 — state broadcasts are on-change only.
        """
        old_value = self._current_states.get(state_name, "")
        if old_value == new_value:
            return StateChange(
                plugin_id=self.plugin_id,
                state_name=state_name,
                old_value=old_value,
                new_value=new_value,
                timestamp=datetime.now(UTC).isoformat(),
                source=source,
            )

        change = StateChange(
            plugin_id=self.plugin_id,
            state_name=state_name,
            old_value=old_value,
            new_value=new_value,
            timestamp=datetime.now(UTC).isoformat(),
            source=source or self.plugin_id,
        )

        self._current_states[state_name] = new_value

        topic = make_state_topic(self.plugin_id, state_name)
        await self.event_bus.publish(topic, change.to_dict())
        return change

    def get_current(self, state_name: str) -> str:
        """Get the current value of a state."""
        return self._current_states.get(state_name, "")

    def get_all_states(self) -> dict[str, str]:
        """Get all current state values."""
        return dict(self._current_states)
