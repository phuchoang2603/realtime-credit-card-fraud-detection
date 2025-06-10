from fastapi import FastAPI
from opentelemetry import trace
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor, ConsoleSpanExporter
from opentelemetry.sdk.resources import Resource
from functools import wraps


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


def traceable(func):
    """
    A decorator that adds an OpenTelemetry span to a function.
    The span is automatically named after the function.
    """

    @wraps(func)
    async def wrapper(*args, **kwargs):
        tracer = get_tracer(func.__module__)
        with tracer.start_as_current_span(func.__name__) as span:
            try:
                # Execute the original async function
                result = await func(*args, **kwargs)
                return result
            except Exception as e:
                # Record the exception in the span and re-raise it
                span.record_exception(e)
                raise

    return wrapper
