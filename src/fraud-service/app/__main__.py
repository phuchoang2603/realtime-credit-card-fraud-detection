import asyncio
import os
import signal
import sys
from threading import Timer

from app.config import Settings, settings_from_env
from app.main import FraudServer
from app.telemetry.logging_config import get_logger, setup_logging

log = get_logger(__name__)


async def serve(settings: Settings) -> None:
    server = FraudServer(settings)
    stopping = asyncio.Event()
    watchdog = None

    def request_shutdown():
        nonlocal watchdog
        if stopping.is_set():
            return
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
        if watchdog:
            watchdog.cancel()


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
