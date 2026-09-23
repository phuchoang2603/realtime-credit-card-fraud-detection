import numpy as np
import pandas as pd
import pytest
from structlog import get_logger

from app.application import FraudApplication
from app.errors import ModelPredictionError
from app.schema import TransactionFeatures
from app.utils.data_preprocessing import align_features_for_prediction
from app.utils.metrics_config import NoopMetrics


@pytest.mark.parametrize(
    "probability,expected",
    [(0.1, False), (0.5, False), (0.8, True)],
    ids=["below-threshold", "at-threshold", "above-threshold"],
)
def test_prediction_threshold(sample_legitimate_payload, probability, expected):
    class Model:
        def predict_proba(self, features):
            return np.array([[1 - probability, probability]])

    result = FraudApplication(Model(), NoopMetrics(), get_logger()).predict(
        TransactionFeatures.model_validate(sample_legitimate_payload), "threshold"
    )
    assert result.is_fraud is expected
    assert result.fraud_probability == probability


def test_feature_alignment_rejects_missing_columns():
    with pytest.raises(ValueError, match="missing required feature columns"):
        align_features_for_prediction(pd.DataFrame({"TX_AMOUNT": [1.0]}))


@pytest.mark.parametrize("probability", [float("nan"), 1.1], ids=["nonfinite", "out-of-range"])
def test_invalid_model_output_is_rejected(sample_legitimate_payload, probability):
    class Model:
        def predict_proba(self, features):
            return np.array([[0, probability]])

    with pytest.raises(ModelPredictionError):
        FraudApplication(Model(), NoopMetrics(), get_logger()).predict(
            TransactionFeatures.model_validate(sample_legitimate_payload), "invalid-output"
        )
