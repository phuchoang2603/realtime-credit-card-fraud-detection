import os
from functools import wraps

from fastapi import FastAPI
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor


def setup_tracing(app: FastAPI, service_name: str):
    """
    Sets up OpenTelemetry tracing to export traces to Grafana Alloy.
    """
    # --- Conditionally disable tracing for tests ---
    if os.environ.get("TESTING_MODE", "false").lower() == "true":
        print("TESTING_MODE is active. Skipping OpenTelemetry tracing setup.")
        return

    resource = Resource(attributes={"service.name": service_name})
    provider = TracerProvider(resource=resource)

    otlp_endpoint = os.environ.get(
        "OTEL_EXPORTER_OTLP_ENDPOINT", "http://alloy:4317/v1/traces"
    )
    # Configure the exporter to send traces to Alloy's OTLP port
    otlp_exporter = OTLPSpanExporter(endpoint=otlp_endpoint, insecure=True)
    provider.add_span_processor(BatchSpanProcessor(otlp_exporter))

    trace.set_tracer_provider(provider)
    FastAPIInstrumentor.instrument_app(app)


def get_tracer(name: str):
    """Returns a configured OpenTelemetry tracer instance."""
    return trace.get_tracer(name)


def traceable(func):
    """
    A decorator that adds an OpenTelemetry span to an async function.
    """

    @wraps(func)
    async def wrapper(*args, **kwargs):
        tracer = get_tracer(func.__module__)
        with tracer.start_as_current_span(func.__name__) as span:
            try:
                result = await func(*args, **kwargs)
                return result
            except Exception as e:
                span.record_exception(e)
                raise

    return wrapper
