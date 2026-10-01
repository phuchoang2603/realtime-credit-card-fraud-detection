from __future__ import annotations

import logging
from collections.abc import Callable
from threading import Thread

from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor, SpanExporter


class TracingRuntime:
    """An instance-owned provider; never replaces the global OpenTelemetry provider."""

    def __init__(
        self,
        enabled: bool,
        service_name: str,
        endpoint: str | None = None,
        exporter_factory: Callable[[], SpanExporter] | None = None,
    ):
        self.enabled = enabled
        self.service_name = service_name
        self.endpoint = endpoint
        self.exporter_factory = exporter_factory
        self.provider: TracerProvider | None = None

    def start(self) -> None:
        if not self.enabled:
            return
        if self.exporter_factory is None and not (self.endpoint and self.endpoint.strip()):
            raise ValueError("OTEL_EXPORTER_OTLP_ENDPOINT is required when tracing is enabled")
        provider = TracerProvider(
            resource=Resource.create({"service.name": self.service_name}), shutdown_on_exit=False
        )
        try:
            exporter = (
                self.exporter_factory()
                if self.exporter_factory
                else OTLPSpanExporter(endpoint=self.endpoint, timeout=2)
            )
            provider.add_span_processor(BatchSpanProcessor(exporter))
        except Exception:
            provider.shutdown()
            raise
        self.provider = provider

    def stop(self, timeout: float = 3) -> None:
        provider, self.provider = self.provider, None
        if provider is None:
            return
        # Exporters are external code: an unresponsive backend cannot hold process exit.
        worker = Thread(target=provider.shutdown, daemon=True, name="trace-shutdown")
        worker.start()
        worker.join(timeout=timeout)
        if worker.is_alive():
            logging.getLogger(__name__).warning("Trace shutdown exceeded cleanup budget")
