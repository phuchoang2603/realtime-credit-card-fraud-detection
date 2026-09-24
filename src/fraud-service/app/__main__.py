import asyncio
import os
import signal
import sys
from threading import Timer

from app.config import Settings, settings_from_env
from app.main import FraudServer
from app.model import load_model
from app.utils.logging_config import get_logger, setup_logging

log = get_logger(__name__)


async def serve(settings: Settings, model_loader=load_model) -> None:
    server = FraudServer(settings, model_loader=model_loader)
    stopping = asyncio.Event()
    watchdog = None

    def request_shutdown():
        nonlocal watchdog
        if stopping.is_set():
            return
        # Python cannot interrupt a running native model call. Bound process exit
        # after RPC drain and telemetry cleanup even if an executor thread hangs.
        watchdog = Timer(settings.graceful_shutdown_timeout + 5, lambda: os._exit(0))
        watchdog.daemon = True
        watchdog.start()
        stopping.set()

    loop = asyncio.get_running_loop()
    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, request_shutdown)
    await server.start()
    log.info("gRPC server started", port=settings.grpc_port)
    try:
        await stopping.wait()
    finally:
        await server.stop()
        log.info("gRPC server stopped")
        # Leave the daemon watchdog armed until interpreter exit: the executor's
        # exit hook otherwise waits forever on a stuck native model thread.


def main() -> None:
    try:
        settings = settings_from_env()
    except ValueError as exc:
        setup_logging()
        log.error("Invalid fraud configuration", error=str(exc))
        sys.exit(1)
    setup_logging(settings.service_name)
    asyncio.run(serve(settings))


if __name__ == "__main__":
    main()
