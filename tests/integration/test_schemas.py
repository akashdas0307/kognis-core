"""Schema validation tests for Kognis shared schemas.

Validates that all schema files are valid YAML, contain required fields,
and that example data conforms to the schema structure.
"""

import os
from pathlib import Path

import pytest
import yaml

SCHEMAS_DIR = Path(__file__).parent.parent.parent / "schemas"


def _load_schema(name: str) -> dict:
    """Load a schema file by name."""
    path = SCHEMAS_DIR / name
    if not path.exists():
        pytest.skip(f"Schema file {name} not found")
    with open(path) as f:
        return yaml.safe_load(f)


class TestSchemaFiles:
    """Verify all schema files exist and are valid YAML."""

    EXPECTED_SCHEMAS = [
        "manifest-v1.yaml",
        "envelope-v1.yaml",
        "pipeline-template-v1.yaml",
        "health-pulse-v1.yaml",
        "state-broadcast-v1.yaml",
        "registry-entry-v1.yaml",
    ]

    @pytest.mark.parametrize("filename", EXPECTED_SCHEMAS)
    def test_schema_file_exists(self, filename: str):
        path = SCHEMAS_DIR / filename
        assert path.exists(), f"Schema file {filename} does not exist"

    @pytest.mark.parametrize("filename", EXPECTED_SCHEMAS)
    def test_schema_is_valid_yaml(self, filename: str):
        data = _load_schema(filename)
        assert isinstance(data, dict), f"Schema {filename} is not a dict"

    @pytest.mark.parametrize("filename", EXPECTED_SCHEMAS)
    def test_schema_has_api_version(self, filename: str):
        data = _load_schema(filename)
        assert "api_version" in data, f"Schema {filename} missing api_version"
        assert data["api_version"] == 1

    @pytest.mark.parametrize("filename", EXPECTED_SCHEMAS)
    def test_schema_has_name(self, filename: str):
        data = _load_schema(filename)
        assert "schema_name" in data, f"Schema {filename} missing schema_name"

    @pytest.mark.parametrize("filename", EXPECTED_SCHEMAS)
    def test_schema_has_version(self, filename: str):
        data = _load_schema(filename)
        assert "schema_version" in data, f"Schema {filename} missing schema_version"

    @pytest.mark.parametrize("filename", EXPECTED_SCHEMAS)
    def test_schema_has_description(self, filename: str):
        data = _load_schema(filename)
        assert "description" in data, f"Schema {filename} missing description"

    @pytest.mark.parametrize("filename", EXPECTED_SCHEMAS)
    def test_schema_has_fields(self, filename: str):
        data = _load_schema(filename)
        assert "fields" in data, f"Schema {filename} missing fields section"


class TestManifestSchema:
    """Validate manifest schema structure."""

    def test_manifest_has_identity_fields(self):
        data = _load_schema("manifest-v1.yaml")
        fields = data["fields"]
        required_identity = ["manifest_version", "plugin_id", "plugin_name", "version", "author"]
        for field_name in required_identity:
            assert field_name in fields, f"Manifest schema missing field: {field_name}"

    def test_manifest_has_handler_mode(self):
        data = _load_schema("manifest-v1.yaml")
        fields = data["fields"]
        assert "handler_mode" in fields
        assert "enum" in fields["handler_mode"]
        assert "stateless" in fields["handler_mode"]["enum"]
        assert "stateful_agent" in fields["handler_mode"]["enum"]

    def test_manifest_has_slot_registrations(self):
        data = _load_schema("manifest-v1.yaml")
        fields = data["fields"]
        assert "slot_registrations" in fields

    def test_manifest_has_capabilities(self):
        data = _load_schema("manifest-v1.yaml")
        fields = data["fields"]
        assert "provides_capabilities" in fields
        assert "requires_capabilities" in fields


class TestEnvelopeSchema:
    """Validate envelope schema structure."""

    def test_envelope_has_routing(self):
        data = _load_schema("envelope-v1.yaml")
        fields = data["fields"]
        assert "routing" in fields
        routing_fields = fields["routing"]["fields"]
        assert "pipeline" in routing_fields
        assert "hop_count" in routing_fields
        assert "completed_stages" in routing_fields

    def test_envelope_has_metadata(self):
        data = _load_schema("envelope-v1.yaml")
        fields = data["fields"]
        assert "metadata" in fields
        meta_fields = fields["metadata"]["fields"]
        assert "priority" in meta_fields
        assert "trust_level" in meta_fields

    def test_envelope_has_constraints(self):
        data = _load_schema("envelope-v1.yaml")
        assert "constraints" in data
        assert len(data["constraints"]) > 0


class TestPipelineTemplateSchema:
    """Validate pipeline template schema structure."""

    def test_pipeline_has_slots(self):
        data = _load_schema("pipeline-template-v1.yaml")
        fields = data["fields"]
        assert "slots" in fields
        assert "pipeline_id" in fields
        assert "accepted_message_types" in fields

    def test_slot_items_have_required_fields(self):
        data = _load_schema("pipeline-template-v1.yaml")
        slot_items = data["fields"]["slots"]["items"]
        assert "slot_id" in slot_items
        assert "required" in slot_items


class TestHealthPulseSchema:
    """Validate health pulse schema structure."""

    def test_pulse_has_status_enum(self):
        data = _load_schema("health-pulse-v1.yaml")
        pulse_fields = data["fields"]["health_pulse"]["fields"]
        status = pulse_fields["status"]
        assert "enum" in status
        expected = ["HEALTHY", "DEGRADED", "ERROR", "CRITICAL", "UNRESPONSIVE"]
        assert set(status["enum"]) == set(expected)

    def test_pulse_has_metrics(self):
        data = _load_schema("health-pulse-v1.yaml")
        pulse_fields = data["fields"]["health_pulse"]["fields"]
        assert "metrics" in pulse_fields

    def test_pulse_has_alerts(self):
        data = _load_schema("health-pulse-v1.yaml")
        pulse_fields = data["fields"]["health_pulse"]["fields"]
        assert "alerts" in pulse_fields


class TestStateBroadcastSchema:
    """Validate state broadcast schema structure."""

    def test_broadcast_has_transport(self):
        data = _load_schema("state-broadcast-v1.yaml")
        assert "transport" in data
        assert data["transport"]["protocol"] == "NATS pub/sub"

    def test_broadcast_has_topic_pattern(self):
        data = _load_schema("state-broadcast-v1.yaml")
        assert "topic_pattern" in data["transport"]

    def test_broadcast_has_manifest_declaration(self):
        data = _load_schema("state-broadcast-v1.yaml")
        assert "manifest_declaration" in data


class TestRegistryEntrySchema:
    """Validate registry entry schema structure."""

    def test_registry_has_verification(self):
        data = _load_schema("registry-entry-v1.yaml")
        fields = data["fields"]
        assert "verified" in fields

    def test_registry_has_compatibility(self):
        data = _load_schema("registry-entry-v1.yaml")
        fields = data["fields"]
        assert "min_framework_version" in fields