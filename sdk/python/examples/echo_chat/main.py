import asyncio
import logging
import os
import sys

# Add the SDK directory to sys.path so we can import kognis_sdk
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "../..")))

from kognis_sdk.envelope import Envelope
from kognis_sdk.manifest import Manifest
from kognis_sdk.plugin import Plugin

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
logger = logging.getLogger("echo_chat")

class EchoChatPlugin(Plugin):
    """A stateless plugin that echoes back any text it receives."""

    async def on_startup(self) -> None:
        logger.info("Echo Chat Plugin: online")
        # Register a handler for the 'chat_input' slot
        self.register_slot_handler("chat_input", self.handle_chat)

    async def handle_chat(self, envelope: Envelope) -> Envelope:
        """Process an incoming chat message."""
        text = envelope.payload.get("text", "")
        logger.info(f"Received chat: {text}")

        # Create a response payload
        response_text = f"You said: {text}"

        # create_envelope helper is useful here
        # In a real scenario, we might modify the existing envelope or create a new one
        envelope.payload["echo"] = response_text
        envelope.payload["processed_by"] = self.plugin_id

        return envelope

async def main():
    manifest_path = os.path.join(os.path.dirname(__file__), "manifest.yaml")
    manifest = Manifest.from_yaml(manifest_path)
    plugin = EchoChatPlugin(manifest)

    try:
        await plugin.run()
    except KeyboardInterrupt:
        await plugin.stop()

if __name__ == "__main__":
    asyncio.run(main())
