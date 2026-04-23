class BeliefNetwork:
    """Represents a simplified cognitive belief network."""

    def __init__(self):
        self.beliefs = {} # statement -> confidence

    def update_belief(self, statement: str, delta: float):
        current = self.beliefs.get(statement, 0.5)
        self.beliefs[statement] = max(0, min(1.0, current + delta))

    def get_truth_score(self, statement: str) -> float:
        return self.beliefs.get(statement, 0.5)

    def get_summary(self) -> str:
        if not self.beliefs:
            return "No beliefs formed yet."
        top = sorted(self.beliefs.items(), key=lambda x: x[1], reverse=True)[0]
        return f"Primary belief: {top[0]} (conf: {top[1]:.2f})"
