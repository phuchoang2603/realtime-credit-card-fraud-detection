from __future__ import annotations

from dataclasses import dataclass
from threading import Thread
from typing import Any

from prometheus_client import CollectorRegistry, Counter, Histogram, start_wsgi_server


@dataclass(slots=True)
class Metrics:
    registry: CollectorRegistry
    decisions: Any
    latency: Any
    server: Any = None
    thread: Thread | None = None

    @classmethod
    def create(cls) -> Metrics:
        registry = CollectorRegistry(auto_describe=True)
        return cls(
            registry=registry,
            decisions=Counter(
                "fraud_decisions_total", "Decisions by outcome and reason.", ["outcome", "reason"], registry=registry
            ),
            latency=Histogram("fraud_decision_latency_seconds", "Decision evaluation latency.", registry=registry),
        )

    def start(self, port: int, address: str = "0.0.0.0") -> None:
        self.server, self.thread = start_wsgi_server(port=port, addr=address, registry=self.registry)

    def record_decision(self, outcome: str, reasons: tuple[str, ...]) -> None:
        for reason in reasons or ("NONE",):
            self.decisions.labels(outcome=outcome, reason=reason).inc()

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
    def record_decision(self, outcome: str, reasons: tuple[str, ...]) -> None:
        del outcome, reasons

    def observe_latency(self, seconds: float) -> None:
        del seconds

    def close(self) -> None:
        return None
