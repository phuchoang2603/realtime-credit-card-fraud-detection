from __future__ import annotations

import math
import os
from dataclasses import dataclass
from pathlib import Path

DEFAULT_MODEL_PATH = Path(__file__).resolve().parents[1] / "models" / "model.pkl"


@dataclass(frozen=True, slots=True)
class Settings:
    grpc_port: int = 8000
    inference_workers: int = 4
    model_path: Path = DEFAULT_MODEL_PATH
    metrics_enabled: bool = True
    metrics_port: int = 8010
    graceful_shutdown_timeout: float = 30.0
    tracing_enabled: bool = True
    service_name: str = "fraud-service"

    def validate(self) -> None:
        if not 1 <= self.grpc_port <= 65535:
            raise ValueError("GRPC_PORT must be between 1 and 65535")
        if self.inference_workers < 1:
            raise ValueError("INFERENCE_WORKERS must be positive")
        if not str(self.model_path).strip():
            raise ValueError("MODEL_PATH must not be empty")
        if not 1 <= self.metrics_port <= 65535:
            raise ValueError("METRICS_PORT must be between 1 and 65535")
        if not math.isfinite(self.graceful_shutdown_timeout) or self.graceful_shutdown_timeout < 1:
            raise ValueError("GRACEFUL_SHUTDOWN_TIMEOUT must be finite and at least one second")
        if not self.service_name.strip():
            raise ValueError("OTEL_SERVICE_NAME must not be empty")


def settings_from_env() -> Settings:
    settings = Settings(
        grpc_port=_env_int("GRPC_PORT", 8000),
        inference_workers=_env_int("INFERENCE_WORKERS", 4),
        model_path=Path(os.environ.get("MODEL_PATH") or DEFAULT_MODEL_PATH),
        metrics_enabled=_env_bool("METRICS_ENABLED", True),
        metrics_port=_env_int("METRICS_PORT", 8010),
        graceful_shutdown_timeout=_env_float("GRACEFUL_SHUTDOWN_TIMEOUT", 30.0),
        tracing_enabled=_env_bool("TRACING_ENABLED", True),
        service_name=os.environ.get("OTEL_SERVICE_NAME", "fraud-service"),
    )
    settings.validate()
    return settings


def _env_int(name: str, default: int) -> int:
    value = os.environ.get(name)
    if value is None:
        return default
    try:
        return int(value)
    except ValueError:
        raise ValueError(f"{name} must be an integer") from None


def _env_float(name: str, default: float) -> float:
    value = os.environ.get(name)
    if value is None:
        return default
    try:
        return float(value)
    except ValueError:
        raise ValueError(f"{name} must be a number") from None


def _env_bool(name: str, default: bool) -> bool:
    value = os.environ.get(name)
    if value is None:
        return default
    normalized = value.lower()
    if normalized not in {"true", "false"}:
        raise ValueError(f"{name} must be true or false")
    return normalized == "true"
