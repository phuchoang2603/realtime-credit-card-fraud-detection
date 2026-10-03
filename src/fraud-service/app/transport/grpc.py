from __future__ import annotations

import time
from datetime import UTC, datetime

import grpc
from google.protobuf.timestamp_pb2 import Timestamp
from pydantic import ValidationError

from app.domain.policy import evaluate
from app.runtime import RuntimeState
from app.transport.mapping import decision_input_from_proto
from fraud.v2 import fraud_pb2, fraud_pb2_grpc


class FraudService(fraud_pb2_grpc.FraudServiceServicer):
    def __init__(self, state: RuntimeState):
        self.state = state

    async def Decide(self, request, context):
        if not self.state.ready:
            await context.abort(grpc.StatusCode.UNAVAILABLE, "Decision service unavailable")
        try:
            decision_input = decision_input_from_proto(request)
        except ValidationError, ValueError, OverflowError:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "Invalid or missing decision input")
        started = time.perf_counter()
        result = evaluate(decision_input)
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
