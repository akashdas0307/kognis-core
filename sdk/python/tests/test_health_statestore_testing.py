"""Tests for health pulse, state broadcast, state store, and testing harness.

M-010, M-011, M-012.
"""

import json
import os
import pytest
import tarfile
from pathlib import Path

from kognis_sdk.health import (
    HealthPulse,
    HealthPulseEmitter,
    StateBroadcaster,
    StateChange,
    HEALTHY,
    DEGRADED,
    ERROR,
    CRITICAL,
    UNRESPONSIVE,
)
from kognis_sdk.eventbus import EventBusClient
from kognis_sdk.state_store import StateStore, StateStoreError
from kognis_sdk.testing import TestCore
from kognis_sdk.plugin import Plugin
from kognis_sdk.stateful_agent import StatefulAgent
from kognis_sdk.envelope import Envelope, create_envelope
from kognis_sdk.manifest import Manifest, SlotRegistration


class TestHealthPulse:
    def test_creation(self):
        hp = HealthPulse(plugin_id="p1", timestamp="2025-01-01T00:00:00Z", status=HEALTHY)
        assert hp.status == HEALTHY
        assert hp.metrics == {}
        assert hp.alerts == []

    def test_to_dict(self):
        hp = HealthPulse(plugin_id="p1", timestamp="t", status=DEGRADED)
        d = hp.to_dict()
        assert "health_pulse" in d
        assert d["health_pulse"]["status"] == DEGRADED

    def test_from_dict(self):
        d = {"health_pulse": {"plugin_id": "p1", "timestamp": "t", "status": ERROR, "metrics": {"x": 1}}}
        hp = HealthPulse.from_dict(d)
        assert hp.status == ERROR
        assert hp.metrics == {"x": 1}

    def test_add_alert(self):
        hp = HealthPulse(plugin_id="p1", timestamp="t", status=CRITICAL)
        hp.add_alert("error", "KGN-TEST-001", "test alert")
        assert len(hp.alerts) == 1
        assert hp.alerts[0]["severity"] == "error"

    def test_valid_statuses(self):
        for s in (HEALTHY, DEGRADED, ERROR, CRITICAL, UNRESPONSIVE):
            hp = HealthPulse(plugin_id="p1", timestamp="t", status=s)
            assert hp.status == s


class TestHealthPulseEmitter:
    @pytest.mark.asyncio
    async def test_creation(self):
        eb = EventBusClient()
        emitter = HealthPulseEmitter(plugin_id="p1", event_bus=eb)
        assert emitter.status == HEALTHY

    @pytest.mark.asyncio
    async def test_set_status(self):
        eb = EventBusClient()
        emitter = HealthPulseEmitter(plugin_id="p1", event_bus=eb)
        emitter.set_status(DEGRADED)
        assert emitter.status == DEGRADED

    @pytest.mark.asyncio
    async def test_invalid_status(self):
        eb = EventBusClient()
        emitter = HealthPulseEmitter(plugin_id="p1", event_bus=eb)
        with pytest.raises(ValueError, match="Invalid health status"):
            emitter.set_status("BOGUS")

    @pytest.mark.asyncio
    async def test_build_pulse(self):
        eb = EventBusClient()
        emitter = HealthPulseEmitter(plugin_id="p1", event_bus=eb)
        emitter.set_metrics({"queue_depth": 5})
        emitter.set_activity("processing")
        pulse = emitter.build_pulse()
        assert pulse.status == HEALTHY
        assert pulse.metrics["queue_depth"] == 5
        assert pulse.current_activity == "processing"

    @pytest.mark.asyncio
    async def test_emit(self):
        eb = EventBusClient()
        await eb.connect(token="tok")
        emitter = HealthPulseEmitter(plugin_id="p1", event_bus=eb)
        pulse = await emitter.emit()
        assert pulse.plugin_id == "p1"


class TestStateBroadcaster:
    @pytest.mark.asyncio
    async def test_broadcast_change(self):
        eb = EventBusClient()
        await eb.connect(token="tok")
        broadcaster = StateBroadcaster(plugin_id="p1", event_bus=eb)
        change = await broadcaster.broadcast_change("mode", "active")
        assert change.old_value == ""
        assert change.new_value == "active"

    @pytest.mark.asyncio
    async def test_no_change_same_value(self):
        eb = EventBusClient()
        await eb.connect(token="tok")
        broadcaster = StateBroadcaster(plugin_id="p1", event_bus=eb)
        await broadcaster.broadcast_change("mode", "active")
        change = await broadcaster.broadcast_change("mode", "active")
        # No actual change, but still returns a StateChange
        assert change.new_value == "active"

    @pytest.mark.asyncio
    async def test_get_current(self):
        eb = EventBusClient()
        await eb.connect(token="tok")
        broadcaster = StateBroadcaster(plugin_id="p1", event_bus=eb)
        await broadcaster.broadcast_change("mode", "idle")
        assert broadcaster.get_current("mode") == "idle"

    @pytest.mark.asyncio
    async def test_get_all_states(self):
        eb = EventBusClient()
        await eb.connect(token="tok")
        broadcaster = StateBroadcaster(plugin_id="p1", event_bus=eb)
        await broadcaster.broadcast_change("a", "1")
        await broadcaster.broadcast_change("b", "2")
        states = broadcaster.get_all_states()
        assert states == {"a": "1", "b": "2"}


class TestStateChange:
    def test_to_dict(self):
        sc = StateChange(plugin_id="p1", state_name="mode", old_value="idle", new_value="active", timestamp="t")
        d = sc.to_dict()
        assert d["plugin_id"] == "p1"
        assert d["new_value"] == "active"


class TestStateStore:
    def test_set_and_get(self, tmp_path):
        store = StateStore(plugin_id="p1", state_dir=str(tmp_path / "state"), backup_dir=str(tmp_path / "backup"))
        store.set("key1", "value1")
        assert store.get("key1") == "value1"

    def test_get_missing_key(self, tmp_path):
        store = StateStore(plugin_id="p1", state_dir=str(tmp_path / "state"), backup_dir=str(tmp_path / "backup"))
        assert store.get("missing") is None
        assert store.get("missing", "default") == "default"

    def test_delete(self, tmp_path):
        store = StateStore(plugin_id="p1", state_dir=str(tmp_path / "state"), backup_dir=str(tmp_path / "backup"))
        store.set("key1", "value1")
        store.delete("key1")
        assert store.get("key1") is None

    def test_update(self, tmp_path):
        store = StateStore(plugin_id="p1", state_dir=str(tmp_path / "state"), backup_dir=str(tmp_path / "backup"))
        store.update({"a": 1, "b": 2})
        assert store.get("a") == 1
        assert store.get("b") == 2

    def test_keys(self, tmp_path):
        store = StateStore(plugin_id="p1", state_dir=str(tmp_path / "state"), backup_dir=str(tmp_path / "backup"))
        store.update({"a": 1, "b": 2})
        assert set(store.keys()) == {"a", "b"}

    def test_as_dict(self, tmp_path):
        store = StateStore(plugin_id="p1", state_dir=str(tmp_path / "state"), backup_dir=str(tmp_path / "backup"))
        store.update({"x": 10})
        d = store.as_dict()
        assert d == {"x": 10}

    def test_persistence(self, tmp_path):
        state_dir = str(tmp_path / "state")
        store1 = StateStore(plugin_id="p1", state_dir=state_dir, backup_dir=str(tmp_path / "backup"))
        store1.set("persistent_key", "persistent_value")

        store2 = StateStore(plugin_id="p1", state_dir=state_dir, backup_dir=str(tmp_path / "backup"))
        state = store2.load()
        assert state["persistent_key"] == "persistent_value"

    def test_create_snapshot(self, tmp_path):
        store = StateStore(plugin_id="p1", state_dir=str(tmp_path / "state"), backup_dir=str(tmp_path / "backup"))
        store.update({"key": "val"})
        snapshot_path = store.create_snapshot()
        assert snapshot_path.exists()
        assert snapshot_path.name.endswith(".tar.gz")

    def test_no_valid_backup(self, tmp_path):
        store = StateStore(plugin_id="nonexistent", state_dir=str(tmp_path / "empty"), backup_dir=str(tmp_path / "b"))
        with pytest.raises(StateStoreError, match="no_valid_backup"):
            store.load()

    def test_clear(self, tmp_path):
        store = StateStore(plugin_id="p1", state_dir=str(tmp_path / "state"), backup_dir=str(tmp_path / "backup"))
        store.set("key1", "val1")
        store.clear()
        assert store.as_dict() == {}


class TestTestCore:
    def test_make_manifest(self):
        tc = TestCore()
        m = tc.make_test_manifest()
        assert m.plugin_id == "com.test.plugin"
        assert m.manifest_version == 1

    def test_make_manifest_custom(self):
        tc = TestCore()
        m = tc.make_test_manifest(plugin_id="custom", slots=[("p1", "s1"), ("p2", "s2")])
        assert m.plugin_id == "custom"
        assert len(m.slot_registrations) == 2

    def test_make_envelope(self):
        tc = TestCore()
        env = tc.make_test_envelope()
        assert env.message_type == "user_text_input"
        assert env.payload == {"text": "test input"}

    @pytest.mark.asyncio
    async def test_dispatch(self):
        tc = TestCore()
        m = tc.make_test_manifest()
        p = Plugin(manifest=m)

        async def handler(env: Envelope) -> Envelope:
            return env.with_enrichment("test", {"ok": True})

        p.register_slot_handler("s", handler)
        tc.register_plugin(p)

        env = tc.make_test_envelope()
        result = await tc.dispatch(slot="s", envelope=env)
        assert result.envelope.enrichments.get("test") == {"ok": True}
        assert result.errors == []

    @pytest.mark.asyncio
    async def test_dispatch_no_handler(self):
        tc = TestCore()
        m = tc.make_test_manifest()
        p = Plugin(manifest=m)
        tc.register_plugin(p)

        env = tc.make_test_envelope()
        result = await tc.dispatch(slot="nonexistent", envelope=env)
        assert len(result.errors) > 0

    @pytest.mark.asyncio
    async def test_dispatch_no_plugin(self):
        tc = TestCore()
        env = tc.make_test_envelope()
        result = await tc.dispatch(slot="s", envelope=env)
        assert "No plugins registered" in result.errors

    @pytest.mark.asyncio
    async def test_dispatch_count(self):
        tc = TestCore()
        m = tc.make_test_manifest()
        p = Plugin(manifest=m)

        async def handler(env: Envelope) -> Envelope:
            return env

        p.register_slot_handler("s", handler)
        tc.register_plugin(p)

        env = tc.make_test_envelope()
        await tc.dispatch(slot="s", envelope=env)
        await tc.dispatch(slot="s", envelope=env)
        assert tc.dispatch_count == 2

    @pytest.mark.asyncio
    async def test_run_cycle(self):
        tc = TestCore()
        m = tc.make_test_manifest(handler_mode="stateful_agent")
        agent = StatefulAgent(manifest=m)
        tc.register_agent(agent)
        await tc.run_cycle(agent)
        assert agent._cycle_count == 1

    def test_reset(self):
        tc = TestCore()
        tc._dispatched.append({"test": True})
        tc.reset()
        assert tc.dispatch_count == 0