from datetime import datetime
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field


class TransactionFeatures(BaseModel):
    """Validated features for one transaction: raw identifiers for rules plus engineered model inputs."""

    model_config = ConfigDict(frozen=True, extra="forbid", allow_inf_nan=False)

    # Raw features used by logging and pre-prediction rules, not by the model.
    TRANSACTION_ID: int
    TX_DATETIME: datetime
    CUSTOMER_ID: int
    TERMINAL_ID: int
    TX_TIME_SECONDS: int
    TX_TIME_DAYS: int

    # Engineered model features, in the column order the model expects.
    TX_AMOUNT: float
    TX_DURING_WEEKEND: Literal[0, 1]
    TX_DURING_NIGHT: Literal[0, 1]
    CUSTOMER_ID_NB_TX_1DAY_WINDOW: float
    CUSTOMER_ID_AVG_AMOUNT_1DAY_WINDOW: float
    CUSTOMER_ID_NB_TX_7DAY_WINDOW: float
    CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW: float
    CUSTOMER_ID_NB_TX_30DAY_WINDOW: float
    CUSTOMER_ID_AVG_AMOUNT_30DAY_WINDOW: float
    TERMINAL_ID_NB_TX_1DAY_WINDOW: float
    TERMINAL_ID_RISK_1DAY_WINDOW: float
    TERMINAL_ID_NB_TX_7DAY_WINDOW: float
    TERMINAL_ID_RISK_7DAY_WINDOW: float
    TERMINAL_ID_NB_TX_30DAY_WINDOW: float
    TERMINAL_ID_RISK_30DAY_WINDOW: float


RAW_FEATURES = frozenset(
    {"TRANSACTION_ID", "TX_DATETIME", "CUSTOMER_ID", "TERMINAL_ID", "TX_TIME_SECONDS", "TX_TIME_DAYS"}
)
MODEL_FEATURES = [name for name in TransactionFeatures.model_fields if name not in RAW_FEATURES]


class Prediction(BaseModel):
    is_fraud: bool
    fraud_probability: float = Field(..., ge=0, le=1)
