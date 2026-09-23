from threading import Event

import grpc
import numpy as np
import pytest
from grpc_health.v1 import health_pb2
from opentelemetry import trace
from opentelemetry.sdk.trace.export.in_memory_span_exporter import InMemorySpanExporter

from app.utils.tracing_config import TracingRuntime

pytestmark = pytest.mark.api


def health(client, name):
    return client.health.Check(health_pb2.HealthCheckRequest(service=name), timeout=1).status


def test_prediction_wire_and_model_aware_health(rpc_server, prediction_request):
    with rpc_server() as client:
        assert health(client, "readiness") == health_pb2.HealthCheckResponse.SERVING
        result, call = client.stub.Predict.with_call(
            prediction_request, timeout=2, metadata=(("x-request-id", "caller-42"),)
        )
        assert result.is_fraud and result.fraud_probability == 0.8
        assert dict(call.initial_metadata())["x-request-id"] == "caller-42"

    def missing(_):
        raise FileNotFoundError("missing model")

    with rpc_server(loader=missing) as client:
        assert health(client, "liveness") == health_pb2.HealthCheckResponse.SERVING
        assert health(client, "readiness") == health_pb2.HealthCheckResponse.NOT_SERVING
        with pytest.raises(grpc.RpcError) as error:
            client.stub.Predict(prediction_request, timeout=2)
        assert error.value.code() == grpc.StatusCode.UNAVAILABLE
        assert dict(error.value.initial_metadata())["x-request-id"]


@pytest.mark.parametrize("invalid", ["missing", "nonfinite", "invalid-flag", "invalid-timestamp"])
def test_wire_validation_rejects_invalid_features(rpc_server, prediction_request, invalid):
    if invalid == "missing":
        prediction_request.ClearField("tx_during_weekend")
    elif invalid == "nonfinite":
        prediction_request.tx_amount = float("nan")
    elif invalid == "invalid-flag":
        prediction_request.tx_during_weekend = 2
    else:
        prediction_request.tx_datetime.seconds = 10**15
    with rpc_server() as client, pytest.raises(grpc.RpcError) as error:
        client.stub.Predict(prediction_request, timeout=2)
    assert error.value.code() == grpc.StatusCode.INVALID_ARGUMENT


def test_rule_rejection_maps_to_permission_denied(rpc_server, prediction_request):
    prediction_request.customer_id = 323
    with rpc_server() as client, pytest.raises(grpc.RpcError) as error:
        client.stub.Predict(prediction_request, timeout=2)
    assert error.value.code() == grpc.StatusCode.PERMISSION_DENIED


def test_model_failure_is_sanitized_and_traced(rpc_server, prediction_request, caplog):
    class Broken:
        def predict_proba(self, _):
            raise RuntimeError("secret backend credentials")

    exporter = InMemorySpanExporter()
    runtime = TracingRuntime(True, "rpc-test", lambda: exporter)
    global_provider = trace.get_tracer_provider()
    with caplog.at_level("INFO"), rpc_server(loader=lambda _: Broken(), tracing=runtime) as client:
        with pytest.raises(grpc.RpcError) as error:
            client.stub.Predict(
                prediction_request,
                timeout=2,
                metadata=(
                    ("x-request-id", "failure-42"),
                    ("traceparent", "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01"),
                ),
            )
        assert error.value.code() == grpc.StatusCode.INTERNAL
        assert error.value.details() == "Prediction failed"
    span = exporter.get_finished_spans()[0]
    assert span.context.trace_id == int("0123456789abcdef0123456789abcdef", 16)
    assert span.parent.span_id == int("0123456789abcdef", 16)
    assert span.attributes["rpc.grpc.status_code"] == 13
    assert span.events == ()
    assert "failure-42" in caplog.text and "secret backend" not in caplog.text
    assert trace.get_tracer_provider() is global_provider


@pytest.mark.parametrize("expire", [False, True], ids=["drain-completes", "drain-expires"])
def test_slow_inference_keeps_health_responsive_and_shutdown_bounded(rpc_server, prediction_request, expire):
    entered, release = Event(), Event()

    class Slow:
        def predict_proba(self, _):
            entered.set()
            assert release.wait(5)
            return np.array([[0.8, 0.2]])

    with rpc_server(loader=lambda _: Slow()) as client:
        call = client.stub.Predict.future(prediction_request, timeout=4)
        try:
            assert entered.wait(2)
            assert health(client, "liveness") == health_pb2.HealthCheckResponse.SERVING
            stopped = client.stop(0.05 if expire else 2)
            if not expire:
                release.set()
                assert call.result(2).fraud_probability == 0.2
            stopped.result(2)
            assert not client.server.state.ready
            if expire:
                with pytest.raises(grpc.RpcError):
                    call.result(1)
        finally:
            release.set()


def test_client_deadline_does_not_block_subsequent_health(rpc_server, prediction_request):
    from app.config import Settings

    entered, release, second = Event(), Event(), Event()

    class Slow:
        def predict_proba(self, _):
            if entered.is_set():
                second.set()
            entered.set()
            release.wait(3)
            return np.array([[0.8, 0.2]])

    with rpc_server(
        loader=lambda _: Slow(), settings=Settings(metrics_enabled=False, tracing_enabled=False, inference_workers=1)
    ) as client:
        try:
            call = client.stub.Predict.future(prediction_request, timeout=0.1)
            assert entered.wait(1)
            with pytest.raises(grpc.RpcError) as error:
                call.result(1)
            assert error.value.code() == grpc.StatusCode.DEADLINE_EXCEEDED
            # A canceled native call still owns the only worker slot.
            with pytest.raises(grpc.RpcError) as queued:
                client.stub.Predict(prediction_request, timeout=0.1)
            assert queued.value.code() == grpc.StatusCode.DEADLINE_EXCEEDED
            assert not second.is_set()
            assert health(client, "liveness") == health_pb2.HealthCheckResponse.SERVING
        finally:
            release.set()


def test_metrics_listener_is_owned_by_server_lifecycle(rpc_server):
    import socket

    from app.config import Settings

    with socket.socket() as probe:
        probe.bind(("127.0.0.1", 0))
        port = probe.getsockname()[1]
    settings = Settings(metrics_port=port, tracing_enabled=False)
    with rpc_server(settings=settings) as client:
        with socket.create_connection(("127.0.0.1", port), timeout=1):
            pass
        assert health(client, "readiness") == health_pb2.HealthCheckResponse.SERVING
    with socket.socket() as probe:
        assert probe.connect_ex(("127.0.0.1", port)) != 0


def test_failed_startup_releases_metrics_listener(rpc_server):
    import socket

    from app.config import Settings

    with socket.socket() as probe:
        probe.bind(("127.0.0.1", 0))
        port = probe.getsockname()[1]

    def broken_exporter():
        raise ValueError("invalid exporter")

    runtime = TracingRuntime(True, "broken", broken_exporter)
    with pytest.raises(ValueError, match="invalid exporter"):
        with rpc_server(settings=Settings(metrics_port=port), tracing=runtime):
            pass
    with socket.socket() as probe:
        assert probe.connect_ex(("127.0.0.1", port)) != 0
    assert runtime.provider is None


def test_unexpected_failure_is_sanitized(rpc_server, prediction_request, monkeypatch):
    def broken(*_):
        raise RuntimeError("private adapter address")

    monkeypatch.setattr("app.main.FraudApplication.predict", broken)
    with rpc_server() as client, pytest.raises(grpc.RpcError) as error:
        client.stub.Predict(prediction_request, timeout=2)
    assert error.value.code() == grpc.StatusCode.INTERNAL
    assert error.value.details() == "Internal server error"
