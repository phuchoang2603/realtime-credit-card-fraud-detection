import pytest


@pytest.fixture
def sample_legitimate_payload():
    """A pytest fixture to provide sample legitimate transaction data."""
    return {
        "TRANSACTION_ID": 1,
        "TX_DATETIME": "2025-06-12T10:00:00Z",
        "CUSTOMER_ID": 1001,
        "TERMINAL_ID": 2001,
        "TX_TIME_SECONDS": 1749615600,
        "TX_TIME_DAYS": 20250,
        "TX_AMOUNT": 75.50,
        "TX_DURING_WEEKEND": 0,
        "TX_DURING_NIGHT": 0,
        "CUSTOMER_ID_NB_TX_1DAY_WINDOW": 2.0,
        "CUSTOMER_ID_AVG_AMOUNT_1DAY_WINDOW": 75.50,
        "CUSTOMER_ID_NB_TX_7DAY_WINDOW": 10.0,
        "CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW": 80.0,
        "CUSTOMER_ID_NB_TX_30DAY_WINDOW": 30.0,
        "CUSTOMER_ID_AVG_AMOUNT_30DAY_WINDOW": 85.0,
        "TERMINAL_ID_NB_TX_1DAY_WINDOW": 50.0,
        "TERMINAL_ID_RISK_1DAY_WINDOW": 0.1,
        "TERMINAL_ID_NB_TX_7DAY_WINDOW": 350.0,
        "TERMINAL_ID_RISK_7DAY_WINDOW": 0.15,
        "TERMINAL_ID_NB_TX_30DAY_WINDOW": 1500.0,
        "TERMINAL_ID_RISK_30DAY_WINDOW": 0.12,
    }


@pytest.fixture
def prediction_request(sample_legitimate_payload):
    from google.protobuf.json_format import ParseDict

    from fraud.v1.fraud_pb2 import PredictRequest

    return ParseDict({key.lower(): value for key, value in sample_legitimate_payload.items()}, PredictRequest())


@pytest.fixture
def rpc_server():
    """Real loopback gRPC transport with fixture-owned server, channel and loop."""
    import asyncio
    from contextlib import contextmanager
    from threading import Thread
    from types import SimpleNamespace

    import grpc
    import numpy as np
    from grpc_health.v1.health_pb2_grpc import HealthStub

    from app.config import Settings
    from app.main import FraudServer
    from fraud.v1.fraud_pb2_grpc import FraudServiceStub

    class Model:
        def predict_proba(self, _):
            return np.array([[0.2, 0.8]])

    @contextmanager
    def running(loader=lambda _: Model(), settings=None, tracing=None):
        loop = asyncio.new_event_loop()
        thread = Thread(target=loop.run_forever, daemon=True)
        thread.start()
        server = None

        async def start():
            nonlocal server
            server = FraudServer(settings or Settings(metrics_enabled=False, tracing_enabled=False), loader, tracing)
            port = await server.start("127.0.0.1:0")
            return port

        try:
            port = asyncio.run_coroutine_threadsafe(start(), loop).result(10)
            with grpc.insecure_channel(f"127.0.0.1:{port}") as channel:
                yield SimpleNamespace(
                    stub=FraudServiceStub(channel),
                    health=HealthStub(channel),
                    server=server,
                    stop=lambda grace: asyncio.run_coroutine_threadsafe(server.stop(grace), loop),
                )
        finally:
            if server is not None:
                asyncio.run_coroutine_threadsafe(server.stop(0), loop).result(10)
            loop.call_soon_threadsafe(loop.stop)
            thread.join(5)
            loop.close()

    return running
