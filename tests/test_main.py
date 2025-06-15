import pytest
from fastapi.testclient import TestClient
from unittest.mock import patch, MagicMock
import numpy as np

# Make the app directory importable
import sys
import os

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from app.main import app
from app.schema import TransactionFeatures


# A fixture to provide a test client for each test function
@pytest.fixture
def client():
    with TestClient(app) as c:
        yield c


# Sample legitimate transaction data for testing
sample_legitimate_payload = {
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

# --- Test Cases ---


def test_health_check(client):
    """
    Tests the /health endpoint.
    It should return {"status": "ok"} when the model is loaded.
    """
    with patch("app.main.model", new=MagicMock()):
        response = client.get("/health")
        assert response.status_code == 200
        assert response.json() == {"status": "ok"}


def test_health_check_no_model(client):
    """
    Tests the /health endpoint when the model is not loaded.
    """
    with patch("app.main.model", new=None):
        response = client.get("/health")
        assert response.status_code == 200
        assert response.json() == {"status": "service_up_no_model"}


@patch("app.main.model")
def test_predict_legitimate_transaction(mock_model, client):
    """
    Tests the /predict endpoint with a legitimate transaction.
    """
    mock_model.predict_proba.return_value = np.array([[0.9, 0.1]])

    response = client.post("/predict", json=sample_legitimate_payload)

    assert response.status_code == 200
    json_response = response.json()
    assert json_response["is_fraud"] is False
    assert json_response["fraud_probability"] == 0.1
    mock_model.predict_proba.assert_called_once()


@patch("app.main.model")
def test_predict_fraudulent_transaction(mock_model, client):
    """
    Tests the /predict endpoint with a transaction flagged as fraud.
    """
    mock_model.predict_proba.return_value = np.array([[0.2, 0.8]])

    response = client.post("/predict", json=sample_legitimate_payload)

    assert response.status_code == 200
    json_response = response.json()
    assert json_response["is_fraud"] is True
    assert json_response["fraud_probability"] == 0.8
    mock_model.predict_proba.assert_called_once()


def test_predict_blocked_customer(client):
    """
    Tests the pre-prediction check for blocked customers.
    """
    payload = sample_legitimate_payload.copy()
    payload["CUSTOMER_ID"] = 323
    response = client.post("/predict", json=payload)
    assert response.status_code == 403
    assert "blocked by issuer" in response.json()["detail"]


def test_predict_compromised_terminal(client):
    """
    Tests the pre-prediction check for high-risk terminals.
    """
    payload = sample_legitimate_payload.copy()
    payload["TERMINAL_ID"] = 4692
    response = client.post("/predict", json=payload)
    assert response.status_code == 403
    assert "High-risk terminal" in response.json()["detail"]


def test_predict_anomalous_amount(client):
    """
    Tests the pre-prediction rule for anomalous amounts.
    """
    payload = sample_legitimate_payload.copy()
    payload["CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW"] = 20.0
    payload["TX_AMOUNT"] = 150.0
    response = client.post("/predict", json=payload)
    assert response.status_code == 403
    assert "highly anomalous" in response.json()["detail"]


def test_predict_missing_feature(client):
    """
    Tests the API's response when a required feature is missing.
    """
    payload = sample_legitimate_payload.copy()
    del payload["TX_DURING_WEEKEND"]

    response = client.post("/predict", json=payload)

    assert response.status_code == 422
    response_json = response.json()
    assert response_json["detail"][0]["type"] == "missing"
    assert response_json["detail"][0]["loc"] == ["body", "TX_DURING_WEEKEND"]
    assert "Field required" in response_json["detail"][0]["msg"]


@patch("app.main.model", new=None)
def test_predict_model_not_loaded(client):
    """
    Tests the /predict endpoint when the model is not loaded.
    """
    response = client.post("/predict", json=sample_legitimate_payload)
    assert response.status_code == 503
    assert response.json()["detail"] == "Model not available"
