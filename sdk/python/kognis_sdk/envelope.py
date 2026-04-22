"""Message envelope handling.

Implements SPEC 01: Message Envelope. Provides creation, validation,
and manipulation of the universal message envelope format.
"""

from __future__ import annotations

import uuid
from dataclasses import dataclass, field
from datetime import UTC, datetime
from typing import Any

# Spec-defined constants
MAX_HOP_COUNT = 20
MAX_REVISION_COUNT = 3


@dataclass
class RoutingInfo:
    """Envelope routing state."""

    pipeline: str
    completed_stages: list[str] = field(default_factory=list)
    current_stage: str | None = None
    failed_stages: list[dict[str, str]] = field(default_factory=list)
    hop_count: int = 0
    entry_slot: str = ""

    def to_dict(self) -> dict[str, Any]:
        return {
            "pipeline": self.pipeline,
            "completed_stages": self.completed_stages,
            "current_stage": self.current_stage,
            "failed_stages": self.failed_stages,
            "hop_count": self.hop_count,
            "entry_slot": self.entry_slot,
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> RoutingInfo:
        return cls(
            pipeline=data["pipeline"],
            completed_stages=data.get("completed_stages", []),
            current_stage=data.get("current_stage"),
            failed_stages=data.get("failed_stages", []),
            hop_count=data.get("hop_count", 0),
            entry_slot=data.get("entry_slot", ""),
        )


@dataclass
class EnvelopeMetadata:
    """Envelope metadata fields."""

    priority: str = "tier_3_normal"
    trust_level: str = "internal"
    trace_id: str = ""
    revision_count: int = 0
    parent_envelope_id: str | None = None
    correlation_id: str | None = None

    def to_dict(self) -> dict[str, Any]:
        return {
            "priority": self.priority,
            "trust_level": self.trust_level,
            "trace_id": self.trace_id,
            "revision_count": self.revision_count,
            "parent_envelope_id": self.parent_envelope_id,
            "correlation_id": self.correlation_id,
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> EnvelopeMetadata:
        return cls(
            priority=data.get("priority", "tier_3_normal"),
            trust_level=data.get("trust_level", "internal"),
            trace_id=data.get("trace_id", ""),
            revision_count=data.get("revision_count", 0),
            parent_envelope_id=data.get("parent_envelope_id"),
            correlation_id=data.get("correlation_id"),
        )


class EnvelopeError(Exception):
    """Raised when envelope validation or constraints are violated."""

    def __init__(self, code: str, message: str) -> None:
        self.code = code
        self.message = message
        super().__init__(f"[{code}] {message}")


@dataclass
class Envelope:
    """Universal message envelope for Kognis pipeline system.

    Spec reference: docs/spec/01-message-envelope.md

    Envelopes are immutable once dispatched. Plugins produce new envelopes
    rather than modifying existing ones.
    """

    id: str
    created_at: str
    origin_plugin: str
    message_type: str
    payload: dict[str, Any]
    routing: RoutingInfo
    metadata: EnvelopeMetadata
    enrichments: dict[str, Any] = field(default_factory=dict)
    envelope_version: int = 1

    def to_dict(self) -> dict[str, Any]:
        return {
            "envelope_version": self.envelope_version,
            "id": self.id,
            "created_at": self.created_at,
            "origin_plugin": self.origin_plugin,
            "message_type": self.message_type,
            "payload": self.payload,
            "routing": self.routing.to_dict(),
            "enrichments": self.enrichments,
            "metadata": self.metadata.to_dict(),
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> Envelope:
        return cls(
            envelope_version=data.get("envelope_version", 1),
            id=data["id"],
            created_at=data["created_at"],
            origin_plugin=data["origin_plugin"],
            message_type=data["message_type"],
            payload=data.get("payload", {}),
            routing=RoutingInfo.from_dict(data["routing"]),
            enrichments=data.get("enrichments", {}),
            metadata=EnvelopeMetadata.from_dict(data.get("metadata", {})),
        )

    def with_enrichment(self, namespace: str, data: dict[str, Any]) -> Envelope:
        """Create a new envelope with enrichment data added.

        Enrichments are additive — adds to the specified namespace only.
        Does not modify the original envelope.
        """
        new_enrichments = {**self.enrichments, namespace: data}
        return Envelope(
            envelope_version=self.envelope_version,
            id=self.id,
            created_at=self.created_at,
            origin_plugin=self.origin_plugin,
            message_type=self.message_type,
            payload=self.payload,
            routing=self.routing,
            enrichments=new_enrichments,
            metadata=self.metadata,
        )

    def with_hop_increment(self) -> Envelope:
        """Create a new envelope with hop_count incremented.

        Raises EnvelopeError if hop_count would exceed MAX_HOP_COUNT.
        """
        new_count = self.routing.hop_count + 1
        if new_count > MAX_HOP_COUNT:
            raise EnvelopeError(
                "loop_detected", f"hop_count {new_count} exceeds max {MAX_HOP_COUNT}"
            )

        new_routing = RoutingInfo(
            pipeline=self.routing.pipeline,
            completed_stages=self.routing.completed_stages,
            current_stage=self.routing.current_stage,
            failed_stages=self.routing.failed_stages,
            hop_count=new_count,
            entry_slot=self.routing.entry_slot,
        )
        return Envelope(
            envelope_version=self.envelope_version,
            id=self.id,
            created_at=self.created_at,
            origin_plugin=self.origin_plugin,
            message_type=self.message_type,
            payload=self.payload,
            routing=new_routing,
            enrichments=self.enrichments,
            metadata=self.metadata,
        )

    def with_completed_stage(self, stage: str) -> Envelope:
        """Create a new envelope marking a stage as completed."""
        new_completed = [*self.routing.completed_stages, stage]
        new_routing = RoutingInfo(
            pipeline=self.routing.pipeline,
            completed_stages=new_completed,
            current_stage=None,
            failed_stages=self.routing.failed_stages,
            hop_count=self.routing.hop_count,
            entry_slot=self.routing.entry_slot,
        )
        return Envelope(
            envelope_version=self.envelope_version,
            id=self.id,
            created_at=self.created_at,
            origin_plugin=self.origin_plugin,
            message_type=self.message_type,
            payload=self.payload,
            routing=new_routing,
            enrichments=self.enrichments,
            metadata=self.metadata,
        )

    def with_failed_stage(self, stage: str, reason: str) -> Envelope:
        """Create a new envelope recording a failed stage."""
        new_failed = [*self.routing.failed_stages, {"stage": stage, "reason": reason}]
        new_routing = RoutingInfo(
            pipeline=self.routing.pipeline,
            completed_stages=self.routing.completed_stages,
            current_stage=None,
            failed_stages=new_failed,
            hop_count=self.routing.hop_count,
            entry_slot=self.routing.entry_slot,
        )
        return Envelope(
            envelope_version=self.envelope_version,
            id=self.id,
            created_at=self.created_at,
            origin_plugin=self.origin_plugin,
            message_type=self.message_type,
            payload=self.payload,
            routing=new_routing,
            enrichments=self.enrichments,
            metadata=self.metadata,
        )

    def with_revision(self) -> Envelope:
        """Create a new envelope with revision_count incremented.

        Only used by action_review slot. Max 3 revisions.
        """
        new_count = self.metadata.revision_count + 1
        if new_count > MAX_REVISION_COUNT:
            raise EnvelopeError(
                "max_revisions_exceeded",
                f"revision_count {new_count} exceeds max {MAX_REVISION_COUNT}",
            )

        new_metadata = EnvelopeMetadata(
            priority=self.metadata.priority,
            trust_level=self.metadata.trust_level,
            trace_id=self.metadata.trace_id,
            revision_count=new_count,
            parent_envelope_id=self.metadata.parent_envelope_id,
            correlation_id=self.metadata.correlation_id,
        )
        return Envelope(
            envelope_version=self.envelope_version,
            id=self.id,
            created_at=self.created_at,
            origin_plugin=self.origin_plugin,
            message_type=self.message_type,
            payload=self.payload,
            routing=self.routing,
            enrichments=self.enrichments,
            metadata=new_metadata,
        )

    def derive(self, message_type: str, payload: dict[str, Any]) -> Envelope:
        """Create a derived envelope from this one.

        Sets parent_envelope_id and generates a new ID.
        """
        now = datetime.now(UTC).isoformat()
        return Envelope(
            envelope_version=self.envelope_version,
            id=str(uuid.uuid4()),
            created_at=now,
            origin_plugin=self.origin_plugin,
            message_type=message_type,
            payload=payload,
            routing=RoutingInfo(pipeline=self.routing.pipeline),
            enrichments={},
            metadata=EnvelopeMetadata(
                priority=self.metadata.priority,
                trust_level=self.metadata.trust_level,
                trace_id=self.metadata.trace_id,
                revision_count=0,
                parent_envelope_id=self.id,
                correlation_id=self.metadata.correlation_id,
            ),
        )


def create_envelope(
    origin_plugin: str,
    message_type: str,
    payload: dict[str, Any],
    pipeline: str,
    entry_slot: str = "",
    priority: str = "tier_3_normal",
    trust_level: str = "internal",
) -> Envelope:
    """Create a new message envelope.

    Convenience function for envelope creation with defaults.
    """
    now = datetime.now(UTC).isoformat()
    return Envelope(
        id=str(uuid.uuid4()),
        created_at=now,
        origin_plugin=origin_plugin,
        message_type=message_type,
        payload=payload,
        routing=RoutingInfo(pipeline=pipeline, entry_slot=entry_slot),
        metadata=EnvelopeMetadata(
            priority=priority,
            trust_level=trust_level,
            trace_id=str(uuid.uuid4()),
        ),
    )


def validate_envelope(envelope: Envelope) -> list[str]:
    """Validate an envelope and return list of error strings."""
    errors: list[str] = []

    if not envelope.id:
        errors.append("id is required")
    if not envelope.created_at:
        errors.append("created_at is required")
    if not envelope.origin_plugin:
        errors.append("origin_plugin is required")
    if not envelope.message_type:
        errors.append("message_type is required")
    if not envelope.routing.pipeline:
        errors.append("routing.pipeline is required")
    if envelope.routing.hop_count < 0:
        errors.append("hop_count must be non-negative")
    if envelope.metadata.priority not in ("tier_1_immediate", "tier_2_elevated", "tier_3_normal"):
        errors.append(f"Invalid priority: {envelope.metadata.priority}")
    valid_trust = ("tier_1_creator", "tier_2_trusted", "tier_3_external", "internal")
    if envelope.metadata.trust_level not in valid_trust:
        errors.append(f"Invalid trust_level: {envelope.metadata.trust_level}")
    if envelope.metadata.revision_count < 0:
        errors.append("revision_count must be non-negative")
    if envelope.metadata.revision_count > MAX_REVISION_COUNT:
        errors.append(f"revision_count exceeds max {MAX_REVISION_COUNT}")
    if envelope.routing.hop_count > MAX_HOP_COUNT:
        errors.append(f"hop_count exceeds max {MAX_HOP_COUNT}")

    return errors
