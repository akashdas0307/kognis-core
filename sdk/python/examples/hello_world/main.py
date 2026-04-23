import asyncio
import logging
import os
import sys

# Add the SDK directory to sys.path so we can import kognis_sdk
# This is useful when running the example from the repository
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "../..")))

from kognis_sdk.manifest import Manifest
from kognis_sdk.plugin import Plugin

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s"
)
logger = logging.getLogger("hello_world")

class HelloWorldPlugin(Plugin):
    """A simple plugin that says hello."""

    async def on_startup(self) -> None:
        logger.info("Hello World Plugin: on_startup called")
        logger.info("I am now running and emitting health pulses.")

    async def on_shutdown(self) -> None:
        logger.info("Hello World Plugin: on_shutdown called")

async def main():
    # Load manifest from the same directory as this script
    manifest_path = os.path.join(os.path.dirname(__file__), "manifest.yaml")
    manifest = Manifest.from_yaml(manifest_path)

    # Instantiate and start the plugin
    plugin = HelloWorldPlugin(manifest)

    logger.info("Starting HelloWorldPlugin...")
    try:
        await plugin.run()
    except KeyboardInterrupt:
        await plugin.stop()

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        pass
