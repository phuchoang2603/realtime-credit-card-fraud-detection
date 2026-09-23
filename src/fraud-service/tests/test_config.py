import pytest

from app.config import settings_from_env


@pytest.mark.parametrize(
    "name,value",
    [
        ("GRPC_PORT", "0"),
        ("METRICS_PORT", "65536"),
        ("INFERENCE_WORKERS", "0"),
        ("GRACEFUL_SHUTDOWN_TIMEOUT", "0"),
        ("GRACEFUL_SHUTDOWN_TIMEOUT", "nan"),
        ("MODEL_PATH", ""),
        ("OTEL_SERVICE_NAME", " "),
        ("METRICS_ENABLED", "sometimes"),
    ],
)
def test_invalid_environment_fails_before_serving(monkeypatch, name, value):
    monkeypatch.setenv("TESTING_MODE", "false")
    monkeypatch.setenv(name, value)
    with pytest.raises(ValueError, match=name):
        settings_from_env()


def test_service_configuration_uses_environment(monkeypatch, tmp_path):
    model = tmp_path / "model.pkl"
    for key, value in {
        "TESTING_MODE": "false",
        "MODEL_PATH": str(model),
        "GRPC_PORT": "9000",
        "INFERENCE_WORKERS": "2",
        "METRICS_ENABLED": "FALSE",
        "TRACING_ENABLED": "false",
        "GRACEFUL_SHUTDOWN_TIMEOUT": "12",
    }.items():
        monkeypatch.setenv(key, value)
    settings = settings_from_env()
    assert settings.model_path == model
    assert settings.grpc_port == 9000
    assert settings.inference_workers == 2
    assert settings.graceful_shutdown_timeout == 12
    assert not settings.metrics_enabled and not settings.tracing_enabled
