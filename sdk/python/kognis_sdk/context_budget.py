"""Context Budget Manager for LLM context window management.

Implements SPEC 10: Context Budget Manager. Ensures critical
information always fits in the model's context window.
"""

from __future__ import annotations

from dataclasses import dataclass
from enum import Enum
from typing import Any


class PriorityTier(Enum):
    MUST = "MUST"
    HIGH = "HIGH"
    MEDIUM = "MEDIUM"
    LOW = "LOW"


@dataclass
class ContextBlock:
    """A block of context with priority and estimated token count."""

    name: str
    content: str
    priority: PriorityTier
    token_count: int = 0

    def estimate_tokens(self, chars_per_token: float = 4.0) -> int:
        """Estimate token count from content length."""
        if self.token_count > 0:
            return self.token_count
        return int(len(self.content) / chars_per_token)


class ContextBudgetError(Exception):
    """Raised when context budget cannot be satisfied."""

    def __init__(self, code: str, message: str) -> None:
        self.code = code
        self.message = message
        super().__init__(f"[{code}] {message}")


@dataclass
class BudgetConfig:
    """Configuration for context budget management."""

    output_budget: int = 4000
    safety_margin: int = 500
    model_context_window: int = 128000
    trim_low_first: bool = True
    chars_per_token: float = 4.0


class ContextBudgetManager:
    """Manages LLM context window allocation and trimming.

    Spec reference: docs/spec/10-context-budget-manager.md

    Ensures critical information (MUST + HIGH tiers) always fits
    within the model's context window. Trims lower-priority
    blocks when budget is exceeded.
    """

    def __init__(self, config: BudgetConfig | None = None) -> None:
        self.config = config or BudgetConfig()
        self._trim_log: list[dict[str, Any]] = []

    def calculate_available_budget(self) -> int:
        """Calculate available input token budget.

        Spec reference: SPEC 10 Section 10.3

        Available = window - output_budget - safety_margin
        """
        return (
            self.config.model_context_window - self.config.output_budget - self.config.safety_margin
        )

    def assemble(
        self,
        blocks: list[ContextBlock],
        context_window: int | None = None,
    ) -> list[ContextBlock]:
        """Assemble context blocks within budget, trimming as needed.

        Spec reference: SPEC 10 Section 10.3

        Steps:
        1. Get target model's context window
        2. Reserve output budget and safety margin
        3. Calculate available input budget
        4. Assemble blocks by priority
        5. Trim LOW tier first, then MEDIUM
        6. Never trim MUST or HIGH
        7. Raise error if MUST+HIGH > budget
        """
        if context_window is not None:
            saved = self.config.model_context_window
            self.config.model_context_window = context_window

        budget = self.calculate_available_budget()

        if context_window is not None:
            self.config.model_context_window = saved

        for block in blocks:
            block.estimate_tokens(self.config.chars_per_token)

        must_blocks = [b for b in blocks if b.priority == PriorityTier.MUST]
        high_blocks = [b for b in blocks if b.priority == PriorityTier.HIGH]
        medium_blocks = [b for b in blocks if b.priority == PriorityTier.MEDIUM]
        low_blocks = [b for b in blocks if b.priority == PriorityTier.LOW]

        must_tokens = sum(b.token_count for b in must_blocks)
        high_tokens = sum(b.token_count for b in high_blocks)

        if must_tokens + high_tokens > budget:
            raise ContextBudgetError(
                "KGN-CONTEXT-TRIM_FAILED",
                f"MUST+HIGH ({must_tokens + high_tokens}) exceeds budget ({budget})",
            )

        result = list(must_blocks) + list(high_blocks)
        used = must_tokens + high_tokens

        remaining = budget - used

        for block in medium_blocks:
            if used + block.token_count <= budget:
                result.append(block)
                used += block.token_count
                remaining -= block.token_count
            else:
                self._log_trim(block, "medium", "budget_exceeded")

        for block in low_blocks:
            if used + block.token_count <= budget:
                result.append(block)
                used += block.token_count
            else:
                self._log_trim(block, "low", "budget_exceeded")

        return result

    def _log_trim(self, block: ContextBlock, tier: str, reason: str) -> None:
        """Log a trim action for adaptive feedback."""
        self._trim_log.append(
            {
                "block_name": block.name,
                "tier": tier,
                "reason": reason,
                "token_count": block.token_count,
            }
        )

    @property
    def trim_count(self) -> int:
        """Number of blocks trimmed in this session."""
        return len(self._trim_log)

    @property
    def trim_log(self) -> list[dict[str, Any]]:
        """Full trim history for adaptive feedback analysis."""
        return list(self._trim_log)

    def frequent_trimming(self, threshold: int = 5) -> bool:
        """Check if trimming is happening frequently.

        Spec reference: SPEC 10 Section 10.4

        When trimming happens frequently, system should adjust:
        - Memory plugin reduces default retrieval size
        - Environmental summary shortens
        - Cognitive Core may compact working memory
        """
        return self.trim_count > threshold
