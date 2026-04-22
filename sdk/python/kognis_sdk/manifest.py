"""Plugin manifest parsing and validation.

Implements SPEC 02: Plugin Manifest. Parses plugin.yaml files into
typed Manifest objects with full validation.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import yaml


@dataclass
class SlotRegistration:
    """A plugin's registration for a pipeline slot."""

    pipeline: str
    slot: str
    priority: int
    message_types_handled: list[str] = field(default_factory=list)
    message_types_produced: list[str] = field(default_factory=list)
    timeout_seconds: int = 30
    retry_attempts: int = 0
    optional: bool = False
    max_concurrent: int = 1

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> SlotRegistration:
        return cls(
            pipeline=data.get("pipeline", ""),
            slot=data.get("slot", ""),
            priority=data.get("priority", 0),
            message_types_handled=data.get("message_types_handled", []),
            message_types_produced=data.get("message_types_produced", []),
            timeout_seconds=data.get("timeout_seconds", 30),
            retry_attempts=data.get("retry_attempts", 0),
            optional=data.get("optional", False),
            max_concurrent=data.get("max_concurrent", 1),
        )


@dataclass
class CapabilitySpec:
    """A capability provided by a plugin."""

    capability_id: str
    description: str
    latency_class: str = "medium"
    llm_tool_description: str | None = None
    llm_tool_expose_to: list[str] = field(default_factory=list)
    authentication_required: bool = False
    params_schema: dict[str, Any] | None = None
    response_schema: dict[str, Any] | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> CapabilitySpec:
        return cls(
            capability_id=data.get("capability_id", ""),
            description=data.get("description", ""),
            latency_class=data.get("latency_class", "medium"),
            llm_tool_description=data.get("llm_tool_description"),
            llm_tool_expose_to=data.get("llm_tool_expose_to", []),
            authentication_required=data.get("authentication_required", False),
            params_schema=data.get("params_schema"),
            response_schema=data.get("response_schema"),
        )


@dataclass
class RequiredCapability:
    """A capability required from another plugin."""

    capability_id: str
    optional: bool = False

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> RequiredCapability:
        if isinstance(data, str):
            return cls(capability_id=data, optional=False)
        return cls(
            capability_id=data["capability_id"],
            optional=data.get("optional", False),
        )


@dataclass
class EventSubscription:
    """An event bus subscription."""

    topic: str
    handler: str

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> EventSubscription:
        return cls(topic=data["topic"], handler=data["handler"])


@dataclass
class EventPublication:
    """An event bus publication."""

    topic: str
    schema_ref: str | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> EventPublication:
        if isinstance(data, str):
            return cls(topic=data, schema_ref=None)
        return cls(topic=data["topic"], schema_ref=data.get("schema_ref"))


@dataclass
class StateBroadcast:
    """A state broadcast declaration."""

    state_name: str
    values: list[str]
    change_topic: str
    description: str | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> StateBroadcast:
        return cls(
            state_name=data["state_name"],
            values=data["values"],
            change_topic=data["change_topic"],
            description=data.get("description"),
        )


@dataclass
class RuntimeSpec:
    """Plugin runtime configuration."""

    entrypoint: str
    working_directory: str | None = None
    python_version: str | None = None
    system_packages: list[str] = field(default_factory=list)
    external_commands: list[str] = field(default_factory=list)

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> RuntimeSpec:
        env_req = data.get("environment_requirements", {})
        return cls(
            entrypoint=data["entrypoint"],
            working_directory=data.get("working_directory"),
            python_version=env_req.get("python_version") if env_req else None,
            system_packages=env_req.get("system_packages", []) if env_req else [],
            external_commands=env_req.get("external_commands", []) if env_req else [],
        )


@dataclass
class LifecycleSpec:
    """Plugin lifecycle configuration."""

    startup_order: int = 50
    health_pulse_interval: int = 30
    state_broadcast: bool = False
    sleep_behavior: str = "suspend"

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> LifecycleSpec:
        return cls(
            startup_order=data.get("startup_order", 50),
            health_pulse_interval=data.get("health_pulse_interval", 30),
            state_broadcast=data.get("state_broadcast", False),
            sleep_behavior=data.get("sleep_behavior", "suspend"),
        )


@dataclass
class Manifest:
    """Parsed and validated plugin manifest.

    Spec reference: docs/spec/02-plugin-manifest.md
    """

    manifest_version: int
    plugin_id: str
    plugin_name: str
    version: str
    author: str
    license: str
    description: str
    language: str
    runtime: RuntimeSpec
    handler_mode: str
    slot_registrations: list[SlotRegistration]
    provides_capabilities: list[CapabilitySpec] = field(default_factory=list)
    requires_capabilities: list[RequiredCapability] = field(default_factory=list)
    event_subscriptions: list[EventSubscription] = field(default_factory=list)
    event_publications: list[EventPublication] = field(default_factory=list)
    state_broadcasts: list[StateBroadcast] = field(default_factory=list)
    permissions: list[str] = field(default_factory=list)
    lifecycle: LifecycleSpec | None = None
    sdk_required_version: str | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> Manifest:
        """Parse a manifest from a dict (e.g., loaded from YAML)."""
        runtime_data = data.get("runtime", {})
        runtime = RuntimeSpec.from_dict(runtime_data) if runtime_data else RuntimeSpec(entrypoint="")

        slot_regs = [SlotRegistration.from_dict(s) for s in data.get("slot_registrations", [])]
        provides_caps = [CapabilitySpec.from_dict(c) for c in data.get("provides_capabilities", [])]
        requires_caps = [RequiredCapability.from_dict(c) for c in data.get("requires_capabilities", [])]
        event_subs = [EventSubscription.from_dict(e) for e in data.get("event_subscriptions", [])]
        event_pubs = [EventPublication.from_dict(e) for e in data.get("event_publications", [])]
        state_bcasts = [StateBroadcast.from_dict(s) for s in data.get("state_broadcasts", [])]

        lifecycle_data = data.get("lifecycle")
        lifecycle = LifecycleSpec.from_dict(lifecycle_data) if lifecycle_data else None

        sdk_data = data.get("sdk", {})

        return cls(
            manifest_version=data["manifest_version"],
            plugin_id=data["plugin_id"],
            plugin_name=data["plugin_name"],
            version=data["version"],
            author=data["author"],
            license=data["license"],
            description=data["description"],
            language=data["language"],
            runtime=runtime,
            handler_mode=data["handler_mode"],
            slot_registrations=slot_regs,
            provides_capabilities=provides_caps,
            requires_capabilities=requires_caps,
            event_subscriptions=event_subs,
            event_publications=event_pubs,
            state_broadcasts=state_bcasts,
            permissions=data.get("permissions", []),
            lifecycle=lifecycle,
            sdk_required_version=sdk_data.get("required_version") if sdk_data else None,
        )

    @classmethod
    def from_yaml(cls, path: str | Path) -> Manifest:
        """Load and parse a manifest from a YAML file."""
        path = Path(path)
        if not path.exists():
            raise FileNotFoundError(f"Manifest file not found: {path}")
        with open(path) as f:
            data = yaml.safe_load(f)
        if data is None:
            raise ValueError(f"Empty manifest file: {path}")
        return cls.from_dict(data)


class ManifestValidationError(Exception):
    """Raised when manifest validation fails."""

    def __init__(self, errors: list[str]) -> None:
        self.errors = errors
        super().__init__(f"Manifest validation failed: {'; '.join(errors)}")


def validate_manifest(manifest: Manifest) -> list[str]:
    """Validate a manifest and return list of error strings.

    Implements SPEC 02 Section 2.4 validation rules.
    """
    errors: list[str] = []

    # 1. Schema validation
    if manifest.manifest_version != 1:
        errors.append(f"Unsupported manifest_version: {manifest.manifest_version}")

    if manifest.handler_mode not in ("stateless", "stateful_agent"):
        errors.append(f"Invalid handler_mode: {manifest.handler_mode}")

    if manifest.language not in ("python", "go", "node", "other"):
        errors.append(f"Invalid language: {manifest.language}")

    if not manifest.version:
        errors.append("version is required")

    if not manifest.plugin_id:
        errors.append("plugin_id is required")

    # 2. Slot registration validation
    for reg in manifest.slot_registrations:
        if not reg.pipeline:
            errors.append("slot_registration missing pipeline")
        if not reg.slot:
            errors.append("slot_registration missing slot")
        if reg.priority < 0 or reg.priority > 100:
            errors.append(f"priority {reg.priority} out of range 0-100 for {reg.pipeline}/{reg.slot}")

    # 3. Capability validation
    for cap in manifest.provides_capabilities:
        if not cap.capability_id:
            errors.append("capability missing capability_id")
        if cap.latency_class not in ("fast", "medium", "slow"):
            errors.append(f"Invalid latency_class: {cap.latency_class}")

    # 4. State broadcast validation
    for sb in manifest.state_broadcasts:
        if not sb.values:
            errors.append(f"state_broadcast '{sb.state_name}' has no values")

    return errors


def validate_manifest_strict(manifest: Manifest) -> None:
    """Validate manifest and raise if errors found."""
    errors = validate_manifest(manifest)
    if errors:
        raise ManifestValidationError(errors)