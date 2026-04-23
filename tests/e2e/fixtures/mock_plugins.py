import pytest
import asyncio
from kognis_sdk.plugin import Plugin, PluginConfig
from kognis_sdk.manifest import Manifest, RuntimeSpec, LifecycleSpec, SlotRegistration
from kognis_sdk.envelope import Envelope

@pytest.fixture
def mock_manifest_factory():
    """Factory fixture to generate valid mock plugin manifests."""
    def _create_manifest(
        plugin_id: str,
        plugin_name: str = "Mock Plugin",
        slots: list[SlotRegistration] = None
    ) -> Manifest:
        return Manifest(
            manifest_version=1,
            plugin_id=plugin_id,
            plugin_name=plugin_name,
            version="1.0.0",
            author="Test",
            license="MIT",
            description="A mock plugin for E2E testing",
            language="python",
            runtime=RuntimeSpec(entrypoint="mock"),
            handler_mode="stateless",
            slot_registrations=slots or [],
            lifecycle=LifecycleSpec(health_pulse_interval=1)
        )
    return _create_manifest


@pytest.fixture
def mock_plugin_factory(mock_manifest_factory):
    """
    Factory fixture to create and yield a connected Mock Plugin.
    Provides async cleanup.
    """
    plugins = []

    async def _create_plugin(plugin_id: str, slots: list[SlotRegistration] = None):
        manifest = mock_manifest_factory(plugin_id=plugin_id, slots=slots)
        config = PluginConfig(socket_path="/tmp/kognis.sock")
        plugin = Plugin(manifest, config)
        
        # Start in background if we don't want to block
        # We need to manually do the start() sequence to await it in tests
        await plugin.start()
        plugins.append(plugin)
        return plugin

    yield _create_plugin

    # Teardown
    for p in plugins:
        if p.is_running:
            try:
                asyncio.run(p.stop())
            except Exception:
                pass


@pytest.fixture
def dummy_handler():
    """A dummy slot handler that just adds an enrichment."""
    async def handler(envelope: Envelope) -> Envelope:
        return envelope.with_enrichment("mock_processed", {"status": "ok"})
    return handler
