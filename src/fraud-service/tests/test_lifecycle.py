import pytest

from app.config import Settings, settings_from_env


@pytest.mark.parametrize(
    ("settings", "message"),
    [
        (Settings(model_path=None), "MODEL_PATH"),
        (Settings(graceful_shutdown_timeout=0), "GRACEFUL_SHUTDOWN_TIMEOUT"),
        (Settings(service_name="  "), "OTEL_SERVICE_NAME"),
    ],
    ids=["missing-model-path", "non-positive-shutdown", "blank-service-name"],
)
def test_settings_validate_all_required_runtime_fields(settings, message):
    with pytest.raises(ValueError, match=message):
        settings.validate()


@pytest.mark.parametrize(
    "name,value", [("MODEL_PATH", ""), ("METRICS_ENABLED", "sometimes"), ("GRACEFUL_SHUTDOWN_TIMEOUT", "nan")]
)
def test_invalid_environment_is_rejected(monkeypatch, name, value):
    monkeypatch.setenv("TESTING_MODE", "false")
    monkeypatch.setenv(name, value)
    with pytest.raises(ValueError, match=name):
        settings_from_env()
