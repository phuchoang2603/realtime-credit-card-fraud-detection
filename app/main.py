import asyncio
import contextlib
import os
import pickle
import time

import pandas as pd
from fastapi import FastAPI, HTTPException, Request
from opentelemetry import trace

# Import configurations and schemas from separate modules
from app.schema import TransactionFeatures, Prediction
from app.utils.logging_config import setup_logging, get_logger
from app.utils.tracing_config import (
    setup_tracing,
    get_tracer,
    traceable,
)
from app.utils.metrics_config import (
    predictions_counter,
    prediction_latency,
    fraud_score_histogram,
)
from app.utils.data_preprocessing import align_features_for_prediction
from app.utils.pre_prediction_checks import (
    run_terminal_control_check,
    run_transaction_blocking_rules,
)

# --- Model and Application Setup ---
setup_logging()
log = get_logger(__name__)
model = None
DEFAULT_MODEL_PATH = os.path.join(
    os.path.dirname(__file__), "..", "models", "model.pkl"
)
MODEL_PATH = os.environ.get("MODEL_PATH", DEFAULT_MODEL_PATH)


@contextlib.asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Manages the application's lifespan events for model loading and cleanup.
    """
    global model
    try:
        with open(MODEL_PATH, "rb") as f:
            model = pickle.load(f)
        log.info("Model loaded successfully.", path=MODEL_PATH)
    except Exception as e:
        log.error("Error loading model.", error=str(e), path=MODEL_PATH)
        model = None
    yield
    # Clean up the ML model and release the resources
    log.info("Clearing model.")
    model = None


app = FastAPI(
    title="Fraud Detection API",
    description="An API to predict credit card transaction fraud.",
    lifespan=lifespan,
)
setup_tracing(app, service_name="fraud-detection-api")
tracer = get_tracer(__name__)


# --- Helper function for model prediction ---
@traceable
async def run_model_prediction(
    transaction: TransactionFeatures, request_id: str
) -> Prediction:
    """Runs the ML model inference in its own trace span."""
    if model is None:
        log.error("Prediction failed: Model not loaded.", request_id=request_id)
        raise HTTPException(status_code=503, detail="Model not available")

    start_time = time.time()
    try:
        input_df = pd.DataFrame([transaction.dict()])
        processed_df = align_features_for_prediction(input_df)
        fraud_probability = model.predict_proba(processed_df)[:, 1][0]
        is_fraud = bool(fraud_probability > 0.5)

        predictions_counter.add(1, {"is_fraud": str(is_fraud)})
        fraud_score_histogram.record(fraud_probability)
        log.info(
            "Prediction successful",
            request_id=request_id,
            is_fraud=is_fraud,
            fraud_probability=float(fraud_probability),
        )
        return Prediction(is_fraud=is_fraud, fraud_probability=fraud_probability)
    except Exception as e:
        log.error("Prediction error", error=str(e), request_id=request_id)
        raise  # The decorator will capture and record the exception
    finally:
        latency = time.time() - start_time
        prediction_latency.record(latency)


# --- API Endpoints ---
@app.get("/health", tags=["Monitoring"])
async def health_check():
    """Health check endpoint to ensure the service is running."""
    return {"status": "ok" if model is not None else "service_up_no_model"}


@app.post("/predict", response_model=Prediction, tags=["Prediction"])
@traceable
async def predict_fraud(request: Request, transaction: TransactionFeatures):
    """
    Orchestrates the fraud detection process by calling traceable helper functions.
    """
    request_id = request.headers.get("X-Request-ID", "N/A")

    log.info(
        "Received prediction request",
        request_id=request_id,
        transaction_id=transaction.TRANSACTION_ID,
        customer_id=transaction.CUSTOMER_ID,
        terminal_id=transaction.TERMINAL_ID,
    )

    span = trace.get_current_span()
    span.set_attribute("transaction_id", transaction.TRANSACTION_ID)
    span.set_attribute("customer_id", transaction.CUSTOMER_ID)
    span.set_attribute("terminal_id", transaction.TERMINAL_ID)

    # 1. Run Pre-Prediction Checks
    await asyncio.gather(
        run_terminal_control_check(customer_id=transaction.CUSTOMER_ID),
        run_transaction_blocking_rules(transaction=transaction),
    )

    # 2. Run Model Prediction
    prediction = await run_model_prediction(
        transaction=transaction, request_id=request_id
    )

    return prediction
