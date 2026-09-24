import pandas as pd

from app.schema import MODEL_FEATURES


def align_features_for_prediction(df: pd.DataFrame) -> pd.DataFrame:
    """Select the engineered model columns in the order the model expects.

    Features arrive already engineered; this only validates presence and ordering.
    """
    try:
        return df[MODEL_FEATURES]
    except KeyError as exc:
        raise ValueError(f"Input data is missing required feature columns: {exc}") from exc
