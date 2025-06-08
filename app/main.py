import os
import time
import joblib
from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from fastapi.encoders import jsonable_encoder
from schema import HouseInfo, HousePrediction
from utils.data_processing import format_input_data
from utils.logging import logger
from utils.middleware import LogMiddleware, RequestIDMiddleware
from utils.metrics import request_counter, predict_duration

app = FastAPI()
app.add_middleware(RequestIDMiddleware)
app.add_middleware(LogMiddleware)

# Load model
model_path = os.environ.get("MODEL_PATH", "../models/model.pkl")
logger.debug("Loading model", extra={"model_path": model_path})
clf = joblib.load(model_path)
logger.debug("Model loaded successfully")


@app.post("/predict", response_model=HousePrediction)
def predict(data: HouseInfo, request: Request):
    start_time = time.time()

    try:
        logger.info(
            f"predict_received lotarea={data.LotArea} yearbuilt={data.YearBuilt} zoning={data.MSZoning}"
        )

        formatted_data = format_input_data(data)
        logger.debug(f"{formatted_data.values.tolist()}")

        price = clf.predict(formatted_data)[0]
        duration = round((time.time() - start_time) * 1000, 2)

        # Record OTEL metrics
        request_counter.add(
            1, {"method": "POST", "endpoint": "/predict", "http_status": "200"}
        )
        predict_duration.record(duration, {"endpoint": "/predict"})

        logger.info(f"predict_success price={price:.2f} duration={duration}ms")

        return HousePrediction(Price=price)

    except Exception as e:
        logger.exception(f"predict_failed error={str(e)}")
        raise


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError):
    # Record OTEL metrics
    request_counter.add(
        1,
        {"method": request.method, "endpoint": request.url.path, "http_status": "422"},
    )

    logger.error(
        "Validation failed",
        extra={
            "error": jsonable_encoder(exc.errors()),
            "req": {
                "method": request.method,
                "url": str(request.url),
                "client": request.client.host,
            },
            "res": {
                "status_code": 422,
            },
        },
    )
    return JSONResponse(
        status_code=422,
        content={"detail": exc.errors()},
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=30000, log_config=None)
