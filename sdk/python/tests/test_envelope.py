"""Tests for envelope handling — M-002.

Validates SPEC 01: Message Envelope creation, validation, and manipulation.
"""

import uuid

import pytest

from kognis_sdk.envelope import (
    MAX_HOP_COUNT,
    MAX_REVISION_COUNT,
    Envelope,
    EnvelopeError,
    EnvelopeMetadata,
    RoutingInfo,
    create_envelope,
    validate_envelope,
)


def make_routing(**overrides) -> RoutingInfo:
    defaults = {"pipeline": "user_text_interaction"}
    defaults.update(overrides)
    return RoutingInfo(**defaults)


def make_metadata(**overrides) -> EnvelopeMetadata:
    defaults = {
        "priority": "tier_3_normal",
        "trust_level": "internal",
        "trace_id": str(uuid.uuid4()),
    }
    defaults.update(overrides)
    return EnvelopeMetadata(**defaults)


def make_envelope(**overrides) -> Envelope:
    defaults = {
        "id": str(uuid.uuid4()),
        "created_at": "2025-01-01T00:00:00+00:00",
        "origin_plugin": "test_plugin",
        "message_type": "user_text_input",
        "payload": {"text": "hello"},
        "routing": make_routing(),
        "metadata": make_metadata(),
    }
    defaults.update(overrides)
    return Envelope(**defaults)


class TestRoutingInfo:
    def test_defaults(self):
        r = RoutingInfo(pipeline="p")
        assert r.pipeline == "p"
        assert r.completed_stages == []
        assert r.current_stage is None
        assert r.failed_stages == []
        assert r.hop_count == 0
        assert r.entry_slot == ""

    def test_to_dict(self):
        r = RoutingInfo(pipeline="p", hop_count=5, current_stage="mid")
        d = r.to_dict()
        assert d["pipeline"] == "p"
        assert d["hop_count"] == 5
        assert d["current_stage"] == "mid"

    def test_from_dict(self):
        d = {"pipeline": "p", "hop_count": 3, "current_stage": "s"}
        r = RoutingInfo.from_dict(d)
        assert r.hop_count == 3
        assert r.current_stage == "s"

    def test_from_dict_defaults(self):
        r = RoutingInfo.from_dict({"pipeline": "p"})
        assert r.completed_stages == []
        assert r.hop_count == 0

    def test_roundtrip(self):
        r = RoutingInfo(pipeline="p", completed_stages=["a", "b"], hop_count=7)
        d = r.to_dict()
        r2 = RoutingInfo.from_dict(d)
        assert r2.pipeline == r.pipeline
        assert r2.completed_stages == r.completed_stages
        assert r2.hop_count == r.hop_count


class TestEnvelopeMetadata:
    def test_defaults(self):
        m = EnvelopeMetadata()
        assert m.priority == "tier_3_normal"
        assert m.trust_level == "internal"
        assert m.trace_id == ""
        assert m.revision_count == 0
        assert m.parent_envelope_id is None
        assert m.correlation_id is None

    def test_to_dict(self):
        m = EnvelopeMetadata(priority="tier_1_immediate", revision_count=2)
        d = m.to_dict()
        assert d["priority"] == "tier_1_immediate"
        assert d["revision_count"] == 2

    def test_from_dict(self):
        d = {"priority": "tier_2_elevated", "trust_level": "tier_1_creator", "trace_id": "abc"}
        m = EnvelopeMetadata.from_dict(d)
        assert m.priority == "tier_2_elevated"
        assert m.trust_level == "tier_1_creator"

    def test_from_dict_defaults(self):
        m = EnvelopeMetadata.from_dict({})
        assert m.priority == "tier_3_normal"
        assert m.revision_count == 0

    def test_roundtrip(self):
        m = EnvelopeMetadata(
            priority="tier_1_immediate", trust_level="tier_2_trusted",
            trace_id="t", revision_count=1, parent_envelope_id="p", correlation_id="c",
        )
        d = m.to_dict()
        m2 = EnvelopeMetadata.from_dict(d)
        assert m2.priority == m.priority
        assert m2.parent_envelope_id == "p"
        assert m2.correlation_id == "c"


class TestEnvelope:
    def test_basic_creation(self):
        e = make_envelope()
        assert e.envelope_version == 1
        assert e.enrichments == {}
        assert e.payload == {"text": "hello"}

    def test_to_dict(self):
        e = make_envelope()
        d = e.to_dict()
        assert d["envelope_version"] == 1
        assert d["id"] == e.id
        assert d["origin_plugin"] == "test_plugin"
        assert "pipeline" in d["routing"]
        assert "priority" in d["metadata"]

    def test_from_dict(self):
        e = make_envelope()
        d = e.to_dict()
        e2 = Envelope.from_dict(d)
        assert e2.id == e.id
        assert e2.origin_plugin == e.origin_plugin
        assert e2.routing.pipeline == e.routing.pipeline

    def test_roundtrip(self):
        e = make_envelope(enrichments={"ns": {"key": "val"}})
        d = e.to_dict()
        e2 = Envelope.from_dict(d)
        assert e2.enrichments == {"ns": {"key": "val"}}
        assert e2.envelope_version == e.envelope_version


class TestEnvelopeWithEnrichment:
    def test_adds_namespace(self):
        e = make_envelope()
        e2 = e.with_enrichment("sentiment", {"score": 0.8})
        assert e2.enrichments == {"sentiment": {"score": 0.8}}

    def test_preserves_existing(self):
        e = make_envelope(enrichments={"ner": {"entities": ["X"]}})
        e2 = e.with_enrichment("sentiment", {"score": 0.5})
        assert e2.enrichments == {"ner": {"entities": ["X"]}, "sentiment": {"score": 0.5}}

    def test_immutable_original(self):
        e = make_envelope()
        e2 = e.with_enrichment("ns", {"k": "v"})
        assert e.enrichments == {}
        assert e2.enrichments == {"ns": {"k": "v"}}

    def test_overwrites_same_namespace(self):
        e = make_envelope(enrichments={"ns": {"old": True}})
        e2 = e.with_enrichment("ns", {"new": True})
        assert e2.enrichments == {"ns": {"new": True}}


class TestEnvelopeWithHopIncrement:
    def test_increments_hop_count(self):
        e = make_envelope(routing=make_routing(hop_count=5))
        e2 = e.with_hop_increment()
        assert e2.routing.hop_count == 6

    def test_immutable_original(self):
        e = make_envelope(routing=make_routing(hop_count=5))
        e.with_hop_increment()  # verify immutability
        assert e.routing.hop_count == 5

    def test_raises_at_max(self):
        e = make_envelope(routing=make_routing(hop_count=MAX_HOP_COUNT))
        with pytest.raises(EnvelopeError) as exc_info:
            e.with_hop_increment()
        assert exc_info.value.code == "loop_detected"

    def test_preserves_other_fields(self):
        e = make_envelope(routing=make_routing(hop_count=3, pipeline="p"))
        e2 = e.with_hop_increment()
        assert e2.routing.pipeline == "p"
        assert e2.id == e.id
        assert e2.metadata == e.metadata


class TestEnvelopeWithCompletedStage:
    def test_appends_stage(self):
        e = make_envelope(routing=make_routing(completed_stages=["stage_a"]))
        e2 = e.with_completed_stage("stage_b")
        assert e2.routing.completed_stages == ["stage_a", "stage_b"]

    def test_clears_current_stage(self):
        e = make_envelope(routing=make_routing(current_stage="stage_b"))
        e2 = e.with_completed_stage("stage_b")
        assert e2.routing.current_stage is None

    def test_immutable_original(self):
        e = make_envelope(routing=make_routing(completed_stages=["a"]))
        e.with_completed_stage("b")  # verify immutability
        assert e.routing.completed_stages == ["a"]


class TestEnvelopeWithFailedStage:
    def test_records_failure(self):
        e = make_envelope()
        e2 = e.with_failed_stage("stage_x", "timeout")
        assert e2.routing.failed_stages == [{"stage": "stage_x", "reason": "timeout"}]

    def test_appends_to_existing(self):
        e = make_envelope(routing=make_routing(failed_stages=[{"stage": "a", "reason": "err"}]))
        e2 = e.with_failed_stage("b", "crash")
        assert len(e2.routing.failed_stages) == 2
        assert e2.routing.failed_stages[1] == {"stage": "b", "reason": "crash"}

    def test_clears_current_stage(self):
        e = make_envelope(routing=make_routing(current_stage="failing"))
        e2 = e.with_failed_stage("failing", "error")
        assert e2.routing.current_stage is None


class TestEnvelopeWithRevision:
    def test_increments_revision_count(self):
        e = make_envelope(metadata=make_metadata(revision_count=0))
        e2 = e.with_revision()
        assert e2.metadata.revision_count == 1

    def test_double_revision(self):
        e = make_envelope(metadata=make_metadata(revision_count=1))
        e2 = e.with_revision()
        assert e2.metadata.revision_count == 2

    def test_raises_at_max(self):
        e = make_envelope(metadata=make_metadata(revision_count=MAX_REVISION_COUNT))
        with pytest.raises(EnvelopeError) as exc_info:
            e.with_revision()
        assert exc_info.value.code == "max_revisions_exceeded"

    def test_immutable_original(self):
        e = make_envelope(metadata=make_metadata(revision_count=0))
        e.with_revision()  # verify immutability
        assert e.metadata.revision_count == 0

    def test_preserves_other_metadata(self):
        e = make_envelope(metadata=make_metadata(trace_id="trace-123", priority="tier_2_elevated"))
        e2 = e.with_revision()
        assert e2.metadata.trace_id == "trace-123"
        assert e2.metadata.priority == "tier_2_elevated"


class TestEnvelopeDerive:
    def test_creates_new_id(self):
        e = make_envelope()
        e2 = e.derive("derived_type", {"data": "new"})
        assert e2.id != e.id

    def test_sets_parent_envelope_id(self):
        e = make_envelope()
        e2 = e.derive("derived_type", {"data": "new"})
        assert e2.metadata.parent_envelope_id == e.id

    def test_resets_enrichments(self):
        e = make_envelope(enrichments={"ns": {"old": True}})
        e2 = e.derive("derived_type", {})
        assert e2.enrichments == {}

    def test_resets_revision_count(self):
        e = make_envelope(metadata=make_metadata(revision_count=2))
        e2 = e.derive("derived_type", {})
        assert e2.metadata.revision_count == 0

    def test_inherits_priority_and_trace(self):
        e = make_envelope(metadata=make_metadata(
            priority="tier_1_immediate", trace_id="trace-abc", correlation_id="corr-1",
        ))
        e2 = e.derive("derived_type", {})
        assert e2.metadata.priority == "tier_1_immediate"
        assert e2.metadata.trace_id == "trace-abc"
        assert e2.metadata.correlation_id == "corr-1"

    def test_new_message_type_and_payload(self):
        e = make_envelope(message_type="original", payload={"old": True})
        e2 = e.derive("derived_type", {"new": True})
        assert e2.message_type == "derived_type"
        assert e2.payload == {"new": True}

    def test_new_routing_pipeline(self):
        e = make_envelope(routing=make_routing(pipeline="user_text_interaction"))
        e2 = e.derive("derived_type", {})
        assert e2.routing.pipeline == "user_text_interaction"
        assert e2.routing.completed_stages == []


class TestCreateEnvelope:
    def test_basic_creation(self):
        e = create_envelope(
            origin_plugin="my_plugin",
            message_type="user_text_input",
            payload={"text": "hi"},
            pipeline="user_text_interaction",
        )
        assert e.origin_plugin == "my_plugin"
        assert e.message_type == "user_text_input"
        assert e.payload == {"text": "hi"}
        assert e.routing.pipeline == "user_text_interaction"
        assert e.routing.hop_count == 0
        assert e.envelope_version == 1
        assert e.enrichments == {}

    def test_generates_uuid_id(self):
        e = create_envelope(
            origin_plugin="p", message_type="t", payload={}, pipeline="p",
        )
        uuid.UUID(e.id)  # should not raise

    def test_generates_trace_id(self):
        e = create_envelope(
            origin_plugin="p", message_type="t", payload={}, pipeline="p",
        )
        uuid.UUID(e.metadata.trace_id)  # should not raise

    def test_custom_priority(self):
        e = create_envelope(
            origin_plugin="p", message_type="t", payload={}, pipeline="p",
            priority="tier_1_immediate",
        )
        assert e.metadata.priority == "tier_1_immediate"

    def test_entry_slot(self):
        e = create_envelope(
            origin_plugin="p", message_type="t", payload={}, pipeline="p",
            entry_slot="input_reception",
        )
        assert e.routing.entry_slot == "input_reception"

    def test_custom_trust_level(self):
        e = create_envelope(
            origin_plugin="p", message_type="t", payload={}, pipeline="p",
            trust_level="tier_1_creator",
        )
        assert e.metadata.trust_level == "tier_1_creator"


class TestValidateEnvelope:
    def test_valid_envelope(self):
        e = create_envelope(
            origin_plugin="p", message_type="t", payload={}, pipeline="p",
        )
        errors = validate_envelope(e)
        assert errors == []

    def test_missing_id(self):
        e = make_envelope(id="")
        errors = validate_envelope(e)
        assert any("id" in err for err in errors)

    def test_missing_created_at(self):
        e = make_envelope(created_at="")
        errors = validate_envelope(e)
        assert any("created_at" in err for err in errors)

    def test_missing_origin_plugin(self):
        e = make_envelope(origin_plugin="")
        errors = validate_envelope(e)
        assert any("origin_plugin" in err for err in errors)

    def test_missing_message_type(self):
        e = make_envelope(message_type="")
        errors = validate_envelope(e)
        assert any("message_type" in err for err in errors)

    def test_missing_pipeline(self):
        e = make_envelope(routing=make_routing(pipeline=""))
        errors = validate_envelope(e)
        assert any("pipeline" in err for err in errors)

    def test_negative_hop_count(self):
        e = make_envelope(routing=make_routing(hop_count=-1))
        errors = validate_envelope(e)
        assert any("hop_count" in err for err in errors)

    def test_invalid_priority(self):
        e = make_envelope(metadata=make_metadata(priority="tier_99_bogus"))
        errors = validate_envelope(e)
        assert any("priority" in err for err in errors)

    def test_invalid_trust_level(self):
        e = make_envelope(metadata=make_metadata(trust_level="tier_99_bogus"))
        errors = validate_envelope(e)
        assert any("trust_level" in err for err in errors)

    def test_negative_revision_count(self):
        e = make_envelope(metadata=make_metadata(revision_count=-1))
        errors = validate_envelope(e)
        assert any("revision_count" in err for err in errors)

    def test_revision_count_exceeds_max(self):
        e = make_envelope(metadata=make_metadata(revision_count=MAX_REVISION_COUNT + 1))
        errors = validate_envelope(e)
        assert any("revision_count" in err for err in errors)

    def test_hop_count_exceeds_max(self):
        e = make_envelope(routing=make_routing(hop_count=MAX_HOP_COUNT + 1))
        errors = validate_envelope(e)
        assert any("hop_count" in err for err in errors)

    def test_valid_priorities(self):
        for p in ("tier_1_immediate", "tier_2_elevated", "tier_3_normal"):
            e = make_envelope(metadata=make_metadata(priority=p))
            errors = validate_envelope(e)
            priority_errors = [err for err in errors if "priority" in err]
            assert priority_errors == [], f"Unexpected error for priority {p}"

    def test_valid_trust_levels(self):
        for tl in ("tier_1_creator", "tier_2_trusted", "tier_3_external", "internal"):
            e = make_envelope(metadata=make_metadata(trust_level=tl))
            errors = validate_envelope(e)
            trust_errors = [err for err in errors if "trust_level" in err]
            assert trust_errors == [], f"Unexpected error for trust_level {tl}"

    def test_multiple_errors(self):
        e = make_envelope(id="", message_type="", origin_plugin="")
        errors = validate_envelope(e)
        assert len(errors) >= 3


class TestEnvelopeError:
    def test_error_format(self):
        err = EnvelopeError("test_code", "test message")
        assert "[test_code] test message" in str(err)
        assert err.code == "test_code"
        assert err.message == "test message"
