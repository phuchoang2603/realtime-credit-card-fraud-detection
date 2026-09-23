from pathlib import Path

import pytest

from app.config import Settings, settings_from_env


@pytest.mark.parametrize("port", [1, 65535])
def test_metrics_port_inclusive_boundaries(port):
    Settings(metrics_port=port).validate()


@pytest.mark.parametrize("port", [-1, 0, 65536])
def test_invalid_metrics_port(port):
    with pytest.raises(ValueError, match="METRICS_PORT"):
        Settings(metrics_port=port).validate()


@pytest.mark.parametrize("value", [float("nan"), float("inf"), -1, 0, 0.5])
def test_shutdown_budget_must_be_finite_and_positive(value):
    with pytest.raises(ValueError, match="GRACEFUL_SHUTDOWN_TIMEOUT"):
        Settings(graceful_shutdown_timeout=value).validate()


def test_environment_defaults_and_overrides(monkeypatch, tmp_path):
    for key in (
        "MODEL_PATH",
        "METRICS_ENABLED",
        "METRICS_PORT",
        "GRACEFUL_SHUTDOWN_TIMEOUT",
        "TRACING_ENABLED",
        "OTEL_SERVICE_NAME",
    ):
        monkeypatch.delenv(key, raising=False)
    monkeypatch.setenv("TESTING_MODE", "false")
    defaults = settings_from_env()
    assert defaults == Settings()
    for key, value in {
        "MODEL_PATH": str(tmp_path / "model.pkl"),
        "METRICS_ENABLED": "false",
        "TRACING_ENABLED": "false",
        "METRICS_PORT": "9010",
        "GRACEFUL_SHUTDOWN_TIMEOUT": "15",
        "OTEL_SERVICE_NAME": "test-fraud",
    }.items():
        monkeypatch.setenv(key, value)
    configured = settings_from_env()
    assert configured == Settings(
        model_path=Path(tmp_path / "model.pkl"),
        metrics_enabled=False,
        metrics_port=9010,
        graceful_shutdown_timeout=15,
        tracing_enabled=False,
        service_name="test-fraud",
    )
    monkeypatch.setenv("TESTING_MODE", "true")
    monkeypatch.setenv("METRICS_ENABLED", "true")
    monkeypatch.setenv("TRACING_ENABLED", "true")
    assert not settings_from_env().metrics_enabled
    assert not settings_from_env().tracing_enabled
