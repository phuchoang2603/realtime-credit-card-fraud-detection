class FraudRuleError(Exception):
    """A safe, client-visible rule rejection."""


class BlockedCustomerError(FraudRuleError):
    def __init__(self, customer_id: int):
        super().__init__(f"Transaction blocked by issuer: Customer account {customer_id} is under review.")


class CompromisedTerminalError(FraudRuleError):
    def __init__(self, terminal_id: int):
        super().__init__(f"Transaction blocked: High-risk terminal ID {terminal_id}.")


class AnomalousAmountError(FraudRuleError):
    def __init__(self):
        super().__init__("Transaction blocked: Amount is highly anomalous.")


class ModelUnavailableError(Exception):
    """Raised when prediction cannot run because the model is unavailable."""


class ModelPredictionError(Exception):
    """Raised when a loaded model fails to produce a prediction."""
