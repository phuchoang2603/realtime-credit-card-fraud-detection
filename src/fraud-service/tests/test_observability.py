from opentelemetry.sdk.trace import TracerProvider

from app.utils.logging_config import add_trace_context


def test_logs_include_real_trace_and_span_ids_when_a_span_exists(monkeypatch):
    monkeypatch.setenv("OTEL_SERVICE_NAME", "fraud-test")
    provider = TracerProvider()
    tracer = provider.get_tracer("verification")

    with tracer.start_as_current_span("request") as span:
        event = add_trace_context(None, "info", {})

    assert event["service"] == "fraud-test"
    assert event["trace_id"] == format(span.get_span_context().trace_id, "032x")
    assert event["span_id"] == format(span.get_span_context().span_id, "016x")
    provider.shutdown()


def test_metrics_exposition_retains_bounded_instrument_contract():
    from prometheus_client import generate_latest

    from app.utils.metrics_config import Metrics

    metrics = Metrics.create()
    metrics.record_prediction(True, 0.8)
    metrics.record_prediction(False, 0.2)
    metrics.observe_latency(0.25)
    samples = {}
    for family in metrics.registry.collect():
        for sample in family.samples:
            samples[(sample.name, tuple(sorted(sample.labels.items())))] = sample.value
    assert samples[("predictions_total", (("is_fraud", "True"),))] == 1
    assert samples[("predictions_total", (("is_fraud", "False"),))] == 1
    assert samples[("prediction_latency_seconds_count", ())] == 1
    assert samples[("prediction_latency_seconds_sum", ())] == 0.25
    assert samples[("fraud_prediction_score_count", ())] == 2
    assert samples[("fraud_prediction_score_sum", ())] == 1
    exposition = generate_latest(metrics.registry).decode()
    assert "request_id" not in exposition
    assert "customer_id" not in exposition


def test_startup_log_cannot_reuse_stale_trace_identifiers():
    event = add_trace_context(None, "info", {"trace_id": "stale", "span_id": "stale", "service": "configured"})
    assert event == {"service": "configured"}
