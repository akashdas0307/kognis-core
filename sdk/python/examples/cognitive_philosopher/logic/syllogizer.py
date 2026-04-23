class Syllogizer:
    """Performs logical reasoning on beliefs."""

    @staticmethod
    def infer_truth(major_premise: bool, minor_premise: bool) -> float:
        """Simple logical implication simulation."""
        if major_premise and minor_premise:
            return 0.9
        if major_premise or minor_premise:
            return 0.4
        return 0.1
