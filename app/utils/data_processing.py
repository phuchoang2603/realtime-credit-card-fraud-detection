import pandas as pd


def align_features_for_prediction(df: pd.DataFrame) -> pd.DataFrame:
    """
    Ensures the input DataFrame has the exact columns in the correct order
    that the model expects for a prediction. This assumes features are
    already engineered before being sent to the API.
    """
    expected_columns = [
        "TX_AMOUNT",
        "TX_DURING_WEEKEND",
        "TX_DURING_NIGHT",
        "CUSTOMER_ID_NB_TX_1DAY_WINDOW",
        "CUSTOMER_ID_AVG_AMOUNT_1DAY_WINDOW",
        "CUSTOMER_ID_NB_TX_7DAY_WINDOW",
        "CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW",
        "CUSTOMER_ID_NB_TX_30DAY_WINDOW",
        "CUSTOMER_ID_AVG_AMOUNT_30DAY_WINDOW",
        "TERMINAL_ID_NB_TX_1DAY_WINDOW",
        "TERMINAL_ID_RISK_1DAY_WINDOW",
        "TERMINAL_ID_NB_TX_7DAY_WINDOW",
        "TERMINAL_ID_RISK_7DAY_WINDOW",
        "TERMINAL_ID_NB_TX_30DAY_WINDOW",
        "TERMINAL_ID_RISK_30DAY_WINDOW",
    ]

    # This function now only validates and reorders columns.
    try:
        df = df[expected_columns]
    except KeyError as e:
        # Provides a more informative error if the client sends a payload
        # with missing pre-engineered features.
        raise ValueError(f"Input data is missing required feature columns: {e}")

    return df
