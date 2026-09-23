from __future__ import annotations

import asyncio
from concurrent.futures import ThreadPoolExecutor
from contextvars import copy_context
from datetime import UTC

import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from pydantic import ValidationError

from app.application import FraudApplication
from app.config import Settings, settings_from_env
from app.errors import FraudRuleError, ModelPredictionError, ModelUnavailableError
from app.model import load_model
from app.rpc import RequestContextInterceptor, request_id
from app.runtime import ApplicationState
from app.schema import TransactionFeatures
from app.utils.logging_config import get_logger, setup_logging
from fraud.v1 import fraud_pb2, fraud_pb2_grpc

log = get_logger(__name__)
SERVICE = "fraud.v1.FraudService"


class FraudService(fraud_pb2_grpc.FraudServiceServicer):
    def __init__(self, state, executor):
        self.state = state
        self.executor = executor
        self.capacity = asyncio.Semaphore(state.settings.inference_workers)

    async def Predict(self, request, context):
        if not self.state.ready:
            await context.abort(grpc.StatusCode.UNAVAILABLE, "Model not available")
        try:
            # Scalar presence is required even when zero is a valid feature value.
            values = {field.name.upper(): value for field, value in request.ListFields()}
            if request.HasField("tx_datetime"):
                values["TX_DATETIME"] = request.tx_datetime.ToDatetime(tzinfo=UTC)
            transaction = TransactionFeatures.model_validate(values)
        except ValidationError, ValueError, OverflowError:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "Invalid or missing transaction features")
        application = FraudApplication(self.state.model, self.state.metrics, log)
        try:
            await self.capacity.acquire()
            try:
                work = asyncio.get_running_loop().run_in_executor(
                    self.executor, copy_context().run, application.predict, transaction, request_id.get()
                )
            except BaseException:
                self.capacity.release()
                raise

            def finished(future):
                self.capacity.release()
                # Retrieve failures even when the original RPC was canceled.
                if not future.cancelled():
                    future.exception()

            work.add_done_callback(finished)
            # Cancellation stops the RPC, not native inference. Keep its slot
            # occupied until the actual worker completes; never queue unbounded work.
            result = await asyncio.shield(work)
        except FraudRuleError as exc:
            await context.abort(grpc.StatusCode.PERMISSION_DENIED, str(exc))
        except ModelUnavailableError:
            await context.abort(grpc.StatusCode.UNAVAILABLE, "Model not available")
        except ModelPredictionError:
            await context.abort(grpc.StatusCode.INTERNAL, "Prediction failed")
        return fraud_pb2.PredictResponse(is_fraud=result.is_fraud, fraud_probability=result.fraud_probability)


class FraudServer:
    """Own the model, workers, telemetry and gRPC listener as one lifecycle."""

    def __init__(self, settings: Settings | None = None, model_loader=load_model, tracing=None):
        setup_logging()
        self.settings = settings or settings_from_env()
        self.settings.validate()
        self.state = ApplicationState(self.settings, model_loader, tracing)
        self.executor = ThreadPoolExecutor(max_workers=self.settings.inference_workers, thread_name_prefix="inference")
        self.health = health.aio.HealthServicer()
        self.server = grpc.aio.server(
            interceptors=[RequestContextInterceptor(self.state)],
            maximum_concurrent_rpcs=64,
            options=[("grpc.max_receive_message_length", 64 * 1024)],
        )
        fraud_pb2_grpc.add_FraudServiceServicer_to_server(FraudService(self.state, self.executor), self.server)
        health_pb2_grpc.add_HealthServicer_to_server(self.health, self.server)

    async def start(self, address: str | None = None) -> int:
        try:
            self.state.start()
            port = self.server.add_insecure_port(address or f"[::]:{self.settings.grpc_port}")
            await self.health.set("liveness", health_pb2.HealthCheckResponse.SERVING)
            status = (
                health_pb2.HealthCheckResponse.SERVING
                if self.state.ready
                else health_pb2.HealthCheckResponse.NOT_SERVING
            )
            for name in ("readiness", SERVICE, ""):
                await self.health.set(name, status)
            await self.server.start()
            return port
        except BaseException:
            await self.stop(0)
            raise

    async def stop(self, grace: float | None = None) -> None:
        self.state.shutting_down = True
        await self.health.enter_graceful_shutdown()
        await self.server.stop(self.settings.graceful_shutdown_timeout if grace is None else grace)
        self.executor.shutdown(wait=False, cancel_futures=True)
        self.state.stop()
