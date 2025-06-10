from fastapi import FastAPI
from opentelemetry import trace
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor, ConsoleSpanExporter
from opentelemetry.sdk.resources import Resource


def setup_tracing(app: FastAPI, service_name: str):
    """
    Sets up OpenTelemetry tracing and instruments the FastAPI application.
    """
    resource = Resource(attributes={"service.name": service_name})

    # Set up the TracerProvider
    provider = TracerProvider(resource=resource)
    provider.add_span_processor(BatchSpanProcessor(ConsoleSpanExporter()))
    trace.set_tracer_provider(provider)

    # Instrument the FastAPI app automatically
    FastAPIInstrumentor.instrument_app(app)


def get_tracer(name: str):
    """Returns a configured OpenTelemetry tracer instance."""
    return trace.get_tracer(name)
