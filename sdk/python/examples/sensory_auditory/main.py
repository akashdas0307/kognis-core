import asyncio
import logging
import os
import sys

# Standard Kognis SDK boilerplate
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "../..")))

# Import local modules
from buffer import AudioBuffer
from processor import AudioProcessor

from kognis_sdk.envelope import Envelope
from kognis_sdk.manifest import Manifest
from kognis_sdk.plugin import Plugin

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
logger = logging.getLogger("sensory_auditory")

class AuditoryPlugin(Plugin):
    """A complex auditory sensor plugin using modular internal logic."""

    async def on_startup(self) -> None:
        logger.info("Auditory System: INITIALIZING")
        self.audio_buffer = AudioBuffer(window_size=2048)
        self.processor = AudioProcessor(sample_rate=16000)

        # Register for voice capture slot
        self.register_slot_handler("voice_capture", self.handle_audio_stream)

    async def handle_audio_stream(self, envelope: Envelope) -> Envelope:
        """Processes a chunk of audio samples from the pipeline."""
        samples = envelope.payload.get("samples", [])

        # 1. Update internal modular buffer
        self.audio_buffer.add_samples(samples)

        if self.audio_buffer.is_full:
            # 2. Extract features using internal processor
            window = self.audio_buffer.get_window()
            features = self.processor.extract_features(window)
            phoneme = self.processor.detect_phoneme(features)

            if phoneme:
                logger.info(f"Detected acoustic event: {phoneme}")
                envelope.payload["phoneme"] = phoneme

            envelope.payload["acoustic_features"] = features

        return envelope

async def main():
    manifest_path = os.path.join(os.path.dirname(__file__), "manifest.yaml")
    manifest = Manifest.from_yaml(manifest_path)
    plugin = AuditoryPlugin(manifest)

    try:
        await plugin.run()
    except KeyboardInterrupt:
        await plugin.stop()

if __name__ == "__main__":
    asyncio.run(main())
