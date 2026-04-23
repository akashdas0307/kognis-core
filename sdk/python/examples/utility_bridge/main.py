import asyncio
import logging
import os
import sys

# Add the SDK directory to sys.path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "../..")))

from kognis_sdk.context_budget import ContextBudgetManager
from kognis_sdk.envelope import Envelope
from kognis_sdk.manifest import Manifest
from kognis_sdk.plugin import Plugin
from kognis_sdk.tool_bridge import ToolBridge

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
logger = logging.getLogger("utility_bridge")

class UtilityBridgePlugin(Plugin):
    """A stateless plugin demonstrating ToolBridge and ContextBudget usage."""

    async def on_startup(self) -> None:
        logger.info("Utility Bridge online.")
        # Initialize specialized SDK managers
        self.tools = ToolBridge(self.control_plane)
        self.budget = ContextBudgetManager()

        self.register_slot_handler("utility_request", self.handle_request)

    async def handle_request(self, envelope: Envelope) -> Envelope:
        """Handle a request requiring budget tracking and tool calls."""
        # 1. Start a budget tracking session
        async with self.budget.track("utility_task") as session:
            logger.info(f"Processing request with session: {session.session_id}")

            # 2. Conceptually call another capability via ToolBridge
            # (Note: Current core routing for this is in development, but SDK logic is shown)
            target = envelope.payload.get("use_tool", "none")

            if target != "none":
                logger.info(f"Requesting tool: {target}")
                # This would perform the Double Handshake (SPEC 04 Section 4.5)
                # result = await self.tools.call(target, {"param": "value"})
                # envelope.payload["tool_result"] = result

            # 3. Simulate resource usage
            await session.consume(tokens=50, energy=0.1)

            envelope.payload["status"] = "processed"
            envelope.payload["budget_remaining"] = self.budget.get_balance()

        return envelope

async def main():
    manifest_path = os.path.join(os.path.dirname(__file__), "manifest.yaml")
    manifest = Manifest.from_yaml(manifest_path)
    plugin = UtilityBridgePlugin(manifest)

    try:
        await plugin.run()
    except KeyboardInterrupt:
        await plugin.stop()

if __name__ == "__main__":
    asyncio.run(main())
