import random

import numpy as np


class AudioProcessor:
    """Simulates acoustic analysis and phoneme detection."""

    def __init__(self, sample_rate: int = 16000):
        self.sample_rate = sample_rate

    def extract_features(self, window: np.ndarray) -> dict:
        """Simulates FFT and spectral energy extraction."""
        energy = np.sum(np.square(window))
        return {
            "energy": float(energy),
            "peak_freq": random.uniform(200, 4000),
            "timestamp_offset": random.random()
        }

    def detect_phoneme(self, features: dict) -> str | None:
        """Mock phoneme classifier based on spectral energy."""
        if features["energy"] > 0.5:
            # Randomly pick a phoneme for simulation
            return random.choice(["/a/", "/e/", "/s/", "/t/"])
        return None
