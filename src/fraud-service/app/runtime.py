from __future__ import annotations

from collections.abc import Callable
from typing import Any

from app.config import Settings
from app.utils.logging_config import get_logger
from app.utils.metrics_config import Metrics, NoopMetrics
from app.utils.tracing_config import TracingRuntime

log = get_logger(__name__)


class ApplicationState:
    def __init__(self, settings: Settings, model_loader: Callable[[Any], Any], tracing: TracingRuntime | None = None):
        self.settings = settings
        self.model_loader = model_loader
        self.model: Any = None
        self.started = False
        self.shutting_down = False
        self.metrics: Metrics | NoopMetrics = NoopMetrics()
        self.tracing = tracing or TracingRuntime(settings.tracing_enabled, settings.service_name)

    def start(self) -> None:
        self.metrics = Metrics.create()
        if self.settings.metrics_enabled:
            self.metrics.start(self.settings.metrics_port)
        try:
            self.model = self.model_loader(self.settings.model_path)
            log.info("Model loaded successfully", path=str(self.settings.model_path))
        except Exception as exc:
            self.model = None
            log.error("Model unavailable", error_type=type(exc).__name__)
        self.tracing.start()
        self.started = True
        self.shutting_down = False

    def stop(self) -> None:
        self.shutting_down = True
        self.started = False
        self.model = None
        try:
            self.metrics.close()
        finally:
            self.tracing.stop()

    @property
    def ready(self) -> bool:
        return self.started and not self.shutting_down and self.model is not None
