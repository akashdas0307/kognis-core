"""Tests for manifest parser — M-001.

Validates SPEC 02: Plugin Manifest parsing and validation.
"""

import pytest
import yaml

from kognis_sdk.manifest import (
    CapabilitySpec,
    EventPublication,
    EventSubscription,
    LifecycleSpec,
    Manifest,
    ManifestValidationError,
    RequiredCapability,
    RuntimeSpec,
    SlotRegistration,
    StateBroadcast,
    validate_manifest,
    validate_manifest_strict,
)

MINIMAL_MANIFEST_DICT = {
    "manifest_version": 1,
    "plugin_id": "com.example.test_plugin",
    "plugin_name": "Test Plugin",
    "version": "1.0.0",
    "author": "Test Author",
    "license": "MIT",
    "description": "A test plugin",
    "language": "python",
    "runtime": {"entrypoint": "main.py"},
    "handler_mode": "stateless",
    "slot_registrations": [
        {
            "pipeline": "user_text_interaction",
            "slot": "input_reception",
            "priority": 50,
        }
    ],
}

FULL_MANIFEST_DICT = {
    "manifest_version": 1,
    "plugin_id": "com.example.full_plugin",
    "plugin_name": "Full Plugin",
    "version": "2.1.0",
    "author": "Full Author",
    "license": "Apache-2.0",
    "description": "A fully-featured plugin",
    "language": "python",
    "runtime": {
        "entrypoint": "plugin.py",
        "working_directory": "/app",
        "environment_requirements": {
            "python_version": "3.11",
            "system_packages": ["ffmpeg"],
            "external_commands": ["ffmpeg"],
        },
    },
    "handler_mode": "stateful_agent",
    "slot_registrations": [
        {
            "pipeline": "user_text_interaction",
            "slot": "cognitive_processing",
            "priority": 30,
            "message_types_handled": ["user_text_input"],
            "message_types_produced": ["cognitive_response"],
            "timeout_seconds": 60,
            "retry_attempts": 2,
            "optional": False,
            "max_concurrent": 3,
        },
        {
            "pipeline": "background_monitoring",
            "slot": "ambient_assessment",
            "priority": 70,
        },
    ],
    "provides_capabilities": [
        {
            "capability_id": "com.example.summarize",
            "description": "Summarize text",
            "latency_class": "fast",
            "llm_tool_description": "Summarize the given text",
            "llm_tool_expose_to": ["cognitive_processing"],
            "authentication_required": True,
            "params_schema": {"type": "object"},
            "response_schema": {"type": "object"},
        }
    ],
    "requires_capabilities": [
        {"capability_id": "com.example.embed", "optional": False},
        {"capability_id": "com.example.translate", "optional": True},
    ],
    "event_subscriptions": [
        {"topic": "memory.updated", "handler": "on_memory_update"},
    ],
    "event_publications": [
        {"topic": "plugin.ready", "schema_ref": "schemas/ready-v1.yaml"},
        "idle_notification",
    ],
    "state_broadcasts": [
        {
            "state_name": "processing_mode",
            "values": ["idle", "busy", "sleeping"],
            "change_topic": "state.processing_mode",
            "description": "Current processing mode",
        }
    ],
    "permissions": ["network.outbound", "filesystem.read"],
    "lifecycle": {
        "startup_order": 30,
        "health_pulse_interval": 15,
        "state_broadcast": True,
        "sleep_behavior": "background",
    },
    "sdk": {"required_version": ">=0.1.0"},
}


class TestSlotRegistration:
    def test_from_dict_minimal(self):
        data = {"pipeline": "user_text_interaction", "slot": "input_reception", "priority": 50}
        reg = SlotRegistration.from_dict(data)
        assert reg.pipeline == "user_text_interaction"
        assert reg.slot == "input_reception"
        assert reg.priority == 50
        assert reg.message_types_handled == []
        assert reg.message_types_produced == []
        assert reg.timeout_seconds == 30
        assert reg.retry_attempts == 0
        assert reg.optional is False
        assert reg.max_concurrent == 1

    def test_from_dict_full(self):
        data = {
            "pipeline": "p",
            "slot": "s",
            "priority": 10,
            "message_types_handled": ["a", "b"],
            "message_types_produced": ["c"],
            "timeout_seconds": 120,
            "retry_attempts": 5,
            "optional": True,
            "max_concurrent": 4,
        }
        reg = SlotRegistration.from_dict(data)
        assert reg.message_types_handled == ["a", "b"]
        assert reg.timeout_seconds == 120
        assert reg.optional is True
        assert reg.max_concurrent == 4


class TestCapabilitySpec:
    def test_from_dict_minimal(self):
        data = {"capability_id": "cap1", "description": "Does things"}
        cap = CapabilitySpec.from_dict(data)
        assert cap.capability_id == "cap1"
        assert cap.latency_class == "medium"
        assert cap.llm_tool_description is None
        assert cap.authentication_required is False

    def test_from_dict_full(self):
        data = {
            "capability_id": "cap2",
            "description": "Does more",
            "latency_class": "slow",
            "llm_tool_description": "Tool desc",
            "llm_tool_expose_to": ["slot_a"],
            "authentication_required": True,
        }
        cap = CapabilitySpec.from_dict(data)
        assert cap.latency_class == "slow"
        assert cap.authentication_required is True


class TestRequiredCapability:
    def test_from_string(self):
        cap = RequiredCapability.from_dict("com.example.embed")
        assert cap.capability_id == "com.example.embed"
        assert cap.optional is False

    def test_from_dict(self):
        cap = RequiredCapability.from_dict({"capability_id": "cap1", "optional": True})
        assert cap.capability_id == "cap1"
        assert cap.optional is True


class TestEventSubscription:
    def test_from_dict(self):
        sub = EventSubscription.from_dict({"topic": "memory.updated", "handler": "on_mem"})
        assert sub.topic == "memory.updated"
        assert sub.handler == "on_mem"


class TestEventPublication:
    def test_from_string(self):
        pub = EventPublication.from_dict("idle_notification")
        assert pub.topic == "idle_notification"
        assert pub.schema_ref is None

    def test_from_dict_with_schema(self):
        pub = EventPublication.from_dict({"topic": "ready", "schema_ref": "schemas/v1.yaml"})
        assert pub.topic == "ready"
        assert pub.schema_ref == "schemas/v1.yaml"


class TestStateBroadcast:
    def test_from_dict(self):
        data = {
            "state_name": "mode",
            "values": ["idle", "busy"],
            "change_topic": "state.mode",
            "description": "Current mode",
        }
        sb = StateBroadcast.from_dict(data)
        assert sb.state_name == "mode"
        assert sb.values == ["idle", "busy"]
        assert sb.change_topic == "state.mode"

    def test_from_dict_no_description(self):
        data = {"state_name": "mode", "values": ["a"], "change_topic": "t"}
        sb = StateBroadcast.from_dict(data)
        assert sb.description is None


class TestRuntimeSpec:
    def test_from_dict_minimal(self):
        rt = RuntimeSpec.from_dict({"entrypoint": "main.py"})
        assert rt.entrypoint == "main.py"
        assert rt.working_directory is None
        assert rt.python_version is None
        assert rt.system_packages == []

    def test_from_dict_full(self):
        data = {
            "entrypoint": "app.py",
            "working_directory": "/opt",
            "environment_requirements": {
                "python_version": "3.12",
                "system_packages": ["libav"],
                "external_commands": ["ffmpeg"],
            },
        }
        rt = RuntimeSpec.from_dict(data)
        assert rt.working_directory == "/opt"
        assert rt.python_version == "3.12"
        assert rt.system_packages == ["libav"]
        assert rt.external_commands == ["ffmpeg"]


class TestLifecycleSpec:
    def test_from_dict_defaults(self):
        lc = LifecycleSpec.from_dict({})
        assert lc.startup_order == 50
        assert lc.health_pulse_interval == 30
        assert lc.state_broadcast is False
        assert lc.sleep_behavior == "suspend"

    def test_from_dict_custom(self):
        lc = LifecycleSpec.from_dict(
            {
                "startup_order": 10,
                "health_pulse_interval": 5,
                "state_broadcast": True,
                "sleep_behavior": "background",
            }
        )
        assert lc.startup_order == 10
        assert lc.state_broadcast is True
        assert lc.sleep_behavior == "background"


class TestManifest:
    def test_from_dict_minimal(self):
        m = Manifest.from_dict(MINIMAL_MANIFEST_DICT)
        assert m.manifest_version == 1
        assert m.plugin_id == "com.example.test_plugin"
        assert m.plugin_name == "Test Plugin"
        assert m.version == "1.0.0"
        assert m.language == "python"
        assert m.handler_mode == "stateless"
        assert len(m.slot_registrations) == 1
        assert m.provides_capabilities == []
        assert m.requires_capabilities == []
        assert m.lifecycle is None
        assert m.sdk_required_version is None

    def test_from_dict_full(self):
        m = Manifest.from_dict(FULL_MANIFEST_DICT)
        assert m.handler_mode == "stateful_agent"
        assert len(m.slot_registrations) == 2
        assert len(m.provides_capabilities) == 1
        assert len(m.requires_capabilities) == 2
        assert m.provides_capabilities[0].capability_id == "com.example.summarize"
        assert m.requires_capabilities[1].optional is True
        assert len(m.event_subscriptions) == 1
        assert len(m.event_publications) == 2
        assert m.event_publications[1].topic == "idle_notification"
        assert len(m.state_broadcasts) == 1
        assert m.permissions == ["network.outbound", "filesystem.read"]
        assert m.lifecycle is not None
        assert m.lifecycle.startup_order == 30
        assert m.sdk_required_version == ">=0.1.0"

    def test_from_dict_missing_runtime(self):
        data = {**MINIMAL_MANIFEST_DICT, "runtime": {}}
        m = Manifest.from_dict(data)
        assert m.runtime.entrypoint == ""

    def test_from_yaml(self, tmp_path):
        yaml_path = tmp_path / "plugin.yaml"
        yaml_path.write_text(yaml.dump(MINIMAL_MANIFEST_DICT, default_flow_style=False))
        m = Manifest.from_yaml(yaml_path)
        assert m.plugin_id == "com.example.test_plugin"

    def test_from_yaml_file_not_found(self):
        with pytest.raises(FileNotFoundError):
            Manifest.from_yaml("/nonexistent/plugin.yaml")

    def test_from_yaml_empty_file(self, tmp_path):
        yaml_path = tmp_path / "empty.yaml"
        yaml_path.write_text("")
        with pytest.raises(ValueError, match="Empty manifest"):
            Manifest.from_yaml(yaml_path)


class TestValidateManifest:
    def test_valid_minimal(self):
        m = Manifest.from_dict(MINIMAL_MANIFEST_DICT)
        errors = validate_manifest(m)
        assert errors == []

    def test_valid_full(self):
        m = Manifest.from_dict(FULL_MANIFEST_DICT)
        errors = validate_manifest(m)
        assert errors == []

    def test_invalid_manifest_version(self):
        data = {**MINIMAL_MANIFEST_DICT, "manifest_version": 2}
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("manifest_version" in e for e in errors)

    def test_invalid_handler_mode(self):
        data = {**MINIMAL_MANIFEST_DICT, "handler_mode": "invalid"}
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("handler_mode" in e for e in errors)

    def test_invalid_language(self):
        data = {**MINIMAL_MANIFEST_DICT, "language": "rust"}
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("language" in e for e in errors)

    def test_empty_version(self):
        data = {**MINIMAL_MANIFEST_DICT, "version": ""}
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("version" in e for e in errors)

    def test_empty_plugin_id(self):
        data = {**MINIMAL_MANIFEST_DICT, "plugin_id": ""}
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("plugin_id" in e for e in errors)

    def test_slot_missing_pipeline(self):
        data = {
            **MINIMAL_MANIFEST_DICT,
            "slot_registrations": [{"slot": "input_reception", "priority": 50}],
        }
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("pipeline" in e for e in errors)

    def test_slot_missing_slot(self):
        data = {
            **MINIMAL_MANIFEST_DICT,
            "slot_registrations": [{"pipeline": "p", "priority": 50}],
        }
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("slot" in e for e in errors)

    def test_slot_priority_out_of_range(self):
        data = {
            **MINIMAL_MANIFEST_DICT,
            "slot_registrations": [
                {"pipeline": "p", "slot": "s", "priority": 101},
            ],
        }
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("priority" in e for e in errors)

    def test_slot_priority_negative(self):
        data = {
            **MINIMAL_MANIFEST_DICT,
            "slot_registrations": [
                {"pipeline": "p", "slot": "s", "priority": -1},
            ],
        }
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("priority" in e for e in errors)

    def test_capability_missing_id(self):
        data = {
            **MINIMAL_MANIFEST_DICT,
            "provides_capabilities": [{"description": "no id", "latency_class": "fast"}],
        }
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("capability_id" in e for e in errors)

    def test_capability_invalid_latency(self):
        data = {
            **MINIMAL_MANIFEST_DICT,
            "provides_capabilities": [
                {"capability_id": "c", "description": "d", "latency_class": "instant"},
            ],
        }
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("latency_class" in e for e in errors)

    def test_state_broadcast_no_values(self):
        data = {
            **MINIMAL_MANIFEST_DICT,
            "state_broadcasts": [
                {"state_name": "x", "values": [], "change_topic": "t"},
            ],
        }
        m = Manifest.from_dict(data)
        errors = validate_manifest(m)
        assert any("state_broadcast" in e for e in errors)


class TestValidateManifestStrict:
    def test_valid_raises_nothing(self):
        m = Manifest.from_dict(MINIMAL_MANIFEST_DICT)
        validate_manifest_strict(m)  # should not raise

    def test_invalid_raises(self):
        data = {**MINIMAL_MANIFEST_DICT, "handler_mode": "bad"}
        m = Manifest.from_dict(data)
        with pytest.raises(ManifestValidationError) as exc_info:
            validate_manifest_strict(m)
        assert "handler_mode" in str(exc_info.value)
        assert len(exc_info.value.errors) > 0


class TestManifestValidationError:
    def test_error_format(self):
        err = ManifestValidationError(["err1", "err2"])
        assert "err1" in str(err)
        assert "err2" in str(err)
        assert err.errors == ["err1", "err2"]
