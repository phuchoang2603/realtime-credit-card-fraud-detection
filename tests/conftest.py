import pytest


@pytest.fixture
def sample_legitimate_payload():
    """A pytest fixture to provide sample legitimate transaction data."""
    return {
        "TRANSACTION_ID": 1,
        "TX_DATETIME": "2025-06-12T10:00:00Z",
        "CUSTOMER_ID": 1001,
        "TERMINAL_ID": 2001,
        "TX_TIME_SECONDS": 1749615600,
        "TX_TIME_DAYS": 20250,
        "TX_AMOUNT": 75.50,
        "TX_DURING_WEEKEND": 0,
        "TX_DURING_NIGHT": 0,
        "CUSTOMER_ID_NB_TX_1DAY_WINDOW": 2.0,
        "CUSTOMER_ID_AVG_AMOUNT_1DAY_WINDOW": 75.50,
        "CUSTOMER_ID_NB_TX_7DAY_WINDOW": 10.0,
        "CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW": 80.0,
        "CUSTOMER_ID_NB_TX_30DAY_WINDOW": 30.0,
        "CUSTOMER_ID_AVG_AMOUNT_30DAY_WINDOW": 85.0,
        "TERMINAL_ID_NB_TX_1DAY_WINDOW": 50.0,
        "TERMINAL_ID_RISK_1DAY_WINDOW": 0.1,
        "TERMINAL_ID_NB_TX_7DAY_WINDOW": 350.0,
        "TERMINAL_ID_RISK_7DAY_WINDOW": 0.15,
        "TERMINAL_ID_NB_TX_30DAY_WINDOW": 1500.0,
        "TERMINAL_ID_RISK_30DAY_WINDOW": 0.12,
    }
