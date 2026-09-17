import logging

import structlog
from opentelemetry import trace


def add_trace_context(logger, method_name, event_dict):
    context = trace.get_current_span().get_span_context()
    if context.is_valid:
        event_dict["trace_id"] = format(context.trace_id, "032x")
        event_dict["span_id"] = format(context.span_id, "016x")
    return event_dict


def setup_logging():
    """
    Configures structlog for JSON-formatted, structured logging.
    This setup is ideal for production environments where logs are parsed by machines.
    """
    logging.basicConfig(level=logging.INFO, format="%(message)s")

    structlog.configure(
        processors=[
            add_trace_context,
            structlog.stdlib.add_log_level,
            structlog.stdlib.add_logger_name,
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.JSONRenderer(),
        ],
        logger_factory=structlog.stdlib.LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
    )


def get_logger(name: str):
    """Returns a configured structlog logger instance."""
    return structlog.get_logger(name)
