from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response
from utils.logging import logger, set_request_id
import uuid


class LogMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request, call_next):
        response = await call_next(request)

        if response.status_code != 422:
            logger.info(
                "Incoming request",
                extra={
                    "req": {
                        "method": request.method,
                        "url": str(request.url),
                        "client": request.client.host,
                    },
                    "res": {"status_code": response.status_code},
                },
            )

        return response


class RequestIDMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        # Try to use incoming header, otherwise generate one
        request_id = request.headers.get("X-Request-ID", str(uuid.uuid4()))
        request.state.request_id = request_id

        set_request_id(request_id)

        # Propagate it back to client response header too
        response: Response = await call_next(request)
        response.headers["X-Request-ID"] = request_id
        return response
