"""Kognis Plugin SDK — Python implementation.

Provides the base classes, protocol clients, and utilities for building
Kognis Framework plugins.
"""

__version__ = "0.1.0"

from kognis_sdk.envelope import (
    Envelope,
    EnvelopeError,
    EnvelopeMetadata,
    RoutingInfo,
    create_envelope,
    validate_envelope,
    MAX_HOP_COUNT,
    MAX_REVISION_COUNT,
)
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
from kognis_sdk.control_plane import (
    ControlPlaneClient,
    ControlPlaneError,
    PluginState,
    DispatchMessage,
    DispatchAck,
    DispatchComplete,
    DispatchFailed,
    CapabilityQuery,
    CapabilityResponse,
    Heartbeat,
    HeartbeatAck,
    RegisterRequest,
    RegisterAck,
    ShutdownRequest,
)
from kognis_sdk.eventbus import (
    EventBusClient,
    EventBusConfig,
    EventBusError,
    make_state_topic,
    make_event_topic,
    parse_topic,
)
from kognis_sdk.plugin import (
    Plugin,
    PluginConfig,
    PluginError,
)
from kognis_sdk.stateful_agent import StatefulAgent
from kognis_sdk.capability import (
    CapabilityRegistryClient,
    RegistryEntry,
)
from kognis_sdk.tool_bridge import (
    ToolBridge,
    ToolSchema,
    ToolUseBlock,
    ToolResult,
)
from kognis_sdk.context_budget import (
    ContextBudgetManager,
    ContextBlock,
    ContextBudgetError,
    BudgetConfig,
    PriorityTier,
)
from kognis_sdk.health import (
    HealthPulseEmitter,
    HealthPulse,
    StateBroadcaster,
    StateChange,
    HEALTHY,
    DEGRADED,
    ERROR,
    CRITICAL,
    UNRESPONSIVE,
)
from kognis_sdk.state_store import (
    StateStore,
    StateStoreError,
)

__all__ = [
    "Envelope", "EnvelopeError", "EnvelopeMetadata", "RoutingInfo",
    "create_envelope", "validate_envelope", "MAX_HOP_COUNT", "MAX_REVISION_COUNT",
    "Manifest", "ManifestValidationError", "SlotRegistration", "CapabilitySpec",
    "RequiredCapability", "EventSubscription", "EventPublication", "StateBroadcast",
    "RuntimeSpec", "LifecycleSpec", "validate_manifest", "validate_manifest_strict",
    "ControlPlaneClient", "ControlPlaneError", "PluginState", "DispatchMessage",
    "DispatchAck", "DispatchComplete", "DispatchFailed", "CapabilityQuery",
    "CapabilityResponse", "Heartbeat", "HeartbeatAck", "RegisterRequest",
    "RegisterAck", "ShutdownRequest",
    "EventBusClient", "EventBusConfig", "EventBusError",
    "make_state_topic", "make_event_topic", "parse_topic",
    "Plugin", "PluginConfig", "PluginError",
    "StatefulAgent",
    "CapabilityRegistryClient", "RegistryEntry",
    "ToolBridge", "ToolSchema", "ToolUseBlock", "ToolResult",
    "ContextBudgetManager", "ContextBlock", "ContextBudgetError", "BudgetConfig", "PriorityTier",
    "HealthPulseEmitter", "HealthPulse", "StateBroadcaster", "StateChange",
    "HEALTHY", "DEGRADED", "ERROR", "CRITICAL", "UNRESPONSIVE",
    "StateStore", "StateStoreError",
]