"""Telemetry contract checks executed by CI without an external backend."""

import json
import logging
from pathlib import Path
from unittest.mock import patch

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from prometheus_client import REGISTRY

from app.utils import logging_config, metrics_config, tracing_config
from app.utils.telemetry_config import service_name


@pytest.mark.parametrize("identity", [None, "custom-fraud-service"])
def test_service_identity_across_signals(monkeypatch, identity):
    monkeypatch.delenv("OTEL_SERVICE_NAME", raising=False)
    if identity:
        monkeypatch.setenv("OTEL_SERVICE_NAME", identity)
    expected = identity or "fraud-service"
    assert service_name() == expected
    assert logging_config.add_trace_context(None, None, {})["service"] == expected
    with (
        patch.object(metrics_config, "start_http_server") as server,
        patch.object(metrics_config, "PrometheusMetricReader"),
        patch.object(metrics_config, "MeterProvider") as provider,
        patch.object(metrics_config, "set_meter_provider"),
        patch.object(metrics_config, "get_meter_provider") as get_provider,
    ):
        metrics_config.setup_metrics("1.0.0")
        assert provider.call_args.kwargs["resource"].attributes["service.name"] == expected
        get_provider.return_value.get_meter.assert_called_once_with(expected, "1.0.0")
        server.assert_called_once_with(port=8010, addr="0.0.0.0")

    monkeypatch.setenv("TESTING_MODE", "false")
    monkeypatch.setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "traces.monitoring:4317")
    with (
        patch.object(tracing_config, "OTLPSpanExporter") as exporter,
        patch.object(tracing_config, "BatchSpanProcessor") as processor,
        patch.object(tracing_config, "TracerProvider") as provider,
        patch.object(tracing_config.trace, "set_tracer_provider") as set_provider,
        patch.object(tracing_config.FastAPIInstrumentor, "instrument_app") as instrument,
    ):
        app = FastAPI()
        tracing_config.setup_tracing(app)
        exporter.assert_called_once_with(endpoint="traces.monitoring:4317", insecure=True)
        assert provider.call_args.kwargs["resource"].attributes["service.name"] == expected
        processor.assert_called_once_with(exporter.return_value)
        provider.return_value.add_span_processor.assert_called_once_with(processor.return_value)
        set_provider.assert_called_once_with(provider.return_value)
        instrument.assert_called_once_with(app)


def test_disabled_export_still_serves_requests(monkeypatch):
    monkeypatch.setenv("TESTING_MODE", "true")
    app = FastAPI()

    @app.get("/health")
    def health():
        return {"status": "ok"}

    with patch.object(tracing_config, "OTLPSpanExporter") as exporter:
        tracing_config.setup_tracing(app)
        with TestClient(app) as client:
            assert client.get("/health").json() == {"status": "ok"}
        exporter.assert_not_called()


def test_request_log_has_matching_span_ids(monkeypatch, caplog):
    monkeypatch.setenv("OTEL_SERVICE_NAME", "custom-fraud-service")
    logging_config.setup_logging()
    provider = TracerProvider()
    try:
        with caplog.at_level(logging.INFO), provider.get_tracer("test").start_as_current_span("request") as span:
            logging_config.get_logger("test.request").info("request")
            record = json.loads(caplog.records[-1].message)
            context = span.get_span_context()
            assert record["service"] == "custom-fraud-service"
            assert record["trace_id"] == f"{context.trace_id:032x}"
            assert record["span_id"] == f"{context.span_id:016x}"
    finally:
        provider.shutdown()


def test_startup_log_has_identity_without_fabricated_span(monkeypatch, caplog):
    monkeypatch.delenv("OTEL_SERVICE_NAME", raising=False)
    logging_config.setup_logging()
    with caplog.at_level(logging.INFO), trace.use_span(trace.INVALID_SPAN):
        logging_config.get_logger("test.startup").info("startup", trace_id="stale", span_id="stale")
    record = json.loads(caplog.records[-1].message)
    assert record["service"] == "fraud-service"
    assert "trace_id" not in record
    assert "span_id" not in record


def test_dashboard_metric_names_match_exporter():
    metrics_config.predictions_counter.add(1, {"is_fraud": "False"})
    metrics_config.prediction_latency.record(0.1)
    metrics_config.fraud_score_histogram.record(0.2)
    samples = {sample.name for family in REGISTRY.collect() for sample in family.samples}
    expected = {"predictions_total", "prediction_latency_seconds_bucket", "fraud_prediction_score_bucket"}
    assert expected <= samples
    dashboard_path = Path(__file__).resolve().parents[3] / "infra/charts/fraud-service/dashboards/fraud-service.json"
    dashboard = json.loads(dashboard_path.read_text())
    queries = [target["expr"] for panel in dashboard["panels"][:3] for target in panel["targets"]]
    for name in expected:
        assert any(name in expression for expression in queries)
    counter = next(family for family in REGISTRY.collect() if family.name == "predictions")
    assert any(sample.labels.get("is_fraud") == "False" for sample in counter.samples)
