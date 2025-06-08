import logging
import sys
import os
import json
from logging import Formatter
from contextvars import ContextVar

request_id_ctx_var: ContextVar[str] = ContextVar("request_id", default="unknown")


def set_request_id(request_id: str):
    request_id_ctx_var.set(request_id)


def get_request_id() -> str:
    return request_id_ctx_var.get()


class RequestIDFilter(logging.Filter):
    def filter(self, record: logging.LogRecord) -> bool:
        record.request_id = get_request_id()
        return True


class CustomJsonFormatter(Formatter):
    def format(self, record: logging.LogRecord) -> str:
        json_record = {
            "timestamp": self.formatTime(record, "%Y-%m-%dT%H:%M:%S.%fZ"),
            "message": record.getMessage(),
            "level": record.levelname,
            "request_id": getattr(record, "request_id", "unknown"),
        }

        optional_fields = ["req", "res", "error"]

        for field in optional_fields:
            value = record.__dict__.get(field)
            if value is not None:
                json_record[field] = value

        if record.levelno == logging.ERROR and record.exc_info:
            json_record["exception"] = self.formatException(record.exc_info)

        return json.dumps(json_record)


def _init_logger():
    logger = logging.getLogger()
    logger.propagate = False

    # Set log level from env
    log_level_str = os.getenv("LOG_LEVEL", "INFO").upper()
    log_level = getattr(logging, log_level_str, logging.INFO)
    logger.setLevel(log_level)

    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(CustomJsonFormatter())
    handler.addFilter(RequestIDFilter())
    logger.handlers = [handler]

    # Disable Uvicorn access log
    logging.getLogger("uvicorn.access").disabled = True

    return logger


logger = _init_logger()
