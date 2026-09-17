import os


def service_name() -> str:
    """Use the same configured identity for logs, metrics, and traces."""
    return os.environ.get("OTEL_SERVICE_NAME", "fraud-service")
