import collections

import numpy as np


class AudioBuffer:
    """Manages a sliding window of audio samples."""

    def __init__(self, window_size: int = 1024):
        self.buffer = collections.deque(maxlen=window_size)
        self.window_size = window_size

    def add_samples(self, samples: list[float]):
        self.buffer.extend(samples)

    def get_window(self) -> np.ndarray:
        """Returns the current window as a numpy array."""
        if len(self.buffer) < self.window_size:
            return np.zeros(self.window_size)
        return np.array(list(self.buffer))

    @property
    def is_full(self) -> bool:
        return len(self.buffer) >= self.window_size
