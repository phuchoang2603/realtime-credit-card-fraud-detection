from __future__ import annotations

import asyncio
from contextlib import nullcontext
from contextvars import ContextVar
from uuid import uuid4

import grpc
from opentelemetry import propagate, trace
from opentelemetry.trace import SpanKind, StatusCode

from app.utils.logging_config import get_logger

log = get_logger(__name__)
request_id: ContextVar[str] = ContextVar("request_id", default="")


def request_id_from(metadata: dict) -> str:
    value = metadata.get("x-request-id", "")
    sanitized = "".join(c for c in value if 32 <= ord(c) < 127).strip()[:128]
    return sanitized or uuid4().hex


class RequestContextInterceptor(grpc.aio.ServerInterceptor):
    def __init__(self, runtime):
        self.runtime = runtime

    async def intercept_service(self, continuation, details):
        handler = await continuation(details)
        if handler is None or handler.unary_unary is None:
            return handler

        async def invoke(request, context):
            metadata = dict(context.invocation_metadata())
            identifier = request_id_from(metadata)
            token = request_id.set(identifier)
            await context.send_initial_metadata((("x-request-id", identifier),))
            provider = self.runtime.tracing.provider
            span_context = (
                provider.get_tracer(__name__).start_as_current_span(
                    details.method,
                    context=propagate.extract(metadata),
                    kind=SpanKind.SERVER,
                    record_exception=False,
                    set_status_on_exception=False,
                )
                if provider
                else nullcontext(trace.INVALID_SPAN)
            )
            code = grpc.StatusCode.OK
            with span_context as span:
                try:
                    return await handler.unary_unary(request, context)
                except grpc.aio.AbortError:
                    code = context.code()
                    raise
                except asyncio.CancelledError:
                    code = grpc.StatusCode.CANCELLED
                    raise
                except Exception as exc:
                    code = grpc.StatusCode.INTERNAL
                    log.error("Request failed", request_id=identifier, error_type=type(exc).__name__)
                    await context.abort(code, "Internal server error")
                finally:
                    code = context.code() or code
                    span.set_attributes(
                        {"rpc.system": "grpc", "rpc.method": details.method, "rpc.grpc.status_code": code.value[0]}
                    )
                    if code != grpc.StatusCode.OK:
                        span.set_status(StatusCode.ERROR)
                    log.info(
                        "RPC request",
                        request_id=identifier,
                        method=details.method,
                        status=code.name,
                    )
                    request_id.reset(token)

        return grpc.unary_unary_rpc_method_handler(
            invoke,
            request_deserializer=handler.request_deserializer,
            response_serializer=handler.response_serializer,
        )
