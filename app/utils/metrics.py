from opentelemetry import metrics
from opentelemetry.exporter.prometheus import PrometheusMetricReader
from opentelemetry.metrics import set_meter_provider
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from prometheus_client import start_http_server

# Start Prometheus endpoint at http://localhost:8001/metrics
start_http_server(port=8001)

# Set up OpenTelemetry metrics
resource = Resource(attributes={SERVICE_NAME: "fastapi-ml-service"})
reader = PrometheusMetricReader()
provider = MeterProvider(resource=resource, metric_readers=[reader])
set_meter_provider(provider)

# Get a meter
meter = metrics.get_meter("fastapi-ml-service")

# Create OTEL metrics
request_counter = meter.create_counter(
    name="app_requests_total",
    unit="1",
    description="Total number of requests",
)

predict_duration = meter.create_histogram(
    name="predict_duration_seconds",
    unit="s",
    description="Duration of /predict endpoint",
)
