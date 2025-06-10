import os
import pickle
import pandas as pd
import time
from fastapi import FastAPI, Request, HTTPException

# Import configurations and schemas from separate modules
from app.schema import TransactionFeatures, Prediction
from app.utils.logging_config import setup_logging, get_logger
from app.utils.tracing_config import setup_tracing, get_tracer
from app.utils.metrics_config import (
    predictions_counter,
    prediction_latency,
    fraud_score_histogram,
)

# Updated import to use the simplified preprocessing function
from app.utils.data_preprocessing import align_features_for_prediction

# --- Application Setup ---
setup_logging()
log = get_logger(__name__)
app = FastAPI(
    title="Fraud Detection API",
    description="An API to predict credit card transaction fraud.",
)
setup_tracing(app, service_name="fraud-detection-api")
tracer = get_tracer(__name__)

# --- Model Loading ---
MODEL_PATH = os.environ.get("MODEL_PATH", "./models/model.pkl")
model = None
try:
    with open(MODEL_PATH, "rb") as f:
        model = pickle.load(f)
    log.info("Model loaded successfully.", path=MODEL_PATH)
except FileNotFoundError:
    log.error("Model file not found.", path=MODEL_PATH)
except Exception as e:
    log.error("Error loading model.", error=str(e), path=MODEL_PATH)


# --- API Endpoints ---
@app.get("/health", tags=["Monitoring"])
async def health_check():
    """Health check endpoint to ensure the service is running."""
    return {"status": "ok" if model is not None else "service_up_no_model"}


@app.post("/predict", response_model=Prediction, tags=["Prediction"])
async def predict_fraud(request: Request, transaction: TransactionFeatures):
    """
    Accepts pre-engineered transaction features and returns a fraud prediction.
    """
    request_id = request.headers.get("X-Request-ID", "N/A")
    with tracer.start_as_current_span("prediction_request") as span:
        span.set_attribute("request_id", request_id)

        if model is None:
            log.error("Prediction failed: Model not loaded.", request_id=request_id)
            raise HTTPException(status_code=503, detail="Model not available")

        log.info("Received prediction request", request_id=request_id)

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
            span.set_attribute("prediction.is_fraud", is_fraud)
            span.set_attribute("prediction.probability", float(fraud_probability))

            return Prediction(is_fraud=is_fraud, fraud_probability=fraud_probability)

        except Exception as e:
            log.error("Prediction error", error=str(e), request_id=request_id)
            span.record_exception(e)
            raise HTTPException(status_code=500, detail=f"Internal server error: {e}")
        finally:
            # This block will always run, ensuring latency is recorded
            # even if an error occurs.
            latency = time.time() - start_time
            prediction_latency.record(latency)
            # --- END OF CORRECTION ---
