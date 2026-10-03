from __future__ import annotations

import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc

from app.config import Settings, settings_from_env
from app.runtime import RuntimeState
from app.transport.grpc import FraudService
from app.transport.rpc import RequestContextInterceptor
from fraud.v2 import fraud_pb2_grpc

SERVICE = "fraud.v2.FraudService"


class FraudServer:
    def __init__(self, settings: Settings | None = None, tracing=None):
        self.settings = settings or settings_from_env()
        self.settings.validate()
        self.state = RuntimeState(self.settings, tracing)
        self.health = health.aio.HealthServicer()
        self.server = grpc.aio.server(
            interceptors=[RequestContextInterceptor(self.state)],
            maximum_concurrent_rpcs=64,
            options=[("grpc.max_receive_message_length", 64 * 1024)],
        )
        fraud_pb2_grpc.add_FraudServiceServicer_to_server(FraudService(self.state), self.server)
        health_pb2_grpc.add_HealthServicer_to_server(self.health, self.server)

    async def start(self, address: str | None = None) -> int:
        try:
            self.state.start()
            port = self.server.add_insecure_port(address or f"[::]:{self.settings.grpc_port}")
            await self.health.set("liveness", health_pb2.HealthCheckResponse.SERVING)
            for name in ("readiness", SERVICE, ""):
                await self.health.set(name, health_pb2.HealthCheckResponse.SERVING)
            await self.server.start()
            return port
        except BaseException:
            await self.stop(0)
            raise

    async def stop(self, grace: float | None = None) -> None:
        self.state.shutting_down = True
        for name in ("readiness", SERVICE, ""):
            await self.health.set(name, health_pb2.HealthCheckResponse.NOT_SERVING)
        try:
            await self.server.stop(self.settings.graceful_shutdown_timeout if grace is None else grace)
        finally:
            self.state.stop()
