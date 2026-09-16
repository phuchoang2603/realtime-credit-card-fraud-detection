from opentelemetry.exporter.prometheus import PrometheusMetricReader
from opentelemetry.metrics import get_meter_provider, set_meter_provider
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.resources import Resource
from prometheus_client import start_http_server


def setup_metrics(service_name: str, app_version: str):
    """
    Sets up OpenTelemetry metrics and starts the Prometheus exporter.
    Returns a meter that can be used to create metric instruments.
    """
    # Start Prometheus client to expose metrics on port 8010
    start_http_server(port=8010, addr="0.0.0.0")

    resource = Resource(attributes={"service.name": service_name})

    # Set up the MeterProvider
    reader = PrometheusMetricReader()
    meter_provider = MeterProvider(resource=resource, metric_readers=[reader])
    set_meter_provider(meter_provider)

    return get_meter_provider().get_meter(service_name, app_version)


# Create a global meter instance
meter = setup_metrics("fraud-detection-api", "1.0.0")

# Define specific metric instruments to be used across the application
predictions_counter = meter.create_counter(
    "predictions_total",
    description="Total number of predictions made.",
)
prediction_latency = meter.create_histogram(
    "prediction_latency_seconds",
    description="Latency of the prediction endpoint in seconds.",
    unit="s",
)
fraud_score_histogram = meter.create_histogram(
    "fraud_prediction_score", description="Distribution of fraud prediction scores."
)
