from __future__ import annotations

from dataclasses import dataclass
from threading import Thread
from typing import Any

from prometheus_client import CollectorRegistry, Counter, Histogram, start_wsgi_server


@dataclass(slots=True)
class Metrics:
    """Owns application metrics and the optional Prometheus listener."""

    registry: CollectorRegistry
    predictions: Any
    latency: Any
    scores: Any
    server: Any = None
    thread: Thread | None = None

    @classmethod
    def create(cls) -> Metrics:
        registry = CollectorRegistry(auto_describe=True)
        return cls(
            registry=registry,
            predictions=Counter(
                "predictions_total", "Total number of predictions made.", ["is_fraud"], registry=registry
            ),
            latency=Histogram(
                "prediction_latency_seconds", "Latency of prediction endpoint in seconds.", registry=registry
            ),
            scores=Histogram("fraud_prediction_score", "Distribution of fraud prediction scores.", registry=registry),
        )

    def start(self, port: int, address: str = "0.0.0.0") -> None:
        self.server, self.thread = start_wsgi_server(port=port, addr=address, registry=self.registry)

    def record_prediction(self, is_fraud: bool, probability: float) -> None:
        self.predictions.labels(is_fraud=str(is_fraud)).inc()
        self.scores.observe(probability)

    def observe_latency(self, seconds: float) -> None:
        self.latency.observe(seconds)

    def close(self) -> None:
        if self.server is not None:
            self.server.shutdown()
            self.server.server_close()
            if self.thread is not None:
                self.thread.join(timeout=1)
            self.server = None
            self.thread = None


class NoopMetrics:
    def record_prediction(self, is_fraud: bool, probability: float) -> None:
        del is_fraud, probability

    def observe_latency(self, seconds: float) -> None:
        del seconds

    def close(self) -> None:
        return None
