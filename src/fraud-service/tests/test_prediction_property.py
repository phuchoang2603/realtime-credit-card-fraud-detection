import hashlib
import os
import platform
from pathlib import Path

import pytest
import sklearn
from hypothesis import given, settings
from hypothesis import strategies as st
from structlog import get_logger

from app.application import FraudApplication
from app.model import load_model
from app.schema import TransactionFeatures
from app.utils.metrics_config import NoopMetrics

pytestmark = pytest.mark.property
settings.register_profile("ci", max_examples=16, deadline=None, derandomize=True)
settings.register_profile("investigation", max_examples=250, deadline=None)
settings.load_profile(os.environ.get("HYPOTHESIS_PROFILE", "ci"))


def find_model_path() -> Path:
    for parent in Path(__file__).resolve().parents:
        candidate = parent / "models" / "model.pkl"
        if candidate.exists():
            return candidate
    raise FileNotFoundError("models/model.pkl")


MODEL_PATH = find_model_path()
_MODEL = None


def model_once():
    global _MODEL
    if _MODEL is None:
        _MODEL = load_model(MODEL_PATH)
        digest = hashlib.sha256(MODEL_PATH.read_bytes()).hexdigest()
        print(
            f"model_sha256={digest} python={platform.python_version()} sklearn={sklearn.__version__} tolerance=1e-12 profile={settings.get_current_profile_name()}"
        )
    return _MODEL


def feature_payloads():
    numeric = st.floats(min_value=0, max_value=1000, allow_nan=False, allow_infinity=False)
    return st.fixed_dictionaries(
        {
            "TRANSACTION_ID": st.integers(min_value=1, max_value=100000),
            "TX_DATETIME": st.just("2025-06-12T10:00:00Z"),
            "CUSTOMER_ID": st.just(1001),
            "TERMINAL_ID": st.just(2001),
            "TX_TIME_SECONDS": st.integers(min_value=1, max_value=2000000000),
            "TX_TIME_DAYS": st.integers(min_value=1, max_value=50000),
            "TX_AMOUNT": st.floats(min_value=0, max_value=100, allow_nan=False, allow_infinity=False),
            "TX_DURING_WEEKEND": st.integers(min_value=0, max_value=1),
            "TX_DURING_NIGHT": st.integers(min_value=0, max_value=1),
            "CUSTOMER_ID_NB_TX_1DAY_WINDOW": numeric,
            "CUSTOMER_ID_AVG_AMOUNT_1DAY_WINDOW": numeric,
            "CUSTOMER_ID_NB_TX_7DAY_WINDOW": numeric,
            "CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW": st.just(1000.0),
            "CUSTOMER_ID_NB_TX_30DAY_WINDOW": numeric,
            "CUSTOMER_ID_AVG_AMOUNT_30DAY_WINDOW": numeric,
            "TERMINAL_ID_NB_TX_1DAY_WINDOW": numeric,
            "TERMINAL_ID_RISK_1DAY_WINDOW": st.floats(min_value=0, max_value=1, allow_nan=False),
            "TERMINAL_ID_NB_TX_7DAY_WINDOW": numeric,
            "TERMINAL_ID_RISK_7DAY_WINDOW": st.floats(min_value=0, max_value=1, allow_nan=False),
            "TERMINAL_ID_NB_TX_30DAY_WINDOW": numeric,
            "TERMINAL_ID_RISK_30DAY_WINDOW": st.floats(min_value=0, max_value=1, allow_nan=False),
        }
    )


@given(feature_payloads())
def test_real_model_prediction_is_repeatable(payload):
    model = model_once()
    service = FraudApplication(model, NoopMetrics(), get_logger("property-test"))
    transaction = TransactionFeatures.model_validate(payload)
    original = transaction.model_dump()

    first = service.predict(transaction, "property-test")
    second = service.predict(transaction, "property-test")

    assert first.is_fraud == second.is_fraud
    assert abs(first.fraud_probability - second.fraud_probability) <= 1e-12
    assert transaction.model_dump() == original
    assert 0 <= first.fraud_probability <= 1
