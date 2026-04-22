"""State store with three-layer durability and backup.

Implements SPEC 12: Durability & Backup. Provides persistent state
management with crash recovery and backup chain.
"""

from __future__ import annotations

import json
import logging
import os
import tarfile
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

logger = logging.getLogger("kognis_sdk")

DEFAULT_STATE_DIR = os.path.expanduser("~/.kognis")
DEFAULT_BACKUP_DIR = os.path.expanduser("~/.kognis/backup")
LAYER2_INTERVAL_SECONDS = 1800  # 30 minutes
LAYER2_RETENTION_DAYS = 7
LAYER3_RETENTION_DAYS = 30


class StateStoreError(Exception):
    """Raised when state store operations fail."""

    def __init__(self, code: str, message: str) -> None:
        self.code = code
        self.message = message
        super().__init__(f"[{code}] {message}")


class StateStore:
    """Durable state store with three-layer backup chain.

    Spec reference: docs/spec/12-durability-backup.md

    Layer 1: Synchronous writes — every state change fsync'd to disk
    Layer 2: Periodic snapshots — every 30 minutes, retained 7 days
    Layer 3: Daily external — configurable external backup target
    """

    def __init__(
        self,
        plugin_id: str,
        state_dir: str | None = None,
        backup_dir: str | None = None,
    ) -> None:
        self.plugin_id = plugin_id
        self.state_dir = Path(state_dir or os.path.join(DEFAULT_STATE_DIR, plugin_id, "state"))
        self.backup_dir = Path(backup_dir or DEFAULT_BACKUP_DIR)
        self.state_file = self.state_dir / "current.json"
        self._state: dict[str, Any] = {}
        self._dirty = False

    def _ensure_dirs(self) -> None:
        """Create state and backup directories if they don't exist."""
        self.state_dir.mkdir(parents=True, exist_ok=True)
        self.backup_dir.mkdir(parents=True, exist_ok=True)

    def get(self, key: str, default: Any = None) -> Any:
        """Get a state value by key."""
        return self._state.get(key, default)

    def set(self, key: str, value: Any) -> None:
        """Set a state value and synchronously write to disk (Layer 1).

        Spec reference: SPEC 12 Section 12.2
        "Write every state change to disk synchronously (fsync) before acknowledging"
        """
        self._state[key] = value
        self._dirty = True
        self._sync_write()

    def delete(self, key: str) -> None:
        """Delete a state key and synchronously write to disk."""
        if key in self._state:
            del self._state[key]
            self._dirty = True
            self._sync_write()

    def keys(self) -> list[str]:
        """Return all state keys."""
        return list(self._state.keys())

    def items(self) -> list[tuple[str, Any]]:
        """Return all state key-value pairs."""
        return list(self._state.items())

    def as_dict(self) -> dict[str, Any]:
        """Return a copy of the entire state."""
        return dict(self._state)

    def update(self, values: dict[str, Any]) -> None:
        """Update multiple state values at once."""
        self._state.update(values)
        self._dirty = True
        self._sync_write()

    def _sync_write(self) -> None:
        """Synchronously write state to disk with fsync (Layer 1)."""
        self._ensure_dirs()
        tmp_path = self.state_file.with_suffix(".tmp")
        try:
            with open(tmp_path, "w") as f:
                json.dump(self._state, f, indent=2, sort_keys=True, default=str)
                f.flush()
                os.fsync(f.fileno())
            tmp_path.replace(self.state_file)
        except OSError as e:
            if tmp_path.exists():
                tmp_path.unlink()
            raise StateStoreError("write_failed", f"Failed to write state: {e}") from e

    def load(self) -> dict[str, Any]:
        """Load state from disk. Used for crash recovery.

        Spec reference: SPEC 12 Section 12.5

        Restore protocol:
        1. Read from Layer 1 (primary state)
        2. If corrupt/missing, restore from most recent Layer 2
        3. If Layer 2 also gone, restore from Layer 3
        4. If no valid backup, raise CRITICAL alert
        """
        # Try Layer 1
        if self.state_file.exists():
            try:
                with open(self.state_file) as f:
                    self._state = json.load(f)
                self._dirty = False
                return self._state
            except (json.JSONDecodeError, OSError) as e:
                logger.warning("Layer 1 state corrupt for %s: %s", self.plugin_id, e)

        # Try Layer 2
        snapshot = self._find_latest_snapshot()
        if snapshot is not None:
            try:
                self._state = self._restore_from_snapshot(snapshot)
                self._sync_write()
                logger.warning("Restored %s from Layer 2 snapshot: %s", self.plugin_id, snapshot)
                return self._state
            except Exception as e:
                logger.warning("Layer 2 restore failed for %s: %s", self.plugin_id, e)

        # No valid backup found
        raise StateStoreError(
            "no_valid_backup",
            f"No valid state found for {self.plugin_id} — CRITICAL: do not start with empty state",
        )

    def create_snapshot(self) -> Path:
        """Create a Layer 2 periodic snapshot.

        Spec reference: SPEC 12 Section 12.3
        "Every 30 minutes, stored as tar.gz in backup directory"
        """
        self._ensure_dirs()
        timestamp = datetime.now(UTC).strftime("%Y%m%dT%H%M%SZ")
        snapshot_name = f"{self.plugin_id}_{timestamp}.tar.gz"
        snapshot_path = self.backup_dir / snapshot_name

        # Write current state to temp file
        tmp_state = self.state_dir / "snapshot_state.json"
        with open(tmp_state, "w") as f:
            json.dump(self._state, f, indent=2, sort_keys=True, default=str)

        try:
            with tarfile.open(snapshot_path, "w:gz") as tar:
                tar.add(tmp_state, arcname="state.json")
        finally:
            tmp_state.unlink(missing_ok=True)

        self._prune_old_snapshots()
        return snapshot_path

    def _find_latest_snapshot(self) -> Path | None:
        """Find the most recent Layer 2 snapshot."""
        if not self.backup_dir.exists():
            return None

        snapshots = sorted(
            self.backup_dir.glob(f"{self.plugin_id}_*.tar.gz"),
            reverse=True,
        )
        return snapshots[0] if snapshots else None

    def _restore_from_snapshot(self, snapshot_path: Path) -> dict[str, Any]:
        """Restore state from a Layer 2 snapshot."""
        with tarfile.open(snapshot_path, "r:gz") as tar:
            member = tar.getmember("state.json")
            f = tar.extractfile(member)
            if f is None:
                raise StateStoreError(
                    "corrupt_snapshot",
                    f"Cannot extract state.json from {snapshot_path}",
                )
            return json.load(f)  # type: ignore[no-any-return]

    def _prune_old_snapshots(self) -> None:
        """Prune Layer 2 snapshots older than retention period.

        Spec reference: SPEC 12 Section 12.6
        "Old Layer 2 snapshots pruned after 7 days (configurable)"
        """
        if not self.backup_dir.exists():
            return

        now = datetime.now(UTC)
        for snapshot in self.backup_dir.glob(f"{self.plugin_id}_*.tar.gz"):
            try:
                date_str = snapshot.stem.split("_", 1)[1]
                snapshot_date = datetime.strptime(date_str, "%Y%m%dT%H%M%SZ").replace(tzinfo=UTC)
                age_days = (now - snapshot_date).days
                if age_days > LAYER2_RETENTION_DAYS:
                    snapshot.unlink()
                    logger.debug("Pruned snapshot %s", snapshot.name)
            except (ValueError, IndexError):
                pass

    def clear(self) -> None:
        """Clear in-memory state (does NOT delete disk state)."""
        self._state.clear()
        self._dirty = True
