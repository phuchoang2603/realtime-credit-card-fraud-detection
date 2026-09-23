import numpy as np
import pandas as pd
import pytest

from app.application import FraudApplication
from app.schema import TransactionFeatures
from app.utils.data_preprocessing import align_features_for_prediction


class FakeModel:
    def __init__(self, probability: float):
        self.probability = probability
        self.calls = 0

    def predict_proba(self, features):
        self.calls += 1
        assert list(features.columns)[0] == "TX_AMOUNT"
        return np.array([[1 - self.probability, self.probability]])


class FailingModel:
    def predict_proba(self, features):
        del features
        raise RuntimeError("backend credentials should never reach the caller")


class RecordingMetrics:
    def __init__(self):
        self.latencies = []
        self.predictions = []

    def observe_latency(self, seconds):
        self.latencies.append(seconds)

    def record_prediction(self, is_fraud, probability):
        self.predictions.append((is_fraud, probability))


class RecordingLogger:
    def __init__(self):
        self.events = []

    def info(self, event, **kwargs):
        self.events.append((event, kwargs))


@pytest.mark.parametrize(
    "probability,expected",
    [(0.1, False), (0.5, False), (0.8, True)],
    ids=["below-threshold", "at-threshold", "above-threshold"],
)
def test_prediction_threshold_partitions(sample_legitimate_payload, probability, expected):
    result = FraudApplication(FakeModel(probability), RecordingMetrics(), RecordingLogger()).predict(
        TransactionFeatures.model_validate(sample_legitimate_payload), "threshold"
    )
    assert result.is_fraud is expected
    assert result.fraud_probability == probability


def test_feature_alignment_reports_missing_preengineered_columns():
    with pytest.raises(ValueError, match="missing required feature columns"):
        align_features_for_prediction(pd.DataFrame({"TX_AMOUNT": [1.0]}))


def test_application_records_duration_metrics_and_structured_outcome(monkeypatch, sample_legitimate_payload):
    metrics = RecordingMetrics()
    logger = RecordingLogger()
    clock = iter([100.0, 100.25])
    monkeypatch.setattr("app.application.time.perf_counter", lambda: next(clock))

    prediction = FraudApplication(FakeModel(0.8), metrics, logger).predict(
        TransactionFeatures.model_validate(sample_legitimate_payload), "request-9"
    )

    assert prediction.is_fraud is True
    assert prediction.fraud_probability == 0.8
    assert metrics.latencies == [0.25]
    assert metrics.predictions == [(True, 0.8)]
    assert logger.events == [
        (
            "Prediction successful",
            {"request_id": "request-9", "is_fraud": True, "fraud_probability": 0.8},
        )
    ]


@pytest.mark.parametrize("probability", [float("nan"), 1.1], ids=["nonfinite", "out-of-range"])
def test_invalid_model_probability_is_rejected(sample_legitimate_payload, probability):
    from app.errors import ModelPredictionError

    with pytest.raises(ModelPredictionError):
        FraudApplication(FakeModel(probability), RecordingMetrics(), RecordingLogger()).predict(
            TransactionFeatures.model_validate(sample_legitimate_payload), "invalid-output"
        )
