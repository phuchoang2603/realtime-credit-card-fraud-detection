from __future__ import annotations

import math
import time
from typing import Any

import pandas as pd

from app.errors import ModelPredictionError, ModelUnavailableError
from app.model import PredictionModel
from app.schema import Prediction, TransactionFeatures
from app.utils.data_preprocessing import align_features_for_prediction
from app.utils.pre_prediction_checks import check_customer, check_transaction


class FraudApplication:
    """Application service that coordinates rules, preprocessing and inference."""

    def __init__(self, model: PredictionModel | None, metrics: Any, logger: Any):
        self.model = model
        self.metrics = metrics
        self.logger = logger

    def predict(self, transaction: TransactionFeatures, request_id: str) -> Prediction:
        if self.model is None:
            raise ModelUnavailableError

        check_customer(transaction.CUSTOMER_ID)
        check_transaction(transaction)

        started = time.perf_counter()
        try:
            features = align_features_for_prediction(pd.DataFrame([transaction.model_dump()]))
            probabilities = self.model.predict_proba(features)
            probability = float(probabilities[:, 1][0])
            if not math.isfinite(probability) or not 0 <= probability <= 1:
                raise ValueError("Model returned an invalid probability")
        except Exception as exc:
            raise ModelPredictionError from exc
        finally:
            self.metrics.observe_latency(time.perf_counter() - started)

        is_fraud = probability > 0.5
        self.metrics.record_prediction(is_fraud, probability)
        self.logger.info(
            "Prediction successful",
            request_id=request_id,
            is_fraud=is_fraud,
            fraud_probability=probability,
        )
        return Prediction(is_fraud=is_fraud, fraud_probability=probability)
