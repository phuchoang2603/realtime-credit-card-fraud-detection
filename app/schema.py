from pydantic import BaseModel, Field
from typing import Literal


class TransactionFeatures(BaseModel):
    """
    Defines the input features for a single transaction prediction, based on the
    feature engineering pipeline from the handbook.
    """

    TX_AMOUNT: float = Field(
        ..., example=100.50, description="The monetary value of the transaction."
    )
    TX_DURING_WEEKEND: Literal[0, 1] = Field(
        ...,
        example=1,
        description="1 if the transaction occurs on a weekend, 0 otherwise.",
    )
    TX_DURING_NIGHT: Literal[0, 1] = Field(
        ...,
        example=0,
        description="1 if the transaction occurs during the night (0pm-6am), 0 otherwise.",
    )

    # Customer-level features
    CUSTOMER_ID_NB_TX_1DAY_WINDOW: float = Field(..., example=1.0)
    CUSTOMER_ID_AVG_AMOUNT_1DAY_WINDOW: float = Field(..., example=75.25)
    CUSTOMER_ID_NB_TX_7DAY_WINDOW: float = Field(..., example=5.0)
    CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW: float = Field(..., example=90.0)
    CUSTOMER_ID_NB_TX_30DAY_WINDOW: float = Field(..., example=20.0)
    CUSTOMER_ID_AVG_AMOUNT_30DAY_WINDOW: float = Field(..., example=85.50)

    # Terminal-level features
    TERMINAL_ID_NB_TX_1DAY_WINDOW: float = Field(..., example=10.0)
    TERMINAL_ID_RISK_1DAY_WINDOW: float = Field(..., example=0.2)
    TERMINAL_ID_NB_TX_7DAY_WINDOW: float = Field(..., example=50.0)
    TERMINAL_ID_RISK_7DAY_WINDOW: float = Field(..., example=0.1)
    TERMINAL_ID_NB_TX_30DAY_WINDOW: float = Field(..., example=200.0)
    TERMINAL_ID_RISK_30DAY_WINDOW: float = Field(..., example=0.05)


class Prediction(BaseModel):
    """
    Defines the structure of the API's prediction response.
    """

    is_fraud: bool
    fraud_probability: float = Field(..., ge=0, le=1)
