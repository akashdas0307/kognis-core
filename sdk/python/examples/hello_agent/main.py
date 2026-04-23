import asyncio
import logging
import os
import random
import sys

# Add the SDK directory to sys.path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "../..")))

from kognis_sdk.manifest import Manifest
from kognis_sdk.stateful_agent import StatefulAgent

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
logger = logging.getLogger("hello_agent")

class HelloAgent(StatefulAgent):
    """A stateful agent that maintains an internal 'mood' and processes cognition cycles."""

    async def on_startup(self) -> None:
        logger.info("Hello Agent: Initializing internal state...")
        self.working_memory["mood"] = "neutral"
        self.working_memory["thoughts"] = []

    async def cognition_cycle(self) -> None:
        """One iteration of the agent's thinking process."""
        # Call the parent cycle to increment cycle count
        await super().cognition_cycle()

        # Simulate thinking
        moods = ["curious", "happy", "contemplative", "focused", "dreamy"]
        current_mood = random.choice(moods)
        self.working_memory["mood"] = current_mood

        thought = f"Cycle {self._cycle_count}: I am feeling {current_mood} today."
        self.working_memory["thoughts"].append(thought)

        logger.info(f"Agent thought: {thought}")

        # Emit state change every 5 cycles
        if self._cycle_count % 5 == 0:
            await self.emit_event("kognis.agent.state_change", {
                "agent_id": self.plugin_id,
                "current_mood": current_mood,
                "cycles": self._cycle_count
            })

        # Slow down the loop for the example
        await asyncio.sleep(2)

async def main():
    manifest_path = os.path.join(os.path.dirname(__file__), "plugin.yaml")
    logger.info(f"Loading manifest from {manifest_path}")
    manifest = Manifest.from_yaml(manifest_path)
    agent = HelloAgent(manifest)

    logger.info("Starting HelloAgent continuous cognition loop...")
    try:
        await agent.run()
    except Exception as e:
        logger.error(f"Agent crashed: {e}", exc_info=True)
        await agent.stop()

if __name__ == "__main__":
    asyncio.run(main())
