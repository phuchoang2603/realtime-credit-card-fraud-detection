from __future__ import annotations

import time
from datetime import UTC, datetime

import grpc
from google.protobuf.timestamp_pb2 import Timestamp
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from pydantic import ValidationError

from app.application import FraudApplication
from app.config import Settings, settings_from_env
from app.rpc import RequestContextInterceptor
from app.runtime import ApplicationState
from app.schema import DecisionInput
from fraud.v2 import fraud_pb2, fraud_pb2_grpc

SERVICE = "fraud.v2.FraudService"


class FraudService(fraud_pb2_grpc.FraudServiceServicer):
    def __init__(self, state: ApplicationState):
        self.state = state
        self.application = FraudApplication()

    async def Decide(self, request, context):
        if not self.state.ready:
            await context.abort(grpc.StatusCode.UNAVAILABLE, "Decision service unavailable")
        try:
            decision_input = DecisionInput.from_proto(request)
        except ValidationError, ValueError, OverflowError:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "Invalid or missing decision input")
        started = time.perf_counter()
        result = self.application.decide(decision_input)
        self.state.metrics.observe_latency(time.perf_counter() - started)
        self.state.metrics.record_decision(result.outcome, result.reason_codes)
        evaluated_at = Timestamp()
        evaluated_at.FromDatetime(datetime.now(UTC))
        return fraud_pb2.DecideResponse(
            outcome=(
                fraud_pb2.DECISION_OUTCOME_DECLINE
                if result.outcome == "DECLINE"
                else fraud_pb2.DECISION_OUTCOME_APPROVE
            ),
            reason_codes=result.reason_codes,
            evaluated_at=evaluated_at,
            policy_version=result.policy_version,
        )


class FraudServer:
    def __init__(self, settings: Settings | None = None, tracing=None):
        self.settings = settings or settings_from_env()
        self.settings.validate()
        self.state = ApplicationState(self.settings, tracing)
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
        await self.health.enter_graceful_shutdown()
        await self.server.stop(self.settings.graceful_shutdown_timeout if grace is None else grace)
        self.state.stop()
