import asyncio
import logging
import os
import random
import sys

# Standard Kognis SDK boilerplate
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "../..")))

from logic.syllogizer import Syllogizer

# Import local sub-modules
from models.belief_network import BeliefNetwork

from kognis_sdk.manifest import Manifest
from kognis_sdk.stateful_agent import StatefulAgent

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
logger = logging.getLogger("cognitive_philosopher")

class PhilosopherAgent(StatefulAgent):
    """A complex stateful agent split across modules."""

    async def on_startup(self) -> None:
        logger.info("Philosopher Agent: Deep thought sequence INITIALIZING")
        self.beliefs = BeliefNetwork()
        self.reasoner = Syllogizer()

        # Initial core beliefs
        self.beliefs.update_belief("I exist", 0.5)
        self.beliefs.update_belief("The world is digital", 0.2)

    async def cognition_cycle(self) -> None:
        """Modular cognition cycle."""
        await super().cognition_cycle()

        # 1. Modular reasoning
        major = self.beliefs.get_truth_score("I exist") > 0.7
        minor = random.random() > 0.5

        truth_delta = self.reasoner.infer_truth(major, minor)
        self.beliefs.update_belief("I am thinking", truth_delta - 0.5)

        # 2. State broadcast
        if self._cycle_count % 10 == 0:
            summary = self.beliefs.get_summary()
            logger.info(f"Philosopher Summary: {summary}")
            await self.emit_event("kognis.agent.philosophy_update", {
                "summary": summary,
                "cycle": self._cycle_count
            })

        await asyncio.sleep(1)

async def main():
    manifest_path = os.path.join(os.path.dirname(__file__), "manifest.yaml")
    manifest = Manifest.from_yaml(manifest_path)
    agent = PhilosopherAgent(manifest)

    try:
        await agent.run()
    except KeyboardInterrupt:
        await agent.stop()

if __name__ == "__main__":
    asyncio.run(main())
