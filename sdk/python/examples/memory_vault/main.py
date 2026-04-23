import asyncio
import logging
import os
import sys

# Add the SDK directory to sys.path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "../..")))

from kognis_sdk.manifest import Manifest
from kognis_sdk.state_store import StateStore
from kognis_sdk.stateful_agent import StatefulAgent

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
logger = logging.getLogger("memory_vault")

class MemoryVaultAgent(StatefulAgent):
    """A stateful agent that uses StateStore for persistent memory across restarts."""

    async def on_startup(self) -> None:
        logger.info("Memory Vault: Accessing persistent state storage...")
        # StateStore in current SDK version uses filesystem fallback
        self.storage = StateStore(plugin_id=self.plugin_id)

        # Load previous memory if it exists
        try:
            stored_data = await self.storage.get("working_memory")
            if stored_data:
                self._working_memory = stored_data
                logger.info(f"Memory Vault: Restored {len(self._working_memory)} keys from storage.")
        except Exception as e:
            logger.warning(f"Memory Vault: Could not load state: {e}")

    async def cognition_cycle(self) -> None:
        await super().cognition_cycle()

        # Increment a persistent counter
        count = self.working_memory.get("persistent_cycle_count", 0)
        self.working_memory["persistent_cycle_count"] = count + 1

        # Save state every 10 cycles
        if self._cycle_count % 10 == 0:
            logger.info(f"Memory Vault: Persisting state at cycle {self._cycle_count}...")
            await self.storage.put("working_memory", self.working_memory)

        await asyncio.sleep(1)

async def main():
    manifest_path = os.path.join(os.path.dirname(__file__), "manifest.yaml")
    manifest = Manifest.from_yaml(manifest_path)
    agent = MemoryVaultAgent(manifest)

    try:
        await agent.run()
    except KeyboardInterrupt:
        await agent.stop()

if __name__ == "__main__":
    asyncio.run(main())
