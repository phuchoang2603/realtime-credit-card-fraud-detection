"""Exercise real gRPC process signals and optionally the Go HTTP -> Python RPC path."""

import argparse
import asyncio
import json
import os
import signal
import socket
import subprocess
import sys
import tempfile
import time
from pathlib import Path
from urllib.error import URLError
from urllib.request import Request, urlopen

import grpc
import numpy as np
from google.protobuf.json_format import ParseDict
from grpc_health.v1 import health_pb2, health_pb2_grpc

from fraud.v1 import fraud_pb2, fraud_pb2_grpc
from tools.test_client import generate_legitimate_data


def free_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def wait_ready(channel, process):
    health = health_pb2_grpc.HealthStub(channel)
    deadline = time.monotonic() + 15
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError(f"process exited during startup: {process.returncode}")
        try:
            response = health.Check(health_pb2.HealthCheckRequest(service="readiness"), timeout=0.2)
            if response.status == health_pb2.HealthCheckResponse.SERVING:
                return
        except grpc.RpcError:
            pass
        time.sleep(0.05)
    raise RuntimeError("readiness deadline exceeded")


def stop(process):
    if process.poll() is None:
        process.kill()
    process.wait(timeout=5)


def process_probe(delay):
    with tempfile.TemporaryDirectory() as directory:
        marker = Path(directory) / "entered"
        port = free_port()
        environment = os.environ | {"TESTING_MODE": "true", "GRPC_PORT": str(port), "GRACEFUL_SHUTDOWN_TIMEOUT": "1"}
        with (Path(directory) / "output").open("w+") as output:
            process = subprocess.Popen(
                [sys.executable, "-m", "tools.lifecycle_probe", "--child", str(delay), "--marker", str(marker)],
                env=environment,
                stdout=output,
                stderr=subprocess.STDOUT,
            )
            try:
                with grpc.insecure_channel(f"127.0.0.1:{port}") as channel:
                    wait_ready(channel, process)
                    request = ParseDict(
                        {k.lower(): v for k, v in generate_legitimate_data().items()}, fraud_pb2.PredictRequest()
                    )
                    call = fraud_pb2_grpc.FraudServiceStub(channel).Predict.future(request, timeout=10)
                    deadline = time.monotonic() + 3
                    while not marker.exists() and time.monotonic() < deadline:
                        time.sleep(0.01)
                    if not marker.exists():
                        raise RuntimeError("inference never started")
                    started = time.monotonic()
                    process.send_signal(signal.SIGTERM)
                    if delay < 1:
                        assert call.result(3).fraud_probability == 0.2
                    else:
                        try:
                            call.result(3)
                        except grpc.RpcError:
                            pass
                        else:
                            raise RuntimeError("overdue RPC was not aborted")
                    process.wait(timeout=8)
                    assert process.returncode == 0
                    assert time.monotonic() - started < 7
            except BaseException:
                output.seek(0)
                print(output.read(), file=sys.stderr)
                raise
            finally:
                stop(process)


def edge_probe(binary):
    port, edge_port = free_port(), free_port()
    environment = os.environ | {"TESTING_MODE": "true", "GRPC_PORT": str(port)}
    with tempfile.TemporaryFile(mode="w+") as output:
        fraud = subprocess.Popen(
            [sys.executable, "-m", "app"], env=environment, stdout=output, stderr=subprocess.STDOUT
        )
        edge = None
        try:
            with grpc.insecure_channel(f"127.0.0.1:{port}") as channel:
                wait_ready(channel, fraud)
                edge = subprocess.Popen(
                    [str(binary)],
                    env=os.environ
                    | {
                        "EDGE_HTTP_ADDR": f"127.0.0.1:{edge_port}",
                        "FRAUD_GRPC_TARGET": f"127.0.0.1:{port}",
                    },
                    stdout=output,
                    stderr=subprocess.STDOUT,
                )
                deadline = time.monotonic() + 5
                while True:
                    try:
                        with urlopen(f"http://127.0.0.1:{edge_port}/ready", timeout=0.2):
                            break
                    except URLError:
                        if time.monotonic() > deadline:
                            raise RuntimeError("edge never became ready")
                        time.sleep(0.05)
                payload = {key.lower(): value for key, value in generate_legitimate_data().items()}
                request = Request(
                    f"http://127.0.0.1:{edge_port}/predict",
                    data=json.dumps(payload).encode(),
                    headers={"Content-Type": "application/json", "X-Request-ID": "cross-language"},
                )
                with urlopen(request, timeout=5) as response:
                    result = json.load(response)
                    assert response.headers["X-Request-ID"] == "cross-language"
                direct = fraud_pb2_grpc.FraudServiceStub(channel).Predict(
                    ParseDict(payload, fraud_pb2.PredictRequest()), timeout=3
                )
                assert result == {"is_fraud": direct.is_fraud, "fraud_probability": direct.fraud_probability}
            for process in (edge, fraud):
                process.send_signal(signal.SIGTERM)
                process.wait(timeout=8)
                assert process.returncode == 0
        except BaseException:
            output.seek(0)
            print(output.read(), file=sys.stderr)
            raise
        finally:
            if edge:
                stop(edge)
            stop(fraud)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--child", type=float)
    parser.add_argument("--marker", type=Path)
    parser.add_argument("--edge-binary", type=Path)
    args = parser.parse_args()
    if args.child is not None:
        from app.__main__ import serve

        class Model:
            def predict_proba(self, _):
                args.marker.touch()
                time.sleep(args.child)
                return np.array([[0.8, 0.2]])

        asyncio.run(serve(model_loader=lambda _: Model()))
        return
    process_probe(0.3)
    process_probe(60)  # A hung native call must not hold interpreter shutdown forever.
    if args.edge_binary:
        edge_probe(args.edge_binary.resolve())
    print(
        "PASS: gRPC readiness, active drain, forced deadline exit"
        + (", Go-to-Python real-model prediction" if args.edge_binary else "")
    )


if __name__ == "__main__":
    main()
