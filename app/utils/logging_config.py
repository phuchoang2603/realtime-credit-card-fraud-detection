import logging
import structlog


def setup_logging():
    """
    Configures structlog for JSON-formatted, structured logging.
    This setup is ideal for production environments where logs are parsed by machines.
    """
    logging.basicConfig(level=logging.INFO, format="%(message)s")

    structlog.configure(
        processors=[
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
