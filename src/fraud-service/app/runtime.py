from __future__ import annotations

from app.config import Settings
from app.telemetry.metrics_config import Metrics, NoopMetrics
from app.telemetry.tracing_config import TracingRuntime


class RuntimeState:
    def __init__(self, settings: Settings, tracing: TracingRuntime | None = None):
        self.settings = settings
        self.started = False
        self.shutting_down = False
        self.metrics: Metrics | NoopMetrics = NoopMetrics()
        self.tracing = tracing or TracingRuntime(
            settings.tracing_enabled, settings.service_name, settings.trace_endpoint
        )

    def start(self) -> None:
        self.metrics = Metrics.create()
        if self.settings.metrics_enabled:
            self.metrics.start(self.settings.metrics_port)
        self.tracing.start()
        self.started = True
        self.shutting_down = False

    def stop(self) -> None:
        self.shutting_down = True
        self.started = False
        try:
            self.metrics.close()
        finally:
            self.tracing.stop()

    @property
    def ready(self) -> bool:
        return self.started and not self.shutting_down
